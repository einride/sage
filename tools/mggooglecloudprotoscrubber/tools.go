package mggooglecloudprotoscrubber

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/magefile/mage/mg"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

const version = "1.1.0"

// nolint: gochecknoglobals
var commandPath string

type Prepare mgtool.Prepare

func Command(ctx context.Context, args ...string) *exec.Cmd {
	mg.CtxDeps(ctx, Prepare.GoogleCloudProtoScrubber)
	return mgtool.Command(ctx, commandPath, args...)
}

func (Prepare) GoogleCloudProtoScrubber(ctx context.Context) error {
	const binaryName = "google-cloud-proto-scrubber"
	binDir := mgpath.FromToolsDir(binaryName, version)
	binary := filepath.Join(binDir, binaryName)
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostArch == mgtool.AMD64 {
		hostArch = mgtool.X8664
	}
	binURL := fmt.Sprintf(
		"https://github.com/einride/google-cloud-proto-scrubber"+
			"/releases/download/v%s/google-cloud-proto-scrubber_%s_%s_%s.tar.gz",
		version,
		version,
		hostOS,
		hostArch,
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
	if err := os.Chmod(binary, 0o755); err != nil {
		return fmt.Errorf("unable to make %s command: %w", binaryName, err)
	}
	commandPath = binary
	return nil
}
