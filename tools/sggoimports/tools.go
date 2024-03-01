package gofumpt

import (
	"context"
	"os/exec"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	name    = "goimports"
	version = "v0.18.0"
)

// Command returns an [*exec.Cmd] for golines.
func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	_, err := sgtool.GoInstall(ctx, "golang.org/x/tools/cmd/goimports", version)
	if err != nil {
		return err
	}
	return nil
}
