package sgspanneraipgo

import (
	"context"
	"os/exec"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	name    = "spanner-aip"
	version = "v0.61.0"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	_, err := sgtool.GoInstall(ctx, "go.einride.tech/spanner-aip", version)
	return err
}
