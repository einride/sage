package sgmdformat

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
	name    = "mdformat"
	syntax  = "gfm"
	version = "0.3.5"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	args = append([]string{"."}, args...)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	toolDir := sg.FromToolsDir(name, version)
	mdformat := filepath.Join(toolDir, "bin", name)
	if _, err := os.Stat(mdformat); err == nil {
		if _, err := sgtool.CreateSymlink(mdformat); err != nil {
			return err
		}
		return nil
	}
	if err := sgpython.Command(ctx, "-m", "venv", toolDir).Run(); err != nil {
		return err
	}
	pip := filepath.Join(toolDir, "bin", "pip")
	if err := sg.Command(ctx, pip, "install", "-U", "pip").Run(); err != nil {
		return err
	}
	if err := sg.Command(ctx, pip, "install", name+"-"+syntax+"=="+version).Run(); err != nil {
		return err
	}
	if _, err := sgtool.CreateSymlink(mdformat); err != nil {
		return err
	}

	return nil
}
