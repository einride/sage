package main

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/magefile/mage/mage"
	"go.einride.tech/mage-tools/mglogr"
	"go.einride.tech/mage-tools/mgmake"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

//go:embed mgmake_gen.go
var mgmakeGen []byte

func main() {
	ctx := logr.NewContext(context.Background(), mglogr.New("mage-tools-build"))
	logr.FromContextOrDiscard(ctx).Info("building binary and generating Makefiles...")
	mageDir := mgpath.FromGitRoot(mgpath.MageDir)
	makeGenGo := filepath.Join(mageDir, "mgmake_gen.go")
	executable := mgpath.FromTools(mgpath.MagefileBinary)
	cmd := mgtool.Command(ctx, "git", "clean", "-fdx", filepath.Dir(executable))
	cmd.Dir = mageDir
	cmd.Stdout = io.Discard
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	if err := os.WriteFile(makeGenGo, mgmakeGen, 0o600); err != nil {
		panic(err)
	}
	cmd = mgtool.Command(ctx, "go", "mod", "tidy")
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
	cmd = mgtool.Command(ctx, executable, mgmake.GenMakefilesTarget, executable)
	cmd.Dir = mageDir
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}
