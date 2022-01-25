package mg

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io/fs"
	"sort"
	"strings"
)

// PkgInfo contains inforamtion about a package of files according to mage's
// parsing rules.
type PkgInfo struct {
	DocPkg *doc.Package
	Funcs  Functions
}

// Function represented a job function from a mage file.
type Function struct {
	Name     string
	Receiver string
	Args     []Arg
}

var _ sort.Interface = (Functions)(nil)

// Functions implements sort interface to optimize compiled output with
// deterministic generated mainfile.
type Functions []*Function

func (s Functions) Len() int {
	return len(s)
}

func (s Functions) Less(i, j int) bool {
	return s[i].TargetName() < s[j].TargetName()
}

func (s Functions) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Arg is an argument to a Function.
type Arg struct {
	Name, Type string
}

// TargetName returns the name of the target as it should appear when used from
// the mage cli.  It is always lowercase.
func (f Function) TargetName() string {
	var names []string

	for _, s := range []string{f.Receiver, f.Name} {
		if s != "" {
			names = append(names, s)
		}
	}
	return strings.Join(names, ":")
}

// ExecCode returns code for the template switch to run the target.
// It wraps each target call to match the func(context.Context) error that
// runTarget requires.
func (f Function) ExecCode() string {
	name := f.Name
	if f.Receiver != "" {
		name = f.Receiver + "{}." + name
	}

	var parseargs string
	for x, arg := range f.Args {
		switch arg.Type {
		case "string":
			parseargs += fmt.Sprintf(`
			arg%d := args[x]
			x++`+"\n", x)
		case "int":
			parseargs += fmt.Sprintf(`
				arg%d, err := strconv.Atoi(args[x])
				if err != nil {
					logger.Error(err, "can't convert argument %%q to int\n", args[x])
					os.Exit(1)
				}
				x++`, x)
		case "bool":
			parseargs += fmt.Sprintf(`
				arg%d, err := strconv.ParseBool(args[x])
				if err != nil {
					logger.Error(err, "convert argument %%q to bool\n", args[x])
					os.Exit(1)
				}
				x++`, x)
		}
	}

	out := parseargs
	out += "if err := " + name + "("
	args := make([]string, 0, len(f.Args))
	args = append(args, "ctx")
	for x := 0; x < len(f.Args); x++ {
		args = append(args, fmt.Sprintf("arg%d", x))
	}
	out += strings.Join(args, ", ")
	out += `); err != nil {
				logger.Error(err, err.Error())
				os.Exit(1)	
			}`
	return out
}

// Package compiles information about a mage package.
func Package(path string) (*PkgInfo, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, path, func(info fs.FileInfo) bool {
		return info.Name() != "mgmake_gen.go"
	}, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse directory: %v", err)
	}
	var pkg *ast.Package
	switch len(pkgs) {
	case 1:
		for _, p := range pkgs {
			pkg = p
		}
	case 0:
		return nil, fmt.Errorf("no importable packages found in %s", path)
	default:
		var names []string
		for name := range pkgs {
			names = append(names, name)
		}
		return nil, fmt.Errorf("multiple packages found in %s: %v", path, strings.Join(names, ", "))
	}
	p := doc.New(pkg, "./", 0)
	pi := &PkgInfo{
		DocPkg: p,
	}
	setNamespaces(pi)
	setFuncs(pi)
	return pi, nil
}

func setFuncs(pi *PkgInfo) {
	for _, f := range pi.DocPkg.Funcs {
		if f.Recv != "" {
			// skip methods
			continue
		}
		if !ast.IsExported(f.Name) || !hasContextParam(f.Decl.Type) || !hasErrorReturn(f.Decl.Type) {
			continue
		}
		fn, err := funcType(f.Decl.Type)
		if err != nil {
			continue
		}
		fn.Name = f.Name
		pi.Funcs = append(pi.Funcs, fn)
	}
}

func setNamespaces(pi *PkgInfo) {
	for _, t := range pi.DocPkg.Types {
		if !isNamespace(t) {
			continue
		}
		for _, f := range t.Methods {
			if !ast.IsExported(f.Name) || !hasContextParam(f.Decl.Type) || !hasErrorReturn(f.Decl.Type) {
				continue
			}
			fn, err := funcType(f.Decl.Type)
			if err != nil {
				continue
			}
			fn.Name = f.Name
			fn.Receiver = t.Name
			pi.Funcs = append(pi.Funcs, fn)
		}
	}
}

func isNamespace(t *doc.Type) bool {
	if len(t.Decl.Specs) != 1 {
		return false
	}
	id, ok := t.Decl.Specs[0].(*ast.TypeSpec)
	if !ok {
		return false
	}
	sel, ok := id.Type.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "mg" && sel.Sel.Name == "Namespace"
}

func hasContextParam(ft *ast.FuncType) bool {
	if ft.Params.NumFields() == 0 {
		return false
	}
	param := ft.Params.List[0]
	sel, ok := param.Type.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	if pkg.Name != "context" {
		return false
	}
	if sel.Sel.Name != "Context" {
		return false
	}
	return true
}

func hasErrorReturn(ft *ast.FuncType) bool {
	res := ft.Results
	if res.NumFields() != 1 {
		return false
	}
	ret := res.List[0]
	if len(ret.Names) > 1 {
		return false
	}
	return fmt.Sprint(ret.Type) == "error"
}

func funcType(ft *ast.FuncType) (*Function, error) {
	f := &Function{}
	x := 1
	for ; x < len(ft.Params.List); x++ {
		param := ft.Params.List[x]
		t := fmt.Sprint(param.Type)
		switch t {
		case "string", "int", "bool":
			// ok
		default:
			return nil, fmt.Errorf("unsupported argument type: %s", t)
		}
		// support for foo, bar string
		for _, name := range param.Names {
			f.Args = append(f.Args, Arg{Name: name.Name, Type: t})
		}
	}
	return f, nil
}

func isSupportedTargetParams(params []*ast.Field) bool {
	if len(params) == 0 {
		return false
	}
	if !isContextParam(params[0]) {
		return false
	}
	for _, customParam := range params[1:] {
		if !isSupportedCustomParam(customParam) {
			return false
		}
	}
	return true
}

func isSupportedCustomParam(arg *ast.Field) bool {
	switch fmt.Sprint(arg.Type) {
	case "string", "int", "bool":
		return true
	default:
		return false
	}
}

func isContextParam(param *ast.Field) bool {
	selectorExpr, ok := param.Type.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := selectorExpr.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "context" && selectorExpr.Sel.Name == "Context"
}

func isContextType(ft *ast.FuncType) bool {
	if ft.Params.NumFields() == 0 {
		return false
	}
	param := ft.Params.List[0]
	sel, ok := param.Type.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	if pkg.Name != "context" {
		return false
	}
	if sel.Sel.Name != "Context" {
		return false
	}
	return true
}
