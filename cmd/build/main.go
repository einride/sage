package main

import (
	"context"
	_ "embed"

	"github.com/go-logr/logr"
	"go.einride.tech/sage/sg"
)

func main() {
	ctx := logr.NewContext(context.Background(), sg.NewLogger("sage"))
	cmd := sg.Command(ctx, "go", "mod", "tidy")
	cmd.Dir = sg.FromSageDir()
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	cmd = sg.Command(ctx, "go", "run", ".")
	cmd.Dir = sg.FromSageDir()
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}
