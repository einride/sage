package mgcocogitto

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-logr/logr"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

const version = "4.0.1"

// nolint: gochecknoglobals
var executable string

func CogCheck(ctx context.Context) error {
	logger := mglog.Logger("cog-check")
	ctx = logr.NewContext(ctx, logger)
	mg.CtxDeps(ctx, prepare)
	logger.Info("checking git commits...")
	return sh.RunV(executable, "check", "--from-latest-tag")
}

func prepare(ctx context.Context) error {
	const toolName = "cocogitto"
	binDir := filepath.Join(mgpath.Tools(), toolName, version)
	toolPath := filepath.Join(binDir, "cog")
	var archiveName string
	switch strings.Split(runtime.GOOS, "/")[0] {
	case "linux":
		archiveName = fmt.Sprintf("cocogitto-%s-x86_64-unknown-linux-musl.tar.gz", version)
	case "darwin":
		archiveName = fmt.Sprintf("cocogitto-%s-x86_64-osx.tar.gz", version)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	binURL := fmt.Sprintf(
		"https://github.com/cocogitto/cocogitto/releases/download/%s/%s",
		version,
		archiveName,
	)
	if err := mgtool.FromRemote(
		ctx,
		binURL,
		mgtool.WithDestinationDir(binDir),
		mgtool.WithUntarGz(),
		mgtool.WithSkipIfFileExists(toolPath),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", toolName, err)
	}
	executable = toolPath
	return os.Chmod(executable, 0o755)
}
