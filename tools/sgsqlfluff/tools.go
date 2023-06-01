package sgsqlfluff

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	name               = "sqlfluff"
	version            = "2.0.0a4"
	dbtBigQueryVersion = "1.5.1"
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
	venvDir := filepath.Join(toolDir, "venv", version)
	binDir := filepath.Join(venvDir, "bin")
	binary := filepath.Join(binDir, name)
	if _, err := os.Stat(binary); err == nil {
		if _, err := sgtool.CreateSymlink(binary); err != nil {
			return err
		}
		return nil
	}
	pip := filepath.Join(binDir, "pip")
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return err
	}
	sg.Logger(ctx).Println("installing packages...")
	if err := sg.Command(
		ctx,
		"python3",
		"-m",
		"venv",
		venvDir,
	).Run(); err != nil {
		return err
	}
	if err := sg.Command(
		ctx,
		pip,
		"install",
		fmt.Sprintf("sqlfluff==%s", version),
	).Run(); err != nil {
		return err
	}
	// install dbt-bigquery and sqlfluff-templater-dbt to enable using
	// templater = dbt in .sqlfluff config file
	if err := sg.Command(
		ctx,
		pip,
		"install",
		fmt.Sprintf("dbt-bigquery==%s", dbtBigQueryVersion),
	).Run(); err != nil {
		return err
	}
	if err := sg.Command(
		ctx,
		pip,
		"install",
		fmt.Sprintf("sqlfluff-templater-dbt==%s", version),
	).Run(); err != nil {
		return err
	}
	if _, err := sgtool.CreateSymlink(binary); err != nil {
		return err
	}
	return nil
}
