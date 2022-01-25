package main

import (
	"context"
	_ "embed"
	"go.einride.tech/mage-tools/mg"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"go.einride.tech/mage-tools/mglogr"
	"go.einride.tech/mage-tools/mgtool"
)

//go:embed gen/mgmake_gen.go
var mgmakeGen []byte

func main() {
	ctx := logr.NewContext(context.Background(), mglogr.New("mage-tools-build"))
	logr.FromContextOrDiscard(ctx).Info("building binary and generating Makefiles...")
	mageDir := mg.FromMageDir()
	makeGenGo := filepath.Join(mageDir, mg.MakeGenGo)
	if err := os.WriteFile(makeGenGo, mgmakeGen, 0o600); err != nil {
		panic(err)
	}
	defer func() {
		if err := os.Remove(makeGenGo); err != nil {
			panic(err)
		}
	}()
	cmd := mgtool.Command(ctx, "go", "mod", "tidy")
	cmd.Dir = mageDir
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	cmd = mgtool.Command(ctx, "go", "run", ".")
	cmd.Dir = mageDir
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}
