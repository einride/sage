package sg

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/doc"
	"path/filepath"
	"strings"

	"github.com/iancoleman/strcase"
	"go.einride.tech/sage/internal/codegen"
)

func generateMakefile(g *codegen.File, pkg *doc.Package, mk Makefile, mks ...Makefile) error {
	includePath, err := filepath.Rel(filepath.Dir(mk.Path), FromGitRoot(SageDir))
	if err != nil {
		return err
	}
	cmd := Command(context.Background(), "go", "list", "-m")
	var b bytes.Buffer
	cmd.Stdout = &b
	if err := cmd.Run(); err != nil {
		return err
	}
	dependencies := fmt.Sprintf("%s/go.mod %s/*.go", includePath, includePath)
	genCommand := fmt.Sprintf("cd %s && go run go.einride.tech/sage run", includePath)
	if strings.TrimSpace(b.String()) == "go.einride.tech/sage" {
		dependencies = fmt.Sprintf("%s/go.mod $(shell find %s/.. -type f -name '*.go')", includePath, includePath)
		genCommand = fmt.Sprintf("cd %s && go run ../main.go run", includePath)
	}

	g.P("# To learn more, see ", includePath, "/sagefile.go and https://github.com/einride/sage.")
	g.P()
	if len(mk.defaultTargetName()) != 0 {
		g.P(".DEFAULT_GOAL := ", toMakeTarget(mk.defaultTargetName()))
		g.P()
	}
	g.P("sagefile := ", filepath.Join(includePath, ToolsDir, SageFileBinary))
	g.P()
	g.P("$(sagefile): ", dependencies)
	g.P("\t@", genCommand)
	g.P()
	g.P(".PHONY: clean-sage")
	g.P("clean-sage:")
	g.P("\t@git clean -fdx ", filepath.Join(includePath, ToolsDir))
	forEachTargetFunction(pkg, func(function *doc.Func, namespace *doc.Type) bool {
		if function.Recv == mk.namespaceName() {
			g.P()
			g.P(".PHONY: ", toMakeTarget(getTargetFunctionName(function)))
			g.P(toMakeTarget(getTargetFunctionName(function)), ": $(sagefile)")
			args := toMakeVars(function.Decl.Type.Params.List[1:])
			if len(args) > 0 {
				for _, arg := range args {
					g.P("ifndef ", arg)
					g.P("\t $(error missing argument ", arg, `="...")`)
					g.P("endif")
				}
			}
			g.P(
				"\t@$(sagefile) ",
				toSageFunction(getTargetFunctionName(function), args),
			)
		}
		return true
	})
	// Add additional makefiles to default makefile
	if mk.namespaceName() == "" {
		for _, i := range mks {
			if i.namespaceName() == "" {
				continue
			}
			mkPath, err := filepath.Rel(FromGitRoot(), filepath.Dir(i.Path))
			if err != nil {
				panic(err)
			}
			g.P()
			g.P(".PHONY: ", toMakeTarget(i.namespaceName()))
			g.P(toMakeTarget(i.namespaceName()), ":")
			g.P("\tmake -C ", mkPath, " -f ", filepath.Base(i.Path))
			g.P()
		}
	}
	g.P()
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

// toSageFunction converts input to a sage Function name with the provided args.
func toSageFunction(target string, args []string) string {
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
