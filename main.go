package main

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"go.einride.tech/sage/sg"
)

var (
	//go:embed example/.sage/main.go
	mainFile []byte
	//go:embed example/.github/dependabot.yml
	exampleDependabotYML []byte
)

func main() {
	ctx := sg.WithLogger(context.Background(), sg.NewLogger("sage"))
	usage := func() {
		sg.Logger(ctx).Println(`Usage:
	init
		to initialize sage`)
		os.Exit(0)
	}
	if len(os.Args) < 2 || len(os.Args) > 2 {
		usage()
	}
	switch os.Args[1] {
	case "init":
		initSage(ctx)
	default:
		usage()
	}
}

func initSage(ctx context.Context) {
	sg.Logger(ctx).Println("initializing sage...")
	if sg.FromWorkDir() != sg.FromGitRoot() {
		sg.Logger(ctx).Fatal("can only be generated in git root directory")
	}
	sageModulePath, err := resolveSageModulePath(ctx)
	if err != nil {
		sg.Logger(ctx).Fatal(err)
	}
	if err := os.Mkdir(sg.FromSageDir(), 0o755); err != nil {
		sg.Logger(ctx).Fatal(err)
	}
	if err := os.WriteFile(sg.FromSageDir("main.go"), mainFile, 0o600); err != nil {
		sg.Logger(ctx).Fatal(err)
	}
	if _, err := os.Stat(sg.FromGitRoot("Makefile")); err == nil {
		const mm = "Makefile.old"
		sg.Logger(ctx).Printf("Makefile already exists, renaming  Makefile to %s", mm)
		if err := os.Rename(sg.FromGitRoot("Makefile"), sg.FromGitRoot(mm)); err != nil {
			sg.Logger(ctx).Fatal(err)
		}
	}
	cmd := sg.Command(ctx, "go", "mod", "init", sageModulePath)
	cmd.Dir = sg.FromSageDir()
	if err := cmd.Run(); err != nil {
		sg.Logger(ctx).Fatal(err)
	}
	cmd = sg.Command(ctx, "go", "mod", "tidy")
	cmd.Dir = sg.FromSageDir()
	if err := cmd.Run(); err != nil {
		sg.Logger(ctx).Fatal(err)
	}
	if err := addToDependabot(); err != nil {
		sg.Logger(ctx).Fatal(err)
	}
	// Generate make targets
	// Use exec.CommandContext instead of sg.Command to avoid double log tags.
	cmd = exec.CommandContext(ctx, "go", "run", ".")
	cmd.Dir = sg.FromSageDir()
	if err := cmd.Run(); err != nil {
		sg.Logger(ctx).Fatal(err)
	}
	sg.Logger(ctx).Println(`successfully initialized!

To get started, have a look at the main.go in the .sage directory,
and look at https://github.com/einride/sage#readme to learn more`)
}

func hasSageDependabotConfig(dependabotYML []byte) bool {
	sc := bufio.NewScanner(bytes.NewReader(dependabotYML))
	sc.Split(bufio.ScanLines)
	for sc.Scan() {
		if bytes.Contains(sc.Bytes(), []byte("directory:")) && bytes.Contains(sc.Bytes(), []byte(sg.FromSageDir())) {
			return true
		}
	}
	return false
}

func appendSageDependabotConfig(dependabotYML []byte) []byte {
	relativeSageDir, err := filepath.Rel(sg.FromGitRoot(), sg.FromSageDir())
	if err != nil {
		panic(err)
	}
	dependabotConfig := fmt.Sprintf(
		`
  - package-ecosystem: gomod
    directory: %s
    schedule:
      interval: weekly`,
		relativeSageDir,
	)
	return append(dependabotYML, []byte(dependabotConfig)...)
}

func addToDependabot() error {
	dependabotYMLPath := sg.FromGitRoot(".github", "dependabot.yml")
	dependabotYML, err := os.ReadFile(dependabotYMLPath)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(dependabotYMLPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dependabotYMLPath, exampleDependabotYML, 0o600)
	}
	if hasSageDependabotConfig(dependabotYML) {
		return nil
	}
	return os.WriteFile(dependabotYMLPath, appendSageDependabotConfig(dependabotYML), 0o600)
}

func resolveSageModulePath(ctx context.Context) (string, error) {
	const moduleName = ".sage"
	if _, err := os.Stat("go.mod"); err != nil {
		return moduleName, nil //nolint:nilerr
	}
	var out bytes.Buffer
	cmd := sg.Command(ctx, "go", "mod", "edit", "-json")
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	var modFile struct {
		Module struct {
			Path string
		}
	}
	if err := json.Unmarshal(out.Bytes(), &modFile); err != nil {
		return "", err
	}
	if modFile.Module.Path == "" {
		return moduleName, nil
	}
	return filepath.Join(modFile.Module.Path, moduleName), nil
}
