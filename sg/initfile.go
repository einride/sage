//nolint:lll
package sg

import (
	"fmt"
	"go/ast"
	"go/doc"
	"reflect"
	"strconv"
	"strings"

	"go.einride.tech/sage/internal/codegen"
)

const (
	stringType = "string"
	intType    = "int"
	boolType   = "bool"
)

func generateInitFile(g *codegen.File, pkg *doc.Package, mks []Makefile) error {
	g.P("func init() {")
	g.P("ctx := ", g.Import("context"), ".Background()")
	g.P("if len(", g.Import("os"), ".Args) < 2 {")
	g.P(g.Import("fmt"), `.Println("Targets:")`)
	forEachTargetFunction(pkg, func(function *doc.Func, namespace *doc.Type) {
		// If function namespace is not part of the to be generated Makefiles, skip it.
		if skipFunction, _ := shouldBeGenerated(mks, function.Recv); !skipFunction {
			return
		}
		g.P(g.Import("fmt"), `.Println("\t`, getTargetFunctionName(function), `")`)
	})
	g.P(g.Import("os"), ".Exit(0)")
	g.P("}")
	g.P("target, args := ", g.Import("os"), ".Args[1], ", g.Import("os"), ".Args[2:]")
	g.P("_ = args")
	g.P("var err error")
	g.P("switch target {")
	forEachTargetFunction(pkg, func(function *doc.Func, namespace *doc.Type) {
		// If function namespace is not part of the to be generated Makefiles, skip it.
		skipFunction, nsStruct := shouldBeGenerated(mks, function.Recv)
		if !skipFunction {
			return
		}
		g.P(`case "`, getTargetFunctionName(function), `":`)
		loggerName := getTargetFunctionName(function)
		// Remove namespace from loggerName
		if strings.Contains(loggerName, ":") {
			loggerName = strings.Split(loggerName, ":")[1]
		}
		g.P("before := ", g.Import("time"), ".Now()")
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
				strings.ReplaceAll(getTargetFunctionName(function), ":", nsStruct),
				"(ctx,",
				strings.Join(args, ","),
				")",
			)
			g.P("if err != nil {")
			g.P(`if os.Getenv("SAGE_TRACE") == "true" {println("`, getTargetFunctionName(function), ` took", `, g.Import("time"), `.Since(before).String(), "to error")}`)
			g.P("logger.Fatal(err)")
			g.P("}")
		} else {
			g.P("err = ", strings.ReplaceAll(getTargetFunctionName(function), ":", nsStruct), "(ctx)")
			g.P("if err != nil {")
			g.P(`if os.Getenv("SAGE_TRACE") == "true" {println("`, getTargetFunctionName(function), ` took", `, g.Import("time"), `.Since(before).String(), "to error")}`)
			g.P("logger.Fatal(err)")
			g.P("}")
		}
		g.P(`if os.Getenv("SAGE_TRACE") == "true" {println("`, getTargetFunctionName(function), ` took", `, g.Import("time"), ".Since(before).String())}")
	})
	g.P("default:")
	g.P("logger := ", g.Import("go.einride.tech/sage/sg"), ".NewLogger(\"sagefile\")")
	g.P(`logger.Fatalf("unknown target specified: %s", target)`)
	g.P("}")
	g.P(g.Import("os"), ".Exit(0)")
	g.P("}")
	return nil
}

// shouldBeGenerated returns true if the namespace equals any of the namespaces in the to be generated Makefiles and
// returns any metadata the namespace might have.
func shouldBeGenerated(mks []Makefile, namespace string) (bool, string) {
	var partOfMakefile bool
	namespaceStruct := "{"
	for _, mk := range mks {
		if mk.namespaceName() == namespace {
			partOfMakefile = true
			val := reflect.Indirect(reflect.ValueOf(mk.Namespace))
			if !reflect.Value.IsValid(val) {
				continue
			}
			for i := 1; i < val.NumField(); i++ {
				namespaceStruct = fmt.Sprintf("%s\n%s:", namespaceStruct, val.Type().Field(i).Name)
				fieldValue := reflect.ValueOf(val.Field(i).Interface())
				switch val.Field(i).Kind() {
				case reflect.String:
					namespaceStruct = fmt.Sprintf(`%s "%v",`, namespaceStruct, fieldValue)
				case reflect.Bool, reflect.Int:
					namespaceStruct = fmt.Sprintf("%s %v,", namespaceStruct, fieldValue)
				default:
					panic("unsupported type for namespace field")
				}
			}
		}
	}
	namespaceStruct += "}."
	return partOfMakefile, namespaceStruct
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

func forEachTargetFunction(pkg *doc.Package, fn func(function *doc.Func, namespace *doc.Type)) {
	for _, function := range pkg.Funcs {
		if function.Recv != "" ||
			!ast.IsExported(function.Name) ||
			!isSupportedTargetFunctionParams(function.Decl.Type.Params.List) {
			continue
		}
		fn(function, nil)
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
			fn(function, nil)
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
