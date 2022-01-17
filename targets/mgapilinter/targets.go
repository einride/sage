package mgapilinter

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/go-logr/logr"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

const version = "1.29.3"

// nolint: gochecknoglobals
var executable string

type Prepare mgtool.Prepare

func (Prepare) APILinter(ctx context.Context) error {
	return prepare(ctx)
}

func APILinterLint(ctx context.Context, args ...string) error {
	logger := mglog.Logger("api-linter-lint")
	ctx = logr.NewContext(ctx, logger)
	mg.CtxDeps(ctx, prepare)
	logger.Info("linting gRPC APIs...")
	return sh.RunV(executable, args...)
}

func prepare(ctx context.Context) error {
	const binaryName = "api-linter"

	hostOS := runtime.GOOS

	binDir := filepath.Join(mgpath.Tools(), binaryName, version, "bin")
	binary := filepath.Join(binDir, binaryName)

	binURL := fmt.Sprintf(
		"https://github.com/googleapis/api-linter/releases/download/v%s/api-linter-%s-%s-amd64.tar.gz",
		version,
		version,
		hostOS,
	)

	if err := mgtool.FromRemote(
		ctx,
		binURL,
		mgtool.WithDestinationDir(binDir),
		mgtool.WithUntarGz(),
		mgtool.WithSkipIfFileExists(binary),
		mgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	executable = binary
	return nil
}
