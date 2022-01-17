package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/magefile/mage/mage"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

//go:embed mgmake_gen.go
var mgmake []byte

func main() {
	mageDir := mgpath.FromGitRoot(mgpath.MageDir)
	mglog.Logger("build").Info("building binary and generating makefiles...")
	executable := filepath.Join(mgpath.Tools(), mgpath.MagefileBinary)
	if err := mgtool.RunInDir("git", mageDir, "clean", "-fdx", filepath.Dir(executable)); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(mageDir, mgpath.MakeGenGo), mgmake, 0o600); err != nil {
		panic(err)
	}
	if err := mgtool.RunInDir("go", mageDir, "mod", "tidy"); err != nil {
		panic(err)
	}
	if exit := mage.ParseAndRun(os.Stdout, os.Stderr, os.Stdin, []string{"-compile", executable}); exit != 0 {
		panic(fmt.Errorf("faild to compile magefile binary"))
	}
	os.Remove(filepath.Join(mageDir, mgpath.MakeGenGo))
	if err := mgtool.RunInDir(executable, mageDir, mgpath.GenMakefilesTarget, executable); err != nil {
		panic(err)
	}
}
