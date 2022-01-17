package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/magefile/mage/mage"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgmake"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

//go:embed mgmake_gen.go
var mgmakeGen []byte

func main() {
	mageDir := mgpath.FromGitRoot(mgpath.MageDir)
	makeGenGo := filepath.Join(mageDir, "mgmake_gen.go")
	mglog.Logger("build").Info("building binary and generating makefiles...")
	executable := filepath.Join(mgpath.Tools(), mgpath.MagefileBinary)
	if err := mgtool.RunInDir("git", mageDir, "clean", "-fdx", filepath.Dir(executable)); err != nil {
		panic(err)
	}
	if err := os.WriteFile(makeGenGo, mgmakeGen, 0o600); err != nil {
		panic(err)
	}
	if err := mgtool.RunInDir("go", mageDir, "mod", "tidy"); err != nil {
		panic(err)
	}
	if exit := mage.ParseAndRun(os.Stdout, os.Stderr, os.Stdin, []string{"-compile", executable}); exit != 0 {
		panic(fmt.Errorf("faild to compile magefile binary"))
	}
	if err := os.Remove(makeGenGo); err != nil {
		panic(err)
	}
	if err := mgtool.RunInDir(executable, mageDir, mgmake.GenMakefilesTarget, executable); err != nil {
		panic(err)
	}
}
