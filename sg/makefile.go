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

const defaultGoVersion = "1.23.4"

type Makefile struct {
	Namespace     interface{}
	Path          string
	DefaultTarget interface{}
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

func generateClaudeMarkdown(_ context.Context, g *codegen.File, pkg *doc.Package, mk Makefile, mks ...Makefile) error {
	includePath, err := filepath.Rel(filepath.Dir(mk.Path), FromSageDir())
	if err != nil {
		return err
	}
	g.P("# Sage Build System - Claude Assistant Guide")
	g.P()
	g.P("This file provides context for Claude on how to use the Sage build system and available make targets.")
	g.P("Generated from .sage/main.go - for more info see https://github.com/einride/sage")
	g.P()

	if len(mk.defaultTargetName()) != 0 {
		g.P("## Default Target")
		g.P()
		g.P("- **Default**: `make` (runs `make ", toMakeTarget(mk.defaultTargetName()), "`)")
		g.P()
	}

	g.P("## Available Make Targets")
	g.P()

	forEachTargetFunction(pkg, func(function *doc.Func, _ *doc.Type) {
		if function.Recv == mk.namespaceName() {
			targetName := toMakeTarget(getTargetFunctionName(function))
			functionName := getTargetFunctionName(function)

			g.P("### `make ", targetName, "`")
			g.P()

			// Extract metadata from function body
			metadata := extractFunctionMetadata(function)

			// Add function description from Go doc comment or extracted purpose
			if function.Doc != "" {
				g.P("**Description**: ", strings.TrimSpace(function.Doc))
			} else if metadata.Purpose != "" {
				g.P("**Description**: ", metadata.Purpose)
			} else {
				g.P("**Description**: ", functionName, " target")
			}
			g.P()

			// Add function name for reference
			g.P("**Source Function**: `", functionName, "()` in ", includePath, "/main.go")
			g.P()

			// Add sage tool information
			if len(metadata.SageTools) > 0 {
				g.P("**Sage Tools Used**: ", strings.Join(metadata.SageTools, ", "))
				g.P()
			}

			// Add dependencies
			if len(metadata.Dependencies) > 0 {
				g.P("**Dependencies**: ", strings.Join(metadata.Dependencies, ", "))
				g.P()
			}

			// Add parameters if any
			if len(function.Decl.Type.Params.List) > 1 {
				args := toMakeVars(function.Decl.Type.Params.List[1:])
				g.P("**Parameters**:")
				for i, arg := range args {
					paramType := "string" // default
					if i < len(function.Decl.Type.Params.List[1:]) {
						paramType = fmt.Sprint(function.Decl.Type.Params.List[1:][i].Type)
					}
					g.P("- `", arg, "` (", paramType, ")")
				}
				g.P()
				g.P("**Usage**: `make ", targetName)
				for _, arg := range args {
					g.P(" ", arg, "=\"value\"")
				}
				g.P("`")
			} else {
				g.P("**Usage**: `make ", targetName, "`")
			}
			g.P()
		}
	})

	// Add sage-specific commands
	g.P("## Sage System Commands")
	g.P()
	g.P("### `make sage`")
	g.P("**Description**: Builds the sage binary and generates Makefiles")
	g.P("**Usage**: `make sage`")
	g.P()
	g.P("### `make update-sage`")
	g.P("**Description**: Updates Sage to the latest version")
	g.P("**Usage**: `make update-sage`")
	g.P()
	g.P("### `make clean-sage`")
	g.P("**Description**: Cleans sage build artifacts")
	g.P("**Usage**: `make clean-sage`")
	g.P()

	g.P("## Understanding .sage Directory")
	g.P()
	g.P("The `.sage/` directory contains:")
	g.P("- `main.go`: Defines all the build targets and their implementations")
	g.P("- `tools/`: Downloaded build tools (managed by Sage)")
	g.P("- `bin/`: Built sage binary")
	g.P("- `build/`: Build artifacts and outputs")
	g.P()
	g.P("## For Claude: How to Work with This Repository")
	g.P()
	g.P("1. **View available targets**: Read this file or run `make` without arguments")
	g.P("2. **Understand a target**: Check the source function in `.sage/main.go`")
	g.P("3. **Run build tasks**: Use the make commands listed above")
	g.P("4. **Add new targets**: Add public functions to `.sage/main.go`, then run `make sage`")
	g.P("5. **Debug build issues**: Check `.sage/main.go` for the actual implementation")

	return nil
}

// FunctionMetadata holds extracted metadata about a function
type FunctionMetadata struct {
	Purpose      string
	Dependencies []string
	SageTools    []string
}

// extractFunctionMetadata parses the function body to extract dependencies and tools
func extractFunctionMetadata(function *doc.Func) FunctionMetadata {
	metadata := FunctionMetadata{}

	if function.Decl.Body == nil {
		return metadata
	}

	// Walk through function body statements
	for _, stmt := range function.Decl.Body.List {
		extractFromStatement(stmt, &metadata)
	}

	return metadata
}

// extractFromStatement recursively extracts metadata from AST statements
func extractFromStatement(stmt ast.Stmt, metadata *FunctionMetadata) {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		extractFromExpr(s.X, metadata)
	case *ast.AssignStmt:
		for _, expr := range s.Rhs {
			extractFromExpr(expr, metadata)
		}
	case *ast.ReturnStmt:
		for _, expr := range s.Results {
			extractFromExpr(expr, metadata)
		}
	case *ast.IfStmt:
		if s.Init != nil {
			extractFromStatement(s.Init, metadata)
		}
		if s.Cond != nil {
			extractFromExpr(s.Cond, metadata)
		}
		if s.Body != nil {
			for _, bodyStmt := range s.Body.List {
				extractFromStatement(bodyStmt, metadata)
			}
		}
		if s.Else != nil {
			extractFromStatement(s.Else, metadata)
		}
	case *ast.BlockStmt:
		for _, blockStmt := range s.List {
			extractFromStatement(blockStmt, metadata)
		}
	}
}

