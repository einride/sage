package mggolangcilint

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/go-logr/logr"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgtool"
)

const version = "1.42.1"

var executable string

const defaultConfig = `run:
  timeout: 10m
  skip-dirs:
    - gen

linters:
  enable-all: true
  disable:
    - dupl # allow duplication
    - funlen # allow long functions
    - gomnd # allow some magic numbers
    - wsl # unwanted amount of whitespace
    - godox # allow TODOs
    - interfacer # deprecated by the author for having too many false positives
    - gocognit # allow higher cognitive complexity
    - testpackage # unwanted convention
    - nestif # allow deep nesting
    - unparam # allow constant parameters
    - goerr113 # allow "dynamic" errors
    - nlreturn # don't enforce newline before return
    - paralleltest # TODO: fix issues and enable
    - exhaustivestruct # don't require exhaustive struct fields
    - wrapcheck # don't require wrapping everywhere
`

func GolangciLint(ctx context.Context) error {
	ctx = logr.NewContext(ctx, mglog.Logger("golangci-lint"))
	mg.CtxDeps(ctx, mg.F(prepare))
	logr.FromContextOrDiscard(ctx).Info("running...")
	configPath := filepath.Join(mgtool.GetCWDPath(".golangci.yml"))
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		configPath = filepath.Join(mgtool.GetPath(), "golangci-lint", ".golangci.yml")
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0o644); err != nil {
			return err
		}
	}
	return sh.RunV(executable, "run", "-c", configPath)
}

func prepare(ctx context.Context) error {
	const binaryName = "golangci-lint"
	toolDir := filepath.Join(mgtool.GetPath(), binaryName)
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
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	executable = binary
	return nil
}
