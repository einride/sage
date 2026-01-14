package sgsqlfluff

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
	name = "sqlfluff"
	// renovate: datasource=pypi depName=sqlfluff
	version            = "2.1.4"
	dbtBigQueryVersion = "1.6.0"
	pythonVersion      = "3.11" // dbt-bigquery 1.6.0 requires Python <3.12 (distutils dependency)
)

// Command runs the sqlfluff CLI.
// If templater = dbt is used in the .sqlfluff file, the working directory of the cmd needs to be set to the same
// directory as where the .sqlfluff file is, and no nested .sqlfluff files can be used.
// Read more here: https://docs.sqlfluff.com/en/stable/configuration.html#known-caveats
func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	toolDir := sg.FromToolsDir(name)
	// Include pythonVersion in path so changing Python version invalidates cached venvs
	venvDir := filepath.Join(toolDir, "venv", version, pythonVersion)
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
	// install sqlfluff, dbt-bigquery, and sqlfluff-templater-dbt to enable using
	// templater = dbt in .sqlfluff config file
	if err := sguv.PipInstall(
		ctx,
		venvDir,
		fmt.Sprintf("sqlfluff==%s", version),
		fmt.Sprintf("dbt-bigquery==%s", dbtBigQueryVersion),
		fmt.Sprintf("sqlfluff-templater-dbt==%s", version),
	); err != nil {
		return err
	}
	if _, err := sgtool.CreateSymlink(binary); err != nil {
		return err
	}
	return nil
}
