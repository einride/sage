package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/magefile/mage/mage"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

var (
	//go:embed example/.mage/tools.mk
	toolsMk string
	//go:embed example/.mage/magefile.go
	magefile string
	//go:embed example/Makefile
	makefile string
	//go:embed example/.mage/mgmake_gen.go
	mgmake string
	// nolint: gochecknoglobals
	mageDir = mgpath.FromGitRoot(mgpath.MageDir)
)

func main() {
	logger := mglog.Logger("mage-tools")
	usage := func() {
		logger.Info(`Usage:
	init	to initialize mage-tools`)
		os.Exit(0)
	}
	if len(os.Args) <= 1 {
		usage()
	}
	switch os.Args[1] {
	case "init":
		if err := initMageTools(); err != nil {
			log.Fatalf(err.Error())
		}
	case "gen":
		if err := gen(); err != nil {
			log.Fatalf(err.Error())
		}
	default:
		usage()
	}
}

func gen() error {
	mglog.Logger("gen").Info("generating makefiles...")
	executable := filepath.Join(mgpath.Tools(), "mgmake", "magefile")
	if err := mgtool.RunInDir("git", mageDir, "clean", "-fdx", filepath.Dir(executable)); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(mageDir, mgpath.MakeGenGo), []byte(mgmake), 0o600); err != nil {
		return err
	}
	if err := mgtool.RunInDir("go", mageDir, "mod", "tidy"); err != nil {
		return err
	}
	if exit := mage.ParseAndRun(os.Stdout, os.Stderr, os.Stdin, []string{"-compile", executable}); exit != 0 {
		return fmt.Errorf("faild to compile magefile binary")
	}
	return mgtool.RunInDir(executable, mageDir, mgpath.GenMakefilesTarget, executable)
}

func initMageTools() error {
	logger := mglog.Logger("init")
	logger.Info("initializing mage-tools...")

	if mgpath.FromWorkDir(".") != mgpath.FromGitRoot(".") {
		return fmt.Errorf("can only be generated in git root directory")
	}

	if err := os.Mkdir(mageDir, 0o755); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(mageDir, mgpath.ToolsMk), []byte(toolsMk), 0o600); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(mageDir, "magefile.go"), []byte(magefile), 0o600); err != nil {
		return err
	}

	_, err := os.Stat("Makefile")
	if err != nil {
		// Write Makefile
		if err := os.WriteFile("Makefile", []byte(makefile), 0o600); err != nil {
			return err
		}
	} else {
		const mm = "Makefile.MAGE"
		logger.Info(fmt.Sprintf("Makefile already exist, writing to %s", mm))
		if err := os.WriteFile(mm, []byte(makefile), 0o600); err != nil {
			return err
		}
	}
	if err := mgtool.RunInDir("go", mageDir, []string{"mod", "init", "mage-tools"}...); err != nil {
		return err
	}
	if err := mgtool.RunInDir("go", mageDir, []string{"mod", "tidy"}...); err != nil {
		return err
	}
	gitIgnore, err := os.OpenFile(".gitignore", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer gitIgnore.Close()
	if _, err := gitIgnore.WriteString(mgpath.Tools()); err != nil {
		return err
	}
	// TODO: Output some documentation, next steps after init, and useful links.
	logger.Info("mage-tools initialized!")
	return nil
}
