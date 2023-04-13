package sgsqlfluff

import (
	"context"
	_ "embed"
	"errors"
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
	dbtBigQueryVersion = "1.4.0"
)

//go:embed .sqlfluff
var DefaultConfig []byte

// Command runs the sqlfluff CLI.
// If templater = dbt is used in the .sqlfluff file, the working directory of the cmd needs to be set to the same
// directory as where the .sqlfluff file is, and no nested .sqlfluff files can be used.
// Read more here: https://docs.sqlfluff.com/en/stable/configuration.html#known-caveats
func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func defaultConfigPath() string {
	return sg.FromToolsDir(name, ".sqlfluff")
}

func CommandInDirectory(ctx context.Context, directory string, args ...string) *exec.Cmd {
	configPath := filepath.Join(directory, "dbt/.sqlfluff")
	if _, err := os.Lstat(configPath); errors.Is(err, os.ErrNotExist) {
		configPath = defaultConfigPath()
	}
	cmdArgs := append(args, []string{"--config", configPath}...)
	cmd := Command(ctx, cmdArgs...)
	cmd.Dir = directory
	return cmd
}

func Run(ctx context.Context, args ...string) error {
	var commands []*exec.Cmd
	path, err := os.Getwd()
	if err != nil {
		return err
	}
	cmd := CommandInDirectory(ctx, filepath.Dir(path), args...)
	commands = append(commands, cmd)
	return cmd.Run()
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
	configPath := defaultConfigPath()
	sg.Logger(ctx).Println(configPath)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(configPath, DefaultConfig, 0o600)
}
