package sgpython

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
	"go.einride.tech/sage/tools/sggit"
)

const (
	name         = "python"
	version      = "3.10.6" // parity with Ubuntu 22.04 LTS
	pyenvVersion = "2.3.18"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	toolDir := sg.FromToolsDir(name, version)
	pyenvDir := filepath.Join(toolDir, "pyenv")
	binDir := filepath.Join(pyenvDir, "versions", version, "bin")
	pythonFromPyenv := filepath.Join(binDir, "python")
	if _, err := os.Stat(pythonFromPyenv); err == nil {
		if _, err := sgtool.CreateSymlink(pythonFromPyenv); err != nil {
			return err
		}
		return nil
	} else if systemPython3, err := exec.LookPath("python3"); err == nil {
		// Special case: Avoid building from source if we already have Python 3 on the system.
		sg.Logger(ctx).Printf("using system Python: %s", systemPython3)
		symlink := filepath.Join(sg.FromBinDir(), name)
		if err := os.MkdirAll(sg.FromBinDir(), 0o755); err != nil {
			return err
		}
		if _, err := os.Lstat(symlink); err == nil {
			if err := os.Remove(symlink); err != nil {
				return err
			}
		}
		return os.Symlink(systemPython3, symlink)
	}
	if err := os.RemoveAll(pyenvDir); err != nil {
		return err
	}
	if err := sggit.Command(
		ctx,
		"clone",
		"--depth",
		"1",
		"--branch",
		"v"+pyenvVersion,
		"https://github.com/pyenv/pyenv.git",
		pyenvDir,
	).Run(); err != nil {
		return err
	}
	cmd := sg.Command(ctx, "bin/pyenv", "install", version)
	cmd.Dir = pyenvDir
	cmd.Env = append(cmd.Env, fmt.Sprintf("PYENV_ROOT=%s", pyenvDir))
	if err := cmd.Run(); err != nil {
		return err
	}
	if _, err := sgtool.CreateSymlink(pythonFromPyenv); err != nil {
		return err
	}
	return nil
}
