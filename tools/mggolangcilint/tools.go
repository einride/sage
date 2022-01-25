package mggolangcilint

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"go.einride.tech/mage-tools/mg"
	"go.einride.tech/mage-tools/mgtool"
)

const version = "1.43.0"

// nolint: gochecknoglobals
var commandPath string

//go:embed golangci.yml
var defaultConfig string

func Command(ctx context.Context, args ...string) *exec.Cmd {
	mg.Deps(ctx, Prepare.GolangciLint)
	return mgtool.Command(ctx, commandPath, args...)
}

func RunCommand(ctx context.Context) *exec.Cmd {
	configPath := mg.FromWorkDir(".golangci.yml")
	cmd := Command(ctx)
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		configPath = mg.FromToolsDir("golangci-lint", ".golangci.yml")
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0o600); err != nil {
			panic(err)
		}
	}
	cmd.Args = append(cmd.Args, "run", "-c", configPath, "--fix")
	return cmd
}

type Prepare mgtool.Prepare

func (Prepare) GolangciLint(ctx context.Context) error {
	const binaryName = "golangci-lint"
	toolDir := mg.FromToolsDir(binaryName)
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
	if err := mgtool.FromRemote(
		ctx,
		binURL,
		mgtool.WithDestinationDir(binDir),
		mgtool.WithUntarGz(),
		mgtool.WithRenameFile(fmt.Sprintf("%s/golangci-lint", golangciLint), binaryName),
		mgtool.WithSkipIfFileExists(binary),
		mgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	commandPath = binary
	return nil
}
