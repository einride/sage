package sgdbt

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
	"go.einride.tech/sage/tools/sguv"
)

const (
	name = "dbt"
	// renovate: datasource=pypi depName=dbt-bigquery
	bigqueryPackageVersion = "1.6.0"
	pythonVersion          = "3.11" // dbt-bigquery 1.6.0 requires Python <3.12 (distutils dependency)
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	toolDir := sg.FromToolsDir(name)
	// Include pythonVersion in path so changing Python version invalidates cached venvs
	venvDir := filepath.Join(toolDir, "venv", bigqueryPackageVersion, pythonVersion)
	binDir := filepath.Join(venvDir, "bin")
	binary := filepath.Join(binDir, name)
	if _, err := os.Stat(binary); err == nil {
		if _, err := sgtool.CreateSymlink(binary); err != nil {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return err
	}
	sg.Logger(ctx).Println("installing packages...")
	if err := sguv.CreateVenv(ctx, venvDir, pythonVersion); err != nil {
		return err
	}
	if err := sguv.PipInstall(ctx, venvDir, fmt.Sprintf("dbt-bigquery==%s", bigqueryPackageVersion)); err != nil {
		return err
	}
	if _, err := sgtool.CreateSymlink(binary); err != nil {
		return err
	}
	return nil
}
