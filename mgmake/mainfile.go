package mgmake

import (
	"fmt"
	"go.einride.tech/mage-tools/internal/codegen"
	"go/ast"
	"go/doc"
	"strings"
)

func generateMainFile2(pkg *doc.Package, g *codegen.File) error {
	g.P("func main() {")
	g.P("args := ", g.Import("os"), ".Args[1:]")
	g.P("ctx := ", g.Import("context"), ".Background()")
	g.P("_ = ctx")
	g.P("if len(args) == 0 {")
	g.P(g.Import("fmt"), `.Println("Targets:")`)
	forEachTargetFunction(pkg, func(function *doc.Func, namespace *doc.Type) bool {
		g.P(g.Import("fmt"), `.Println("\t`, getTargetFunctionName(function, namespace), `")`)
		return true
	})
	g.P("}")
	g.P("}")
	return nil
}

func getTargetFunctionName(function *doc.Func, namespace *doc.Type) string {
	var result strings.Builder
	if namespace != nil {
		_, _ = result.WriteString(namespace.Name)
		_ = result.WriteByte(':')
	}
	_, _ = result.WriteString(function.Name)
	return result.String()
}

func forEachTargetFunction(pkg *doc.Package, fn func(function *doc.Func, namespace *doc.Type) bool) {
	for _, function := range pkg.Funcs {
		if function.Recv != "" ||
			!ast.IsExported(function.Name) ||
			!isSupportedTargetFunctionParams(function.Decl.Type.Params.List) {
			continue
		}
		if !fn(function, nil) {
			return
		}
	}
	for _, namespace := range pkg.Types {
		if !ast.IsExported(namespace.Name) || !isNamespace(namespace) {
			continue
		}
		for _, function := range namespace.Methods {
			if function.Recv != "" ||
				!ast.IsExported(function.Name) ||
				!isSupportedTargetFunctionParams(function.Decl.Type.Params.List) {
				continue
			}
			if !fn(function, nil) {
				return
			}
		}
	}
}

func isSupportedTargetFunctionParams(params []*ast.Field) bool {
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

func isNamespace(t *doc.Type) bool {
	if len(t.Decl.Specs) != 1 {
		return false
	}
	typeSpec, ok := t.Decl.Specs[0].(*ast.TypeSpec)
	if !ok {
		return false
	}
	selectorExpr, ok := typeSpec.Type.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := selectorExpr.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "mg" && selectorExpr.Sel.Name == "Namespace"
}
