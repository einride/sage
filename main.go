package main

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"go.einride.tech/sage/sg"
)

var (
	//go:embed example/.gitignore
	gitignore []byte
	//go:embed example/.sage/sagefile.go
	sagefile []byte
	//go:embed example/.github/dependabot.yml
	exampleDependabotYML []byte
)

func main() {
	ctx := logr.NewContext(context.Background(), sg.NewLogger("sage"))
	logger := logr.FromContextOrDiscard(ctx)
	usage := func() {
		logger.Info(`Usage:
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
	logger := logr.FromContextOrDiscard(ctx)
	logger.Info("initializing sage...")
	if sg.FromWorkDir() != sg.FromGitRoot() {
		panic("can only be generated in git root directory")
	}
	if err := os.Mkdir(sg.FromSageDir(), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(sg.FromSageDir("sagefile.go"), sagefile, 0o600); err != nil {
		panic(err)
	}
	if err := os.WriteFile(sg.FromSageDir(".gitignore"), gitignore, 0o600); err != nil {
		panic(err)
	}
	_, err := os.Stat(sg.FromGitRoot("Makefile"))
	if err == nil {
		const mm = "Makefile.old"
		logger.Info(fmt.Sprintf("Makefile already exists, renaming  Makefile to %s", mm))
		if err := os.Rename(sg.FromGitRoot("Makefile"), sg.FromGitRoot(mm)); err != nil {
			panic(err)
		}
	}
	cmd := sg.Command(ctx, "go", "mod", "init", "sage")
	cmd.Dir = sg.FromSageDir()
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	cmd = sg.Command(ctx, "go", "mod", "tidy")
	cmd.Dir = sg.FromSageDir()
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	if err := addToDependabot(); err != nil {
		panic(err)
	}
	// Generate make targets
	cmd = sg.Command(ctx, "go", "run", "go.einride.tech/sage/cmd/build")
	cmd.Dir = sg.FromSageDir()
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	logger.Info(`
sage has been successfully initialized!

To get started, have a look at the sagefile.go in the .sage directory,
and look at https://github.com/einride/sage#readme to learn more
`)
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
