package sgdocker

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
	name    = "docker"
	version = "20.10.14" // match the version used by Cloud Build
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	// Special case: use local Docker CLI when available.
	if binary, err := exec.LookPath("docker"); err == nil {
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
	if hostArch == sgtool.ARM64 {
		hostArch = "aarch64"
	}
	if hostOS == sgtool.Darwin {
		hostOS = "mac"
	}
	binDir := sg.FromToolsDir(name, version)
	binary := filepath.Join(binDir, "docker", name)
	binURL := fmt.Sprintf(
		"https://download.docker.com/%s/static/stable/%s/docker-%s.tgz",
		hostOS,
		hostArch,
		version,
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
