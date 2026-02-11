package sgrumdl

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	name    = "rumdl"
	version = "0.1.18"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	binDir := sg.FromToolsDir(name, version)
	binary := filepath.Join(binDir, name)
	hostOS := fmt.Sprintf("unknown-%s", runtime.GOOS)
	hostArch := runtime.GOARCH
	if hostArch == sgtool.AMD64 {
		hostArch = sgtool.X8664
	}
	if hostOS == sgtool.Darwin && hostArch == sgtool.ARM64 {
		hostArch = sgtool.X8664
	}
	binURL := fmt.Sprintf(
		"https://github.com/rvben/rumdl/releases/download/v%s/rumdl-v%s-%s-%s-gnu.tar.gz",
		version,
		version,
		hostArch,
		hostOS,
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
	return os.Chmod(binary, 0o755)
}
