package mgtool

import (
	"os"
	"os/exec"

	"go.einride.tech/mage-tools/mgpath"
)

// Command should be used when returning exec.Cmd from tools to set opinionated standard fields.
func Command(path string, args ...string) *exec.Cmd {
	cmd := exec.Command(path)
	cmd.Args = append(cmd.Args, args...)
	cmd.Dir = mgpath.FromGitRoot(".")
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd
}