// extractFromExpr extracts metadata from expressions
func extractFromExpr(expr ast.Expr, metadata *FunctionMetadata) {
	switch e := expr.(type) {
	case *ast.CallExpr:
		extractFromCallExpr(e, metadata)
	case *ast.SelectorExpr:
		// Check for sage tool usage like sggo.TestCommand, sggolangcilintv2.Run
		if ident, ok := e.X.(*ast.Ident); ok {
			if strings.HasPrefix(ident.Name, "sg") && ident.Name != "sg" {
				toolName := ident.Name + "." + e.Sel.Name
				if !contains(metadata.SageTools, toolName) {
					metadata.SageTools = append(metadata.SageTools, toolName)
				}
			}
		}
	}
}

// extractFromCallExpr extracts metadata from function call expressions
func extractFromCallExpr(call *ast.CallExpr, metadata *FunctionMetadata) {
	// Check for sg.Deps and sg.SerialDeps calls
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "sg" {
			if sel.Sel.Name == "Deps" || sel.Sel.Name == "SerialDeps" {
				// Extract dependency function names (skip first argument which is context)
				for i, arg := range call.Args {
					if i == 0 { // Skip context argument
						continue
					}
					if depName := extractDepName(arg); depName != "" {
						if !contains(metadata.Dependencies, depName) {
							metadata.Dependencies = append(metadata.Dependencies, depName)
						}
					}
				}
			} else if sel.Sel.Name == "Logger" {
				// Extract purpose from logger.Println calls
				extractLoggerPurpose(call, metadata)
			}
		}
	}

	// Check for chained calls like sggo.TestCommand(ctx).Run()
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if callExpr, ok := sel.X.(*ast.CallExpr); ok {
			extractFromCallExpr(callExpr, metadata)
		}
	}

	// Recursively check arguments
	for _, arg := range call.Args {
		extractFromExpr(arg, metadata)
	}
}

// extractDepName extracts dependency name from function reference
func extractDepName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.CallExpr:
		// Handle sg.Fn(function, args...) calls
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "sg" && sel.Sel.Name == "Fn" {
				if len(e.Args) > 0 {
					return extractDepName(e.Args[0])
				}
			}
		}
	}
	return ""
}

// extractLoggerPurpose tries to extract purpose from logger statements
func extractLoggerPurpose(call *ast.CallExpr, metadata *FunctionMetadata) {
	// Look for subsequent method calls on the logger
	// This is a simplified extraction - in practice, we'd need to look at the next statement
	// For now, we'll leave this empty and rely on explicit documentation
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
