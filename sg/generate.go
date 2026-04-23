package sg

import (
	"context"
	"fmt"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"

	"go.einride.tech/sage/internal/codegen"
)

// GenerateMakefiles defines which Makefiles should be generated.
func GenerateMakefiles(mks ...Makefile) {
	ctx := WithLogger(context.Background(), NewLogger("sage"))
	Logger(ctx).Println("building binary and generating Makefiles...")
	if len(mks) == 0 {
		panic("no makefiles to generate, see https://github.com/einride/sage#readme for more info")
	}
	// TODO: replace the deprecated function call.
	pkgs, err := parser.ParseDir(token.NewFileSet(), FromSageDir(), nil, parser.ParseComments) //nolint:staticcheck
	if err != nil {
		panic(fmt.Errorf("failed to parse directory: %v", err))
	}
	if len(pkgs) != 1 {
		panic(fmt.Errorf("parser returned unexpected number of packages: %d", len(pkgs)))
	}
	var pkg *doc.Package
	for _, p := range pkgs {
		pkg = doc.New(p, "./", doc.PreserveAST)
	}
	// update .gitignore file
	const gitignoreContent = ".gitignore\ntools/\nbin/\nbuild/\n"
	if err := os.WriteFile(FromSageDir(".gitignore"), []byte(gitignoreContent), 0o600); err != nil {
		panic(err)
	}
	// generate init file before compiling
	initFilename := FromSageDir("generating_sagefile.go")
	initFile := codegen.NewFile(codegen.FileConfig{
		Filename:    initFilename,
		Package:     pkg.Name,
		GeneratedBy: "go.einride.tech/sage",
	})
	if err := generateInitFile(initFile, pkg, mks); err != nil {
		panic(err)
	}
	initFileContent, err := initFile.GoContent()
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(initFilename, initFileContent, 0o600); err != nil {
		panic(err)
	}
	defer func() {
		_ = os.Remove(initFilename)
	}()
	// Compile binary
	compileCmd := Command(ctx, "go", "build", "-o", FromBinDir(sageFileBinary), ".")
	compileCmd.Dir = FromSageDir()
	if err := compileCmd.Run(); err != nil {
		panic(fmt.Errorf("error compiling sagefiles: %w", err))
	}
	// Generate makefiles
	for _, v := range mks {
		if v.Path == "" {
			panic("Path needs to be defined")
		}
		mk := codegen.NewMakefile(codegen.FileConfig{
			GeneratedBy: "go.einride.tech/sage",
		})
		if err := generateMakefile(ctx, mk, pkg, v, mks...); err != nil {
			panic(err)
		}
		if err := os.WriteFile(v.Path, mk.RawContent(), 0o600); err != nil {
			panic(err)
		}
	}
	// Generate GitHub Actions workflows
	for _, v := range mks {
		if v.GitHubWorkflow == nil {
			continue
		}
		if v.DefaultTarget == nil {
			panic("GitHubWorkflow requires Makefile.DefaultTarget to be set")
		}
		if v.GitHubWorkflow.Path == "" {
			panic("GitHubWorkflow.Path must be set")
		}
		Logger(ctx).Println("capturing plan and generating workflow", v.GitHubWorkflow.Path)
		if err := generateGitHubWorkflow(ctx, pkg, v); err != nil {
			panic(err)
		}
	}
}

// generateGitHubWorkflow invokes the compiled sagefile binary in plan mode,
// parses the recorded plan, and writes a rendered GitHub Actions workflow
// YAML to the configured path.
func generateGitHubWorkflow(ctx context.Context, pkg *doc.Package, mk Makefile) error {
	planFile, err := os.CreateTemp("", "sage-plan-*.jsonl")
	if err != nil {
		return fmt.Errorf("create plan temp file: %w", err)
	}
	planPath := planFile.Name()
	if err := planFile.Close(); err != nil {
		return fmt.Errorf("close plan temp file: %w", err)
	}
	defer func() { _ = os.Remove(planPath) }()

	planCtx := ContextWithEnv(ctx, PlanOutputEnv+"="+planPath)
	planCmd := Command(planCtx, FromBinDir(sageFileBinary), mk.defaultTargetName())
	if err := planCmd.Run(); err != nil {
		return fmt.Errorf("capture plan for %s: %w", mk.defaultTargetName(), err)
	}

	plan, err := ReadPlan(planPath)
	if err != nil {
		return err
	}
	groups, err := planToWorkflowGroups(pkg, plan)
	if err != nil {
		return err
	}
	yaml, err := renderWorkflow(*mk.GitHubWorkflow, groups)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(mk.GitHubWorkflow.Path), 0o755); err != nil {
		return fmt.Errorf("create workflow directory: %w", err)
	}
	if err := os.WriteFile(mk.GitHubWorkflow.Path, yaml, 0o600); err != nil {
		return fmt.Errorf("write workflow file: %w", err)
	}
	return nil
}
