package sg

import (
	"context"
	"fmt"
	"go/ast"
	"go/doc"
	"path/filepath"
	"reflect"
	"regexp"
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

var validMakeTarget = regexp.MustCompile(`^[a-z0-9]([a-z0-9._-]*[a-z0-9])?$`)

// getMakeTargetOverride parses the doc comment of a function for a "//sage:target" directive.
// Returns the override target name, or empty string if no annotation is found.
// We read from function.Decl.Doc (the raw AST comment group) because Go's
// ast.CommentGroup.Text() filters out directive-style comments (//word:...).
func getMakeTargetOverride(function *doc.Func) string {
	if function.Decl == nil || function.Decl.Doc == nil {
		return ""
	}
	for _, comment := range function.Decl.Doc.List {
		text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		if after, ok := strings.CutPrefix(text, "sage:target "); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}

// validateMakeTargetOverride validates a sage:target annotation value.
// Panics if the value is invalid.
func validateMakeTargetOverride(name string) {
	if name == "" {
		panic("sage:target annotation value must not be empty")
	}
	if !validMakeTarget.MatchString(name) {
		panic(fmt.Sprintf("sage:target annotation value %q is invalid: must match %s", name, validMakeTarget.String()))
	}
}

// effectiveMakeTarget returns the Make target name for a function.
// If the function has a sage:target annotation, the override is returned (after validation).
// Otherwise, the default kebab-case conversion is used.
func effectiveMakeTarget(function *doc.Func) string {
	if override := getMakeTargetOverride(function); override != "" {
		validateMakeTargetOverride(override)
		return override
	}
	return toMakeTarget(getTargetFunctionName(function))
}

// findDocFunc finds a doc.Func by name in a doc.Package.
// It searches both top-level functions and type methods.
func findDocFunc(pkg *doc.Package, name string) *doc.Func {
	for _, f := range pkg.Funcs {
		if f.Name == name {
			return f
		}
	}
	for _, t := range pkg.Types {
		for _, f := range t.Methods {
			if f.Name == name {
				return f
			}
		}
	}
	return nil
}

func generateMakefile(_ context.Context, g *codegen.File, pkg *doc.Package, mk Makefile, mks ...Makefile) error {
	includePath, err := filepath.Rel(filepath.Dir(mk.Path), FromSageDir())
	if err != nil {
		return err
	}
	g.P("# To learn more, see ", includePath, "/main.go and https://github.com/einride/sage.")
	if defaultName := mk.defaultTargetName(); len(defaultName) != 0 {
		g.P()
		defaultTarget := toMakeTarget(defaultName)
		if f := findDocFunc(pkg, defaultName); f != nil {
			defaultTarget = effectiveMakeTarget(f)
		}
		g.P(".DEFAULT_GOAL := ", defaultTarget)
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
	seenTargets := map[string]string{} // make target -> Go function name
	forEachTargetFunction(pkg, func(function *doc.Func, _ *doc.Type) {
		if function.Recv == mk.namespaceName() {
			target := effectiveMakeTarget(function)
			funcName := getTargetFunctionName(function)
			if existing, ok := seenTargets[target]; ok {
				panic(fmt.Sprintf(
					"duplicate Make target %q: generated by both %s and %s",
					target, existing, funcName,
				))
			}
			seenTargets[target] = funcName
			g.P()
			g.P(".PHONY: ", target)
			g.P(target, ": $(sagefile)")
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
