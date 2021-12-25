package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgtool"
)

var (
	//go:embed example/.mage/tools.mk
	toolsMk string
	//go:embed example/.mage/main.go
	mageMain string
	//go:embed example/.mage/magefile.go
	magefile string
	//go:embed example/Makefile
	makefile string
)

func main() {
	usage := func() {
		logger := mglog.Logger("main")
		logger.Info(`Usage:
	init	to initialize mage-tools`)
		os.Exit(0)
	}
	if len(os.Args) <= 1 {
		usage()
	}

	switch os.Args[1] {
	case "init":
		initMageTools()
	default:
		usage()
	}
}

func initMageTools() {
	logger := mglog.Logger("initMageTools")
	logger.Info("generating mage-tools...")
	defer func() {
		if err := recover(); err != nil {
			logger.Error(err.(error), err.(error).Error())
		}
	}()

	if mgtool.GetCWDPath("") != mgtool.GetGitRootPath("") {
		panic(fmt.Errorf("can only be generated in git root directory"))
	}

	mageDir := filepath.Join(mgtool.GetGitRootPath(""), ".mage")
	err := os.Mkdir(mageDir, 0o755)
	if err != nil {
		panic(err)
	}

	// Write tools.mk
	err = os.WriteFile(filepath.Join(mageDir, "tools.mk"), []byte(toolsMk), 0o644)
	if err != nil {
		panic(err)
	}

	// Write main.go
	err = os.WriteFile(filepath.Join(mageDir, "main.go"), []byte(mageMain), 0o644)
	if err != nil {
		panic(err)
	}

	// Write magefile.go
	err = os.WriteFile(filepath.Join(mageDir, "magefile.go"), []byte(magefile), 0o644)
	if err != nil {
		panic(err)
	}

	_, err = os.Stat("Makefile")
	if err != nil {
		// Write Makefile
		err = os.WriteFile("Makefile", []byte(makefile), 0o644)
		if err != nil {
			panic(err)
		}
	} else {
		const mm = "Makefile.MAGE"
		logger.Info(fmt.Sprintf("Makefile already exist, writing to %s", mm))
		err = os.WriteFile(mm, []byte(makefile), 0o644)
		if err != nil {
			panic(err)
		}
	}

	if err := execCommandInDirectory(mageDir, "go", []string{"mod", "init", "mage-tools"}...); err != nil {
		panic(err)
	}
	if err != nil {
		panic(err)
	}
	if err := execCommandInDirectory(mageDir, "go", []string{"mod", "tidy"}...); err != nil {
		panic(err)
	}
	logger.Info("Done...")
}

func execCommandInDirectory(dir string, command string, args ...string) (err error) {
	c := exec.Command(command, args...)
	c.Env = os.Environ()
	c.Stderr = &bytes.Buffer{}
	c.Stdout = &bytes.Buffer{}
	c.Stdin = os.Stdin
	c.Dir = dir

	return c.Run()
}
