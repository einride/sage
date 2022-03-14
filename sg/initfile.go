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
	g.P(`panic("incorrect number of arguments")`)
	g.P("}")
	g.P("target, args := ", g.Import("os"), ".Args[1] ,", g.Import("os"), ".Args[2:]")
	g.P("parts := ", g.Import("strings"), `.SplitN(target, ":", 2)`)
	g.P("if len(parts) < 2 {")
	g.P(`panic("invalid Makefile structure. This can happen if sage has been recently upgraded and the internal structure of the ` +
		`generated go file has changed. Try running again. If this persists please report it as a bug.")`)
	g.P("}")
	g.P("switch parts[0] {")
	for _, mk := range mks {
		g.P(`case "`, makefileNSPrefix(mk), `":`)
		g.P("if err := ", makefileNSPrefix(mk), "(ctx, parts[1],args); err != nil {")
		g.P("panic(err)")
		g.P("}")
	}
	g.P("}")
	g.P(g.Import("os"), ".Exit(0)")
	g.P("}")
	for _, mk := range mks {
		targets := namespaceTargets(pkg, mk)
		g.P("func ", makefileNSPrefix(mk), "(ctx ", g.Import("context"), ".Context, target string, args []string) error {")
		g.P("if len(target) == 0 {")
		g.P(g.Import("fmt"), `.Println("Targets:")`)
		for _, target := range targets {
			g.P(g.Import("fmt"), `.Println("`, getTargetFunctionName(target), `")`)
		}
		g.P("return nil")
		g.P("}")
		g.P("var err error")
		g.P("switch target {")
		for _, target := range targets {
			g.P(`case "`, getTargetFunctionName(target), `":`)
			generateTargetCase(g, mk, target)
		}
		g.P("}")
		g.P("return err")
		g.P("}")
	}
	return nil
}

func generateTargetCase(g *codegen.File, mk Makefile, target *doc.Func) {
	loggerName := getTargetFunctionName(target)
	g.P("logger := ", g.Import("go.einride.tech/sage/sg"), ".NewLogger(\"", loggerName, "\")")
	g.P("ctx = ", g.Import("go.einride.tech/sage/sg"), ".WithLogger(ctx, logger)")
	if len(target.Decl.Type.Params.List) > 1 {
		expected := countParams(target.Decl.Type.Params.List) - 1
		g.P("if len(args) != ", expected, " {")
		g.P(
			`logger.Fatalf("wrong number of arguments to %s, got %v expected %v",`,
			strconv.Quote(target.Name), ",",
			expected, ",",
			`len(args))`,
		)
		g.P("}")
		var args []string
		var i int
		for _, customParam := range target.Decl.Type.Params.List[1:] {
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
			generateNSTarget(mk, target),
			"(ctx,",
			strings.Join(args, ","),
			")",
		)
		g.P("if err != nil {")
		g.P("logger.Fatal(err)")
		g.P("}")
	} else {
		g.P("err = ", generateNSTarget(mk, target), "(ctx)")
		g.P("if err != nil {")
		g.P("logger.Fatal(err)")
		g.P("}")
	}
}

// generateNSTarget generates a target call in the form of <namespace>.<method>
// for Makefile with namespace and <method> otherwise.
func generateNSTarget(mk Makefile, target *doc.Func) string {
	var builder strings.Builder
	writeString := func(s string) {
		_, err := builder.WriteString(s)
		if err != nil {
			panic(err)
		}
	}

	if mk.Namespace == nil {
		return target.Name
	}

	if target.Recv != "" {
		writeString(target.Recv)
		writeString("{")
	}
	val := reflect.Indirect(reflect.ValueOf(mk.Namespace))
	if !reflect.Value.IsValid(val) {
		panic(fmt.Errorf("invalid namespace: %+v", val))
	}

	for i := 0; i < val.NumField(); i++ {
		// If we find an embedded field we skip it
		if val.Field(i).Kind() == reflect.Struct && reflect.TypeOf(mk.Namespace).Field(i).Anonymous {
			continue
		}

		writeString(fmt.Sprintf("%s: ", val.Type().Field(i).Name))
		fieldValue := reflect.ValueOf(val.Field(i).Interface())
		switch val.Field(i).Kind() {
		case reflect.String:
			writeString(fmt.Sprintf(`"%v"`, fieldValue))
		case reflect.Bool, reflect.Int:
			writeString(fmt.Sprintf("%v", fieldValue))
		default:
			panic("unsupported type for namespace field")
		}

		if i < val.NumField()-1 {
			writeString(",")
		}
	}
	writeString("}.")
	writeString(target.Name)
	return builder.String()
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

func getNamespaceFunctionName(mk Makefile, function *doc.Func) string {
	prefix := makefileNSPrefix(mk)
	return prefix + ":" + getTargetFunctionName(function)
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
