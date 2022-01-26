package sggolangcilint

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const version = "1.43.0"

// nolint: gochecknoglobals
var commandPath string

//go:embed golangci.yml
var defaultConfig string

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, commandPath, args...)
}

func RunCommand(ctx context.Context) *exec.Cmd {
	configPath := sg.FromWorkDir(".golangci.yml")
	cmd := Command(ctx)
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		configPath = sg.FromToolsDir("golangci-lint", ".golangci.yml")
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0o600); err != nil {
			panic(err)
		}
	}
	cmd.Args = append(cmd.Args, "run", "-c", configPath, "--fix")
	return cmd
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
