package sgyq

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
	name    = "yq"
	version = "4.34.1"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	binDir := sg.FromToolsDir(name, version)
	binary := filepath.Join(binDir, name)
	binURL := fmt.Sprintf(
		"https://github.com/mikefarah/yq/releases/download/v%s/yq_%s_%s.tar.gz",
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
		sgtool.WithRenameFile(fmt.Sprintf("./yq_%s_%s", hostOS, hostArch), name),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", name, err)
	}
	return nil
}
