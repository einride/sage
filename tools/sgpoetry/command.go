package sgpoetry

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
	"go.einride.tech/sage/tools/sguv"
)

const (
	name          = "poetry"
	version       = "2.1.2"
	pythonVersion = "3.12"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	// Include pythonVersion in path so changing Python version invalidates cached venvs
	toolDir := sg.FromToolsDir(name, version, pythonVersion)
	poetry := filepath.Join(toolDir, "bin", name)
	if _, err := os.Stat(poetry); err == nil {
		if _, err := sgtool.CreateSymlink(poetry); err != nil {
			return err
		}
		return nil
	}
	// See: https://python-poetry.org/docs/#installing-manually
	if err := sguv.CreateVenv(ctx, toolDir, pythonVersion); err != nil {
		return err
	}
	if err := sguv.PipInstall(ctx, toolDir, name+"=="+version); err != nil {
		return err
	}
	if _, err := sgtool.CreateSymlink(poetry); err != nil {
		return err
	}
	return nil
}
