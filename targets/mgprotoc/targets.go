package mgprotoc

import (
	"context"
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

const version = "3.15.7"

// nolint: gochecknoglobals
var executable string

func Protoc(ctx context.Context, args ...string) error {
	logger := mglog.Logger("protoc")
	ctx = logr.NewContext(ctx, logger)
	mg.CtxDeps(ctx, prepare)
	logger.Info("running...")
	return sh.RunV(executable, args...)
}

func prepare(ctx context.Context) error {
	const binaryName = "protoc"

	binDir := filepath.Join(mgtool.GetPath(), binaryName, version)
	binary := filepath.Join(binDir, "bin", binaryName)

	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostArch == mgtool.AMD64 {
		hostArch = mgtool.X8664
	}

	binURL := fmt.Sprintf(
		"https://github.com/protocolbuffers/protobuf/releases/download/v%s/protoc-%s-%s-%s.zip",
		version,
		version,
		hostOS,
		hostArch,
	)

	if err := mgtool.FromRemote(
		ctx,
		binURL,
		mgtool.WithDestinationDir(binDir),
		mgtool.WithUnzip(),
		mgtool.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}

	if err := os.RemoveAll(filepath.Join(binDir, "include")); err != nil {
		return err
	}
	executable = binary
	return nil
}
