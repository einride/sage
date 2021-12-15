package mgsops

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/go-logr/logr"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgtool"
)

const version = "3.7.1"

var executable string

func Sops(ctx context.Context, file string) error {
	logger := mglog.Logger("sops")
	ctx = logr.NewContext(ctx, logger)
	mg.CtxDeps(ctx, sops)
	logger.Info("running...")
	return sh.RunV(executable, file)
}

func sops(ctx context.Context) error {
	const binaryName = "sops"
	binDir := filepath.Join(mgtool.GetPath(), binaryName, version)
	binary := filepath.Join(binDir, binaryName)
	executable = binary

	hostOS := runtime.GOOS

	binURL := fmt.Sprintf(
		"https://github.com/mozilla/sops/releases/download/v%s/sops-v%s.%s",
		version,
		version,
		hostOS,
	)

	if err := mgtool.FromRemote(
		ctx,
		binURL,
		mgtool.WithDestinationDir(binDir),
		mgtool.WithRenameFile("", binaryName),
		mgtool.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	return nil
}
