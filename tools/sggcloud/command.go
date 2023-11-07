package sggcloud

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	name    = "gcloud"
	version = "453.0.0"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	// Special case: use local gcloud CLI when available.
	if binary, err := exec.LookPath("gcloud"); err == nil {
		if _, err := sgtool.CreateSymlink(binary); err != nil {
			return err
		}
		return nil
	}
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostArch == sgtool.AMD64 {
		hostArch = sgtool.X8664
	}
	if hostOS == sgtool.Darwin && hostArch == sgtool.ARM64 {
		hostArch = "arm"
	}
	binDir := sg.FromToolsDir(name, version)
	binary := filepath.Join(binDir, "google-cloud-sdk", "bin", name)
	binURL := fmt.Sprintf(
		"https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-%s-%s-%s.tar.gz",
		version,
		hostOS,
		hostArch,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(binDir),
		sgtool.WithUntarGz(),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", name, err)
	}
	return nil
}
