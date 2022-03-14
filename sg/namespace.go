package sg

import (
	"go/ast"
	"go/doc"
	"hash/fnv"
	"strconv"
)

// Namespace allows for the grouping of similar commands.
type Namespace struct{}

func makefileNSPrefix(mk Makefile) string {
	h := fnv.New32a()
	h.Write([]byte(mk.Path))
	return "Target" + strconv.Itoa(int(h.Sum32()))
}

func isNamespace(t *doc.Type) bool {
	if len(t.Decl.Specs) != 1 {
		return false
	}
	typeSpec, ok := t.Decl.Specs[0].(*ast.TypeSpec)
	if !ok {
		return false
	}

	// Is it embedded in a struct?
	structType, ok := typeSpec.Type.(*ast.StructType)
	if ok {
		for _, f := range structType.Fields.List {
			// Is it an embedded namespace
			if f.Names == nil && isSelectorNamespaceType(f.Type) {
				return true
			}
		}
	}

	return isSelectorNamespaceType(typeSpec.Type)
}

func isSelectorNamespaceType(expr ast.Expr) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "sg" && selector.Sel.Name == "Namespace"
}

func namespaceTargets(pkg *doc.Package, mk Makefile) []*doc.Func {
	var fns []*doc.Func
	// If no namespace is given then all package functions are elgible
	// for target
	if mk.namespaceName() == "" {
		for _, fn := range pkg.Funcs {
			if !isValidFunction(fn) {
				continue
			}
			fns = append(fns, fn)
		}
	} else {
		for _, ns := range pkg.Types {
			ns := ns
			if !isValidNamespace(ns) || ns.Name != mk.namespaceName() {
				continue
			}
			for _, fn := range ns.Methods {
				if !isValidFunction(fn) {
					continue
				}
				fns = append(fns, fn)
			}
		}
	}
	return fns
}

func isValidNamespace(ns *doc.Type) bool {
	return ast.IsExported(ns.Name) && isNamespace(ns)
}

func isValidFunction(fn *doc.Func) bool {
	return ast.IsExported(fn.Name) && isSupportedTargetFunctionParams(fn.Decl.Type.Params.List)
}
