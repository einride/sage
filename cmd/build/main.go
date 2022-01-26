package main

import (
	"context"
	_ "embed"

	"github.com/go-logr/logr"
	"go.einride.tech/mage-tools/mg"
)

func main() {
	ctx := logr.NewContext(context.Background(), mg.NewLogger("mage-tools-build"))
	mageDir := mg.FromMageDir()
	cmd := mg.Command(ctx, "go", "mod", "tidy")
	cmd.Dir = mageDir
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	cmd = mg.Command(ctx, "go", "run", ".")
	cmd.Dir = mageDir
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}
