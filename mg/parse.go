package mg

import (
	"errors"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
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
	Name      string
	Receiver  string
	IsError   bool
	IsContext bool
	Args      []Arg
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
			x++`, x)
		case "int":
			parseargs += fmt.Sprintf(`
				arg%d, err := strconv.Atoi(args[x])
				if err != nil {
					logger.Printf("can't convert argument %%q to int\n", args[x])
					os.Exit(2)
				}
				x++`, x)
		case "bool":
			parseargs += fmt.Sprintf(`
				arg%d, err := strconv.ParseBool(args[x])
				if err != nil {
					logger.Printf("can't convert argument %%q to bool\n", args[x])
					os.Exit(2)
				}
				x++`, x)
		}
	}

	out := parseargs + `
				wrapFn := func(ctx context.Context) error {
					`
	if f.IsError {
		out += "return "
	}
	out += name + "("
	args := make([]string, 0, len(f.Args))
	if f.IsContext {
		args = append(args, "ctx")
	}
	for x := 0; x < len(f.Args); x++ {
		args = append(args, fmt.Sprintf("arg%d", x))
	}
	out += strings.Join(args, ", ")
	out += ")"
	if !f.IsError {
		out += `
					return nil`
	}
	out += `
				}
				ret := runTarget(wrapFn)`
	return out
}

// Package compiles information about a mage package.
func Package(path string, files []string) (*PkgInfo, error) {
	fset := token.NewFileSet()
	pkg, err := getPackage(path, files, fset)
	if err != nil {
		return nil, err
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
		if !ast.IsExported(f.Name) {
			// skip non-exported functions
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
			if !ast.IsExported(f.Name) {
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

// getPackage returns the importable package at the given path.
func getPackage(path string, files []string, fset *token.FileSet) (*ast.Package, error) {
	var filter func(f os.FileInfo) bool
	if len(files) > 0 {
		fm := make(map[string]bool, len(files))
		for _, f := range files {
			fm[f] = true
		}

		filter = func(f os.FileInfo) bool {
			return fm[f.Name()]
		}
	}

	pkgs, err := parser.ParseDir(fset, path, filter, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse directory: %v", err)
	}

	switch len(pkgs) {
	case 1:
		var pkg *ast.Package
		for _, pkg = range pkgs {
		}
		return pkg, nil
	case 0:
		return nil, fmt.Errorf("no importable packages found in %s", path)
	default:
		var names []string
		for name := range pkgs {
			names = append(names, name)
		}
		return nil, fmt.Errorf("multiple packages found in %s: %v", path, strings.Join(names, ", "))
	}
}

// hasContextParams returns whether or not the first parameter is a context.Context. If it
// determines that hte first parameter makes this function invalid for mage, it'll return a non-nil
// error.
func hasContextParam(ft *ast.FuncType) (bool, error) {
	if ft.Params.NumFields() < 1 {
		return false, nil
	}
	param := ft.Params.List[0]
	sel, ok := param.Type.(*ast.SelectorExpr)
	if !ok {
		return false, nil
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return false, nil
	}
	if pkg.Name != "context" {
		return false, nil
	}
	if sel.Sel.Name != "Context" {
		return false, nil
	}
	if len(param.Names) > 1 {
		// something like foo, bar context.Context
		return false, errors.New("ETOOMANYCONTEXTS")
	}
	return true, nil
}

func hasErrorReturn(ft *ast.FuncType) (bool, error) {
	res := ft.Results
	if res.NumFields() == 0 {
		// void return is ok
		return false, nil
	}
	if res.NumFields() > 1 {
		return false, errors.New("ETOOMANYRETURNS")
	}
	ret := res.List[0]
	if len(ret.Names) > 1 {
		return false, errors.New("ETOOMANYERRORS")
	}
	if fmt.Sprint(ret.Type) == "error" {
		return true, nil
	}
	return false, errors.New("EBADRETURNTYPE")
}

func funcType(ft *ast.FuncType) (*Function, error) {
	var err error
	f := &Function{}
	f.IsContext, err = hasContextParam(ft)
	if err != nil {
		return nil, err
	}
	f.IsError, err = hasErrorReturn(ft)
	if err != nil {
		return nil, err
	}
	x := 0
	if f.IsContext {
		x++
	}
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
