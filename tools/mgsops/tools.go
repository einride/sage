package mgsops

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/go-logr/logr"
	"github.com/magefile/mage/mg"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

const version = "3.7.1"

// nolint: gochecknoglobals
var commandPath string

func Command(ctx context.Context, args ...string) *exec.Cmd {
	ctx = logr.NewContext(ctx, mglog.Logger("sops"))
	mg.CtxDeps(ctx, Prepare.Sops)
	return mgtool.Command(commandPath, args...)
}

type Prepare mgtool.Prepare

func (Prepare) Sops(ctx context.Context) error {
	const binaryName = "sops"
	binDir := mgpath.FromTools(binaryName, version)
	binary := filepath.Join(binDir, binaryName)

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
		mgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	commandPath = binary
	return nil
}
