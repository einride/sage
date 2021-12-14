package mgbuf

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/go-logr/logr"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mgtool"
	"go.einride.tech/mage-tools/tools"
)

const version = "0.55.0"

var executable string

func BufLint(ctx context.Context) error {
	logger := logr.FromContextOrDiscard(ctx).WithName("buf-lint")
	ctx = logr.NewContext(ctx, logger)
	mg.CtxDeps(ctx, prepare)
	return sh.RunV(executable, "lint")
}

func BufGenerate(ctx context.Context) error {
	logger := logr.FromContextOrDiscard(ctx).WithName("buf-generate")
	ctx = logr.NewContext(ctx, logger)
	mg.CtxDeps(ctx, prepare)
	return sh.RunV(executable, "generate")
}

func Buf(ctx context.Context, args ...string) error {
	logger := logr.FromContextOrDiscard(ctx).WithName("buf")
	ctx = logr.NewContext(ctx, logger)
	mg.CtxDeps(ctx, prepare)
	return sh.RunV(executable, args...)
}

func prepare(ctx context.Context) error {
	const binaryName = "buf"
	binDir := filepath.Join(tools.GetPath(), binaryName, version, "bin")
	binary := filepath.Join(binDir, binaryName)
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostArch == tools.AMD64 {
		hostArch = tools.X8664
	}
	binURL := fmt.Sprintf(
		"https://github.com/bufbuild/buf/releases/download/v%s/buf-%s-%s",
		version,
		hostOS,
		hostArch,
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

	executable = binary
	return nil
}