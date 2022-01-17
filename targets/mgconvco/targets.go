package mgconvco

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

const version = "0.3.7"

// nolint: gochecknoglobals
var executable string

type Prepare mgtool.Prepare

func (Prepare) ConvcoCheck() {
	mg.Deps(prepare)
}

func ConvcoCheck(ctx context.Context, rev string) error {
	logger := mglog.Logger("convco-check")
	ctx = logr.NewContext(ctx, logger)
	mg.CtxDeps(ctx, prepare)
	logger.Info("checking...")
	return sh.RunV(executable, "check", rev)
}

func prepare(ctx context.Context) error {
	const toolName = "convco"
	binDir := filepath.Join(mgpath.Tools(), toolName, version)
	toolPath := filepath.Join(binDir, toolName)
	var hostOS string
	switch strings.Split(runtime.GOOS, "/")[0] {
	case "linux":
		hostOS = "ubuntu"
	case "darwin":
		hostOS = "macos"
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	binURL := fmt.Sprintf(
		"https://github.com/convco/convco/releases/download/v%s/convco-%s.zip",
		version,
		hostOS,
	)
	if err := mgtool.FromRemote(
		ctx,
		binURL,
		mgtool.WithDestinationDir(binDir),
		mgtool.WithUnzip(),
		mgtool.WithSkipIfFileExists(toolPath),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", toolName, err)
	}
	executable = toolPath
	return os.Chmod(executable, 0o755)
}
