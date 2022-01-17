package mggooglecloudprotoscrubber

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
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

const version = "1.1.0"

// nolint: gochecknoglobals
var executable string

type Prepare mgtool.Prepare

func (Prepare) GoogleCloudProtoScrubber(ctx context.Context) error {
	return prepare(ctx)
}

func GoogleCloudProtoScrubber(ctx context.Context, fileDescriptorPath string) error {
	ctx = logr.NewContext(ctx, mglog.Logger("google-cloud-proto-scrubber"))
	mg.CtxDeps(ctx, mg.F(prepare))
	logr.FromContextOrDiscard(ctx).Info("scrubbing API descriptor...")
	return sh.RunV(executable, "-f", fileDescriptorPath)
}

func prepare(ctx context.Context) error {
	const binaryName = "google-cloud-proto-scrubber"
	binDir := filepath.Join(mgpath.Tools(), binaryName, version)
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
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	if err := os.Chmod(binary, 0o755); err != nil {
		return fmt.Errorf("unable to make %s executable: %w", binaryName, err)
	}
	executable = binary
	return nil
}
