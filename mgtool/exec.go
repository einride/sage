package mgtool

import (
	"context"
	"fmt"
	"go.einride.tech/mage-tools/mg"
	"os"
	"os/exec"
	"strings"
)

// Command should be used when returning exec.Cmd from tools to set opinionated standard fields.
func Command(_ context.Context, path string, args ...string) *exec.Cmd {
	// TODO: use exec.CommandContext when we have determined there are no side-effects.
	cmd := exec.Command(path)
	cmd.Args = append(cmd.Args, args...)
	cmd.Dir = mg.FromGitRoot(".")
	cmd.Env = os.Environ()
	// TODO: Pipe stdout/stderr through the current context logger to get tagged output.
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd
}

// Output runs the given command, and returns all output from stdout in a neatly, trimmed manner,
// panicking if an error occurs.
func Output(cmd *exec.Cmd) string {
	cmd.Stdout = nil
	output, err := cmd.Output()
	if err != nil {
		panic(fmt.Sprintf("%s failed: %v", cmd.Path, err))
	}
	return strings.TrimSpace(string(output))
}
