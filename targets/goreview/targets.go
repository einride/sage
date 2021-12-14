package goreview

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-logr/logr"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgtool"
	"go.einride.tech/mage-tools/tools"
)

const version = "0.18.0"

var executable string

func Goreview(ctx context.Context) error {
	logger := mglog.Logger("goreview")
	ctx = logr.NewContext(ctx, logger)
	mg.CtxDeps(ctx, prepare)
	logger.Info("running...")
	return sh.RunV(executable, "-c", "1", "./...")
}

func prepare(ctx context.Context) error {
	const toolName = "goreview"
	toolDir := filepath.Join(tools.GetPath(), toolName)
	binDir := filepath.Join(toolDir, version, "bin")
	binary := filepath.Join(binDir, toolName)
	hostOS := strings.Title(runtime.GOOS)
	hostArch := runtime.GOARCH
	if hostArch == tools.AMD64 {
		hostArch = tools.X8664
	}
	fileName := fmt.Sprintf("goreview_%s_%s_%s", version, hostOS, hostArch)
	binURL := fmt.Sprintf(
		"https://github.com/einride/goreview/releases/download/v%s/%s.tar.gz",
		version,
		fileName,
	)
	if err := mgtool.FromRemote(
		ctx,
		binURL,
		mgtool.WithDestinationDir(binDir),
		mgtool.WithUntarGz(),
		mgtool.WithRenameFile(fmt.Sprintf("%s/goreview", fileName), toolName),
		mgtool.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", toolName, err)
	}
	executable = binary
	return nil
}
