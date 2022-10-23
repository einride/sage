package sggovulncheck

import (
	"context"
	"os/exec"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	name    = "govulncheck"
	version = "latest"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	_, err := sgtool.GoInstall(
		ctx,
		"golang.org/x/vuln/cmd/govulncheck",
		version,
	)
	return err
}
