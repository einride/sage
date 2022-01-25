package mg

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/doc"
	"path/filepath"
	"strings"

	"github.com/iancoleman/strcase"
	"go.einride.tech/mage-tools/internal/codegen"
)

func generateMakefile(g *codegen.File, pkg *doc.Package, mk *Makefile, ns string) error {
	includePath, err := filepath.Rel(filepath.Dir(mk.Path), FromGitRoot(MageDir))
	if err != nil {
		return err
	}
	cmd := Command(context.Background(), "go", "list", "-m")
	var b bytes.Buffer
	cmd.Stdout = &b
	if err := cmd.Run(); err != nil {
		return err
	}
	mageDependencies := fmt.Sprintf("%s/go.mod %s/*.go", includePath, includePath)
	genCommand := fmt.Sprintf("cd %s && go run go.einride.tech/mage-tools/cmd/build", includePath)
	if strings.TrimSpace(b.String()) == "go.einride.tech/mage-tools" {
		mageDependencies = fmt.Sprintf("%s/go.mod $(shell find %s/.. -type f -name '*.go')", includePath, includePath)
		genCommand = fmt.Sprintf("cd %s && go run ../cmd/build", includePath)
	}

	g.P("# To learn more, see ", includePath, "/magefile.go and https://github.com/einride/mage-tools.")
	g.P()
	if len(mk.defaultTargetName()) != 0 {
		g.P(".DEFAULT_GOAL := ", toMakeTarget(mk.defaultTargetName()))
		g.P()
	}
	g.P("magefile := ", filepath.Join(includePath, ToolsDir, MagefileBinary))
	g.P()
	g.P("$(magefile): ", mageDependencies)
	g.P("\t@", genCommand)
	g.P()
	g.P(".PHONY: clean-mage-tools")
	g.P("clean-mage-tools:")
	g.P("\t@git clean -fdx ", filepath.Join(includePath, ToolsDir))
	forEachTargetFunction(pkg, func(function *doc.Func, namespace *doc.Type) bool {
		// TODO: Lets see if we can make this conditional better
		if function.Recv == ns || function.Recv == "" && ns == defaultNamespace {
			g.P()
			g.P(".PHONY: ", toMakeTarget(getTargetFunctionName(function)))
			g.P(toMakeTarget(getTargetFunctionName(function)), ": $(magefile)")
			args := toMakeVars(function.Decl.Type.Params.List[1:])
			if len(args) > 0 {
				for _, arg := range args {
					g.P("ifndef ", arg)
					g.P("\t $(error missing argument ", arg, `="...")`)
					g.P("endif")
				}
			}
			g.P(
				"\t@$(magefile) ",
				toMageTarget(getTargetFunctionName(function), args),
			)
		}
		return true
	})
	return nil
}

// toMakeVars converts input to make vars.
func toMakeVars(args []*ast.Field) []string {
	makeVars := make([]string, 0, len(args))
	for _, b := range args {
		for _, name := range b.Names {
			makeVars = append(makeVars, strcase.ToSnake(name.Name))
		}
	}
	return makeVars
}

// toMageTarget converts input to mageTarget with makeVars as arguments.
func toMageTarget(target string, args []string) string {
	for _, arg := range args {
		arg = fmt.Sprintf("$(%s)", arg)
		target += fmt.Sprintf(" %s", arg)
	}
	return target
}

// toMakeTarget converts input to make target format.
func toMakeTarget(str string) string {
	output := str
	if strings.Contains(str, ":") {
		output = strings.Split(str, ":")[1]
	}
	output = strcase.ToKebab(output)
	return strings.ToLower(output)
}
