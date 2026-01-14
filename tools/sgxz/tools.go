package sgxz

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
	// renovate: datasource=github-releases depName=therootcompany/xz-static
	version = "5.2.5"
	name    = "xz"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	const binaryName = "xz"
	toolDir := sg.FromToolsDir(binaryName)
	binDir := filepath.Join(toolDir, version, "bin")
	binary := filepath.Join(binDir, binaryName)
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostArch == sgtool.AMD64 {
		hostArch = sgtool.X8664
	}
	if hostOS == sgtool.Darwin && hostArch == sgtool.ARM64 {
		hostArch = sgtool.X8664
	}
	xz := fmt.Sprintf("%s-%s-%s-%s", binaryName, version, hostOS, hostArch)
	binURL := fmt.Sprintf(
		"https://github.com/therootcompany/xz-static/releases/download/v%s/%s.tar.gz",
		version,
		xz,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(binDir),
		sgtool.WithUntar(),
		sgtool.WithRenameFile(fmt.Sprintf("./%s/xz", xz), binaryName),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	return nil
}
