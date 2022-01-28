package sggolangcilint

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const version = "1.44.0"

// nolint: gochecknoglobals
var commandPath string

//go:embed golangci.yml
var defaultConfig []byte

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, commandPath, args...)
}

// Run GolangCI-Lint in every Go module from the root of the current git repo.
func Run(ctx context.Context, args ...string) error {
	defaultConfigPath := sg.FromToolsDir("golangci-lint", ".golangci.yml")
	if err := os.MkdirAll(filepath.Dir(defaultConfigPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(defaultConfigPath, defaultConfig, 0o600); err != nil {
		return err
	}
	var commands []*exec.Cmd
	if err := filepath.WalkDir(sg.FromGitRoot(), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "go.mod" {
			return nil
		}
		configPath := filepath.Join(filepath.Dir(path), ".golangci.yml")
		if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
			configPath = defaultConfigPath
		}
		pathPrefix, err := filepath.Rel(sg.FromGitRoot(), filepath.Dir(path))
		if err != nil {
			return err
		}
		cmd := Command(ctx, append([]string{"run", "-c", configPath, "--path-prefix", pathPrefix}, args...)...)
		cmd.Dir = filepath.Dir(path)
		commands = append(commands, cmd)
		return cmd.Start()
	}); err != nil {
		return err
	}
	for _, cmd := range commands {
		if err := cmd.Wait(); err != nil {
			return err
		}
	}
	return nil
}

func PrepareCommand(ctx context.Context) error {
	const binaryName = "golangci-lint"
	toolDir := sg.FromToolsDir(binaryName)
	binDir := filepath.Join(toolDir, version, "bin")
	binary := filepath.Join(binDir, binaryName)
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	golangciLint := fmt.Sprintf("golangci-lint-%s-%s-%s", version, hostOS, hostArch)
	binURL := fmt.Sprintf(
		"https://github.com/golangci/golangci-lint/releases/download/v%s/%s.tar.gz",
		version,
		golangciLint,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(binDir),
		sgtool.WithUntarGz(),
		sgtool.WithRenameFile(fmt.Sprintf("%s/golangci-lint", golangciLint), binaryName),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	commandPath = binary
	return nil
}