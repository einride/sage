package mggoreview

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"go.einride.tech/mage-tools/mg"
	"go.einride.tech/mage-tools/mgtool"
)

const version = "0.18.0"

// nolint: gochecknoglobals
var commandPath string

func Command(ctx context.Context, args ...string) *exec.Cmd {
	mg.Deps(ctx, PrepareCommand)
	return mg.Command(ctx, commandPath, args...)
}

func PrepareCommand(ctx context.Context) error {
	const toolName = "goreview"
	toolDir := mg.FromToolsDir(toolName)
	binDir := filepath.Join(toolDir, version, "bin")
	binary := filepath.Join(binDir, toolName)
	hostOS := strings.Title(runtime.GOOS)
	hostArch := runtime.GOARCH
	if hostArch == mgtool.AMD64 {
		hostArch = mgtool.X8664
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
		mgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", toolName, err)
	}
	commandPath = binary
	return nil
}
