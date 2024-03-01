package gofumpt

import (
	"context"
	"os/exec"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	name    = "gci"
	version = "v0.13.0"
)

// Command returns an [*exec.Cmd] for golines.
func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	_, err := sgtool.GoInstall(ctx, "github.com/daixiang0/gci", version)
	if err != nil {
		return err
	}
	return nil
}
