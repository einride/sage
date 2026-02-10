package sggopls

import (
	"context"
	"os/exec"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	name    = "gopls"
	version = "v0.20.0"
)

// Check runs `gopls check <PATH1> <PATH2>`.
// gopls only works with paths to go files.
func Check(ctx context.Context, paths []string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	args := make([]string, 0, 1+len(paths))
	args = append(args, "check")
	args = append(args, paths...)
	return sg.Command(
		ctx,
		sg.FromBinDir(name),
		args...,
	)
}

func PrepareCommand(ctx context.Context) error {
	_, err := sgtool.GoInstall(ctx, "golang.org/x/tools/gopls", version)
	return err
}
