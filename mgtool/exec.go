package mgtool

import (
	"context"
	"os"
	"os/exec"

	"go.einride.tech/mage-tools/mgpath"
)

// Command should be used when returning exec.Cmd from tools to set opinionated standard fields.
func Command(_ context.Context, path string, args ...string) *exec.Cmd {
	cmd := exec.Command(path)
	cmd.Args = append(cmd.Args, args...)
	cmd.Dir = mgpath.FromGitRoot(".")
	cmd.Env = os.Environ()
	// TODO: Pipe stdout/stderr through the current context logger to get tagged output.
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd
}
