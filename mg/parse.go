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

// Package compiles information about a mage package.
func Package(path string) (*PkgInfo, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, path, func(info fs.FileInfo) bool {
		return info.Name() != MakeGenGo
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
