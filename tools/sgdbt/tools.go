package sgdbt

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
	name                   = "dbt"
	bigqueryPackageVersion = "1.3.0"
	pytzVersion            = "2022.7.1"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	toolDir := sg.FromToolsDir(name)
	venvDir := filepath.Join(toolDir, "venv", bigqueryPackageVersion)
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
		fmt.Sprintf("dbt-bigquery==%s", bigqueryPackageVersion),
	).Run(); err != nil {
		return err
	}
	// install pytz since dbt needs it but doesn't list it as a dependency
	// https://github.com/dbt-labs/dbt-core/issues/7075
	if err := sg.Command(
		ctx,
		pip,
		"install",
		fmt.Sprintf("pytz==%s", pytzVersion),
	).Run(); err != nil {
		return err
	}
	if _, err := sgtool.CreateSymlink(binary); err != nil {
		return err
	}
	return nil
}
