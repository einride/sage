package mg

import (
	"fmt"
	"go/ast"
	"go/doc"
	"strings"

	"go.einride.tech/mage-tools/internal/codegen"
)

func generateMainFile(g *codegen.File, pkg *doc.Package) error {
	g.P("func main() {")
	g.P("ctx := ", g.Import("context"), ".Background()")
	g.P("if len(", g.Import("os"), ".Args) < 2 {")
	g.P(g.Import("fmt"), `.Println("Targets:")`)
	forEachTargetFunction(pkg, func(function *doc.Func, namespace *doc.Type) bool {
		g.P(g.Import("fmt"), `.Println("\t`, getTargetFunctionName(function, namespace), `")`)
		return true
	})
	g.P("return")
	g.P("}")
	g.P("target, args := ", g.Import("os"), ".Args[1], ", g.Import("os"), ".Args[2:]")
	g.P("switch target {")
	forEachTargetFunction(pkg, func(function *doc.Func, namespace *doc.Type) bool {
		g.P(`case "`, getTargetFunctionName(function, namespace), `":`)
		g.P(
			"ctx = ",
			g.Import("github.com/go-logr/logr"),
			".NewContext(ctx, ",
			g.Import("go.einride.tech/mage-tools/mglogr"),
			`.New("`,
			getTargetFunctionName(function, namespace),
			`"))`,
		)
		g.P("logger := logr.FromContextOrDiscard(ctx)")
		if len(function.Decl.Type.Params.List) > 1 {
			expected := len(function.Decl.Type.Params.List)
			g.P("if len(args) != ", expected, " {")
			g.P(
				"logger.Info(",
				`"wrong number of arguments",`,
				`"target", `,
				getTargetFunctionName(function, namespace),
				",",
				`"expected",`,
				expected,
				`, "got", len(args))`,
			)
			g.P(g.Import("os"), ".Exit(1)")
			g.P("}")
			var args []string
			var i int
			for _, customParam := range function.Decl.Type.Params.List[1:] {
				for range customParam.Names {
					args = append(args, fmt.Sprintf("arg%v", i))
					switch fmt.Sprint(customParam.Type) {
					case "string":
						g.P("arg", i, " := args[", i, "]")
					case "int":
						g.P("arg", i, ", err :=", g.Import("strconv"), ".Atoi(args[", i, "])")
						g.P("if err != nil {")
						g.P(`logger.Error(err, "can't convert argument %q to int\n", args[`, i, `])`)
						g.P(g.Import("os"), ".Exit(1)")
						g.P("}")
					case "bool":
						g.P("arg", i, ", err :=", g.Import("strconv"), ".ParseBool(args[", i, "])")
						g.P("if err != nil {")
						g.P(`logger.Error(err, "can't convert argument %q to bool\n", args[`, i, `])`)
						g.P(g.Import("os"), ".Exit(1)")
						g.P("}")
					}
					i++
				}
			}
			g.P(
				"if err := ",
				getTargetFunctionName(function, namespace),
				"(ctx,",
				strings.Join(args, ","),
				"); err != nil {",
			)
			g.P("logger.Error(err, err.Error())")
			g.P(g.Import("os"), ".Exit(1)")
			g.P("}")
		} else {
			g.P("if err := ", getTargetFunctionName(function, namespace), "(ctx); err != nil {")
			g.P("logger.Error(err, err.Error())")
			g.P(g.Import("os"), ".Exit(1)")
			g.P("}")
		}
		return true
	})
	g.P("default:")
	g.P(
		"ctx = ",
		g.Import("github.com/go-logr/logr"),
		".NewContext(ctx, ",
		g.Import("go.einride.tech/mage-tools/mglogr"),
		`.New("magefile"))`,
	)
	g.P("logger := logr.FromContextOrDiscard(ctx)")
	g.P(`logger.Info("Unknown target specified: %q\n", target)`)
	g.P(g.Import("os"), ".Exit(1)")
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
		if !ast.IsExported(namespace.Name) || !isNamespace2(namespace) {
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

func isNamespace2(t *doc.Type) bool {
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
