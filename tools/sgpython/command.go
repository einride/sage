package sgpython

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
	"go.einride.tech/sage/tools/sguv"
)

const (
	name    = "python"
	version = "3.10" // use uv's Python version specifier
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	sg.Deps(ctx, sguv.PrepareCommand)
	symlink := sg.FromBinDir(name)
	// Check if symlink already exists and points to a valid Python
	if target, err := os.Readlink(symlink); err == nil {
		if _, err := os.Stat(target); err == nil {
			return nil
		}
	}
	// Install Python using uv
	sg.Logger(ctx).Printf("installing Python %s using uv...", version)
	if err := sguv.Command(ctx, "python", "install", version).Run(); err != nil {
		return err
	}
	// Find the installed Python path
	pythonPath, err := findUvPython(ctx, version)
	if err != nil {
		return err
	}
	// Create symlink
	if _, err := sgtool.CreateSymlink(pythonPath); err != nil {
		return err
	}
	return nil
}

func findUvPython(ctx context.Context, version string) (string, error) {
	cmd := sguv.Command(ctx, "python", "find", version)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return filepath.Clean(strings.TrimSpace(stdout.String())), nil
}
