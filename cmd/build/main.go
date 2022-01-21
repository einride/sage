package main

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/magefile/mage/mage"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgmake"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
	"go.einride.tech/mage-tools/tools/mggo"
)

//go:embed mgmake_gen.go
var mgmakeGen []byte

func main() {
	mageDir := mgpath.FromGitRoot(mgpath.MageDir)
	makeGenGo := filepath.Join(mageDir, "mgmake_gen.go")
	mglog.Logger("build").Info("building binary and generating makefiles...")
	executable := mgpath.FromTools(mgpath.MagefileBinary)
	cmd := mgtool.Command("git", "clean", "-fdx", filepath.Dir(executable))
	cmd.Dir = mageDir
	cmd.Stdout = io.Discard
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	if err := os.WriteFile(makeGenGo, mgmakeGen, 0o600); err != nil {
		panic(err)
	}
	cmd = mggo.Command("mod", "tidy")
	cmd.Dir = mageDir
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	if exit := mage.ParseAndRun(os.Stdout, os.Stderr, os.Stdin, []string{"-compile", executable}); exit != 0 {
		panic(fmt.Errorf("faild to compile magefile binary"))
	}
	if err := os.Remove(makeGenGo); err != nil {
		panic(err)
	}
	cmd = mgtool.Command(executable, mgmake.GenMakefilesTarget, executable)
	cmd.Dir = mageDir
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}
