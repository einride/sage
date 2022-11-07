package sggomockgen

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
	name    = "mockgen"
	version = "1.6.0"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	toolDir := sg.FromToolsDir(name)
	dlDir := filepath.Join(toolDir, version)
	binary := filepath.Join(dlDir, "bin", name)
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	fileName := fmt.Sprintf("mock_%s_%s_%s", version, hostOS, hostArch)
	binURL := fmt.Sprintf(
		"https://github.com/golang/mock/releases/download/v%s/%s.tar.gz",
		version,
		fileName,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(dlDir),
		sgtool.WithUntarGz(),
		sgtool.WithRenameFile(fmt.Sprintf("%s/mockgen", fileName), filepath.Join("bin", "mockgen")),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", name, err)
	}
	if err := os.RemoveAll(filepath.Join(dlDir, fileName)); err != nil {
		return err
	}
	return nil
}
