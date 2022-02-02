package sg

import (
	"fmt"
	"go/ast"
	"go/doc"
	"strconv"
	"strings"

	"go.einride.tech/sage/internal/codegen"
)

const (
	stringType = "string"
	intType    = "int"
	boolType   = "bool"
)

func generateInitFile(g *codegen.File, pkg *doc.Package) error {
	g.P("func init() {")
	g.P("ctx := ", g.Import("context"), ".Background()")
	g.P("if len(", g.Import("os"), ".Args) < 2 {")
	g.P(g.Import("fmt"), `.Println("Targets:")`)
	forEachTargetFunction(pkg, func(function *doc.Func, namespace *doc.Type) bool {
		g.P(g.Import("fmt"), `.Println("\t`, getTargetFunctionName(function), `")`)
		return true
	})
	g.P(g.Import("os"), ".Exit(0)")
	g.P("}")
	g.P("target, args := ", g.Import("os"), ".Args[1], ", g.Import("os"), ".Args[2:]")
	g.P("_ = args")
	g.P("var err error")
	g.P("switch target {")
	forEachTargetFunction(pkg, func(function *doc.Func, namespace *doc.Type) bool {
		g.P(`case "`, getTargetFunctionName(function), `":`)
		loggerName := getTargetFunctionName(function)
		// Remove namespace from loggerName
		if strings.Contains(loggerName, ":") {
			loggerName = strings.Split(loggerName, ":")[1]
		}
		g.P("logger := ", g.Import("go.einride.tech/sage/sg"), ".NewLogger(\"", loggerName, "\")")
		g.P("ctx = ", g.Import("go.einride.tech/sage/sg"), ".WithLogger(ctx, logger)")
		if len(function.Decl.Type.Params.List) > 1 {
			expected := countParams(function.Decl.Type.Params.List) - 1
			g.P("if len(args) != ", expected, " {")
			g.P(
				`logger.Fatalf("wrong number of arguments to %s, got %v expected %v",`,
				strconv.Quote(getTargetFunctionName(function)), ",",
				expected, ",",
				`len(args))`,
			)
			g.P(g.Import("os"), ".Exit(1)")
			g.P("}")
			var args []string
			var i int
			for _, customParam := range function.Decl.Type.Params.List[1:] {
				for range customParam.Names {
					args = append(args, fmt.Sprintf("arg%v", i))
					switch fmt.Sprint(customParam.Type) {
					case stringType:
						g.P("arg", i, " := args[", i, "]")
					case intType:
						g.P("arg", i, ", err :=", g.Import("strconv"), ".Atoi(args[", i, "])")
						g.P("if err != nil {")
						g.P(`logger.Fatalf("can't convert argument %q to int", args[`, i, `])`)
						g.P("}")
					case boolType:
						g.P("arg", i, ", err :=", g.Import("strconv"), ".ParseBool(args[", i, "])")
						g.P("if err != nil {")
						g.P(`logger.Fatalf("can't convert argument to bool", args[`, i, `])`)
						g.P("}")
					}
					i++
				}
			}
			g.P(
				"err = ",
				strings.ReplaceAll(getTargetFunctionName(function), ":", "{}."),
				"(ctx,",
				strings.Join(args, ","),
				")",
			)
			g.P("if err != nil {")
			g.P("logger.Fatal(err)")
			g.P("}")
		} else {
			g.P("err = ", strings.ReplaceAll(getTargetFunctionName(function), ":", "{}."), "(ctx)")
			g.P("if err != nil {")
			g.P("logger.Fatal(err)")
			g.P("}")
		}
		return true
	})
	g.P("default:")
	g.P("logger := ", g.Import("go.einride.tech/sage/sg"), ".NewLogger(\"sagefile\")")
	g.P(`logger.Fatalf("unknown target specified: %s", target)`)
	g.P("}")
	g.P(g.Import("os"), ".Exit(0)")
	g.P("}")
	return nil
}

func countParams(fields []*ast.Field) int {
	var result int
	for _, field := range fields {
		result += len(field.Names)
	}
	return result
}

func getTargetFunctionName(function *doc.Func) string {
	var result strings.Builder
	if function.Recv != "" {
		_, _ = result.WriteString(function.Recv)
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
			if !ast.IsExported(function.Name) ||
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
	case stringType, intType, boolType:
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
	return ident.Name == "sg" && selectorExpr.Sel.Name == "Namespace"
}
