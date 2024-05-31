package sgpoetry

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
	"go.einride.tech/sage/tools/sgpython"
)

const (
	name    = "poetry"
	version = "1.8.3"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	toolDir := sg.FromToolsDir(name, version)
	poetry := filepath.Join(toolDir, "bin", name)
	if _, err := os.Stat(poetry); err == nil {
		if _, err := sgtool.CreateSymlink(poetry); err != nil {
			return err
		}
		return nil
	}
	// See: https://python-poetry.org/docs/#installing-manually
	if err := sgpython.Command(ctx, "-m", "venv", toolDir).Run(); err != nil {
		return err
	}
	pip := filepath.Join(toolDir, "bin", "pip")
	if err := sg.Command(ctx, pip, "install", "-U", "pip", "setuptools").Run(); err != nil {
		return err
	}
	if err := sg.Command(ctx, pip, "install", name+"=="+version).Run(); err != nil {
		return err
	}
	if _, err := sgtool.CreateSymlink(poetry); err != nil {
		return err
	}
	return nil
}
