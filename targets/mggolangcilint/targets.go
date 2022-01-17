package mggolangcilint

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/go-logr/logr"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

const version = "1.43.0"

// nolint: gochecknoglobals
var executable string

//go:embed golangci.yml
var defaultConfig string

type Prepare mgtool.Prepare

func (Prepare) GolangciLint(ctx context.Context) error {
	return prepare(ctx)
}

func GolangciLint(ctx context.Context) error {
	ctx = logr.NewContext(ctx, mglog.Logger("golangci-lint"))
	mg.CtxDeps(ctx, mg.F(prepare))
	logr.FromContextOrDiscard(ctx).Info("running...")
	configPath := mgpath.FromWorkDir(".golangci.yml")
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		configPath = filepath.Join(mgpath.Tools(), "golangci-lint", ".golangci.yml")
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0o600); err != nil {
			return err
		}
	}
	return sh.RunV(executable, "run", "-c", configPath, "--fix")
}

func prepare(ctx context.Context) error {
	const binaryName = "golangci-lint"
	toolDir := filepath.Join(mgpath.Tools(), binaryName)
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
	executable = binary
	return nil
}
