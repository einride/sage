package sg

import (
	"context"
	"fmt"
	"go/ast"
	"go/doc"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"unicode"

	"go.einride.tech/sage/internal/codegen"
	"go.einride.tech/sage/internal/strcase"
)

// defaultGoVersion follows Einride's N-1 Go version policy.
// Renovate is configured to only propose patch updates.
// renovate: datasource=golang-version depName=golang-patches-only
const defaultGoVersion = "1.25.7"

type Makefile struct {
	Namespace     any
	Path          string
	DefaultTarget any
}

func (m Makefile) namespaceName() string {
	if m.Namespace == nil {
		return ""
	}
	return reflect.TypeOf(m.Namespace).Name()
}

func (m Makefile) defaultTargetName() string {
	if m.DefaultTarget == nil {
		return ""
	}
	result := runtime.FuncForPC(reflect.ValueOf(m.DefaultTarget).Pointer()).Name()
	result = strings.TrimPrefix(result, "main.")
	result = strings.TrimPrefix(result, m.namespaceName()+".")
	result = strings.Split(result, "-")[0]
	for _, r := range result {
		if !unicode.IsLetter(r) {
			panic(fmt.Sprintf("Invalid default target %s", result))
		}
	}
	return result
}

func generateMakefile(_ context.Context, g *codegen.File, pkg *doc.Package, mk Makefile, mks ...Makefile) error {
	includePath, err := filepath.Rel(filepath.Dir(mk.Path), FromSageDir())
	if err != nil {
		return err
	}
	g.P("# To learn more, see ", includePath, "/main.go and https://github.com/einride/sage.")
	if len(mk.defaultTargetName()) != 0 {
		g.P()
		g.P(".DEFAULT_GOAL := ", toMakeTarget(mk.defaultTargetName()))
	}
	g.P()
	g.P("cwd := $(dir $(realpath $(firstword $(MAKEFILE_LIST))))")
	g.P("sagefile := $(abspath $(cwd)/", filepath.Join(includePath, binDir, sageFileBinary), ")")
	g.P()
	g.P("# Setup Go.")
	g.P("go := $(shell command -v go 2>/dev/null)")
	g.P("export GOWORK ?= off")
	g.P("ifndef go")
	g.P("SAGE_GO_VERSION ?= ", defaultGoVersion)
	g.P(
		"export GOROOT := $(abspath $(cwd)/",
		filepath.Join(includePath, toolsDir, "go", "$(SAGE_GO_VERSION)", "go"),
		")",
	)
	g.P("export PATH := $(PATH):$(GOROOT)/bin")
	g.P("go := $(GOROOT)/bin/go")
	g.P("os := $(shell uname | tr '[:upper:]' '[:lower:]')")
	g.P("arch := $(shell uname -m)")
	g.P("ifeq ($(arch),x86_64)")
	g.P("arch := amd64")
	g.P("endif")
	g.P("$(go):")
	g.P("\t$(info installing Go $(SAGE_GO_VERSION)...)")
	g.P("\t@mkdir -p $(dir $(GOROOT))")
	g.P("\t@curl -sSL https://go.dev/dl/go$(SAGE_GO_VERSION).$(os)-$(arch).tar.gz | tar xz -C $(dir $(GOROOT))")
	g.P("\t@touch $(GOROOT)/go.mod")
	g.P("\t@chmod +x $(go)")
	g.P("endif")
	g.P()
	g.P(".PHONY: $(sagefile)")
	g.P("$(sagefile): $(go)")
	g.P("\t@cd ", includePath, " && $(go) mod tidy && $(go) run .")
	g.P()
	g.P(".PHONY: sage")
	g.P("sage:")
	g.P("\t@$(MAKE) $(sagefile)")
	g.P()
	g.P(".PHONY: update-sage")
	g.P("update-sage: $(go)")
	g.P("\t@cd ", includePath, " && $(go) get go.einride.tech/sage@latest && $(go) mod tidy && $(go) run .")
	g.P()
	g.P(".PHONY: clean-sage")
	g.P("clean-sage:")
	g.P(
		"\t@git clean -fdx ",
		filepath.Join(includePath, toolsDir),
		" ",
		filepath.Join(includePath, binDir),
		" ",
		filepath.Join(includePath, buildDir),
	)
	forEachTargetFunction(pkg, func(function *doc.Func, _ *doc.Type) {
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
			g.P("\t$(MAKE) -C ", mkPath, " -f ", filepath.Base(i.Path))
		}
	}
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

// toSageFunction converts input to a sage Target name with the provided args.
func toSageFunction(target string, args []string) string {
	for _, arg := range args {
		arg = fmt.Sprintf("\"$(%s)\"", arg)
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
