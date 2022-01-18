package mggosemanticrelease

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

// nolint: gochecknoglobals
var executable string

type Prepare mgtool.Prepare

func (Prepare) GoSemanticRelease(ctx context.Context) error {
	return prepare(ctx)
}

func GoSemanticRelease(ctx context.Context, args ...string) error {
	logger := mglog.Logger("go-semantic-release")
	ctx = logr.NewContext(ctx, logger)
	mg.CtxDeps(ctx, prepare)
	return sh.RunV(executable, args...)
}

func prepare(ctx context.Context) error {
	const (
		binaryName = "gosemantic-release"
		version    = "2.18.0"
	)
	binDir := filepath.Join(mgpath.Tools(), binaryName, version)
	binary := filepath.Join(binDir, binaryName)
	var hostOS string
	switch strings.Split(runtime.GOOS, "/")[0] {
	case "linux":
		hostOS = "linux"
	case "darwin":
		hostOS = "darwin"
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	binURL := fmt.Sprintf(
		"https://github.com/go-semantic-release/semantic-release/releases/download/v%s/semantic-release_v%s_%s_amd64",
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
		mgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	executable = binary
	return os.Chmod(binary, 0o755)
}
