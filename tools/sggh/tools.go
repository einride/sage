package sggh

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
	name    = "gh"
	version = "2.67.0"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	hostOS := runtime.GOOS
	ext := "tar.gz"
	if hostOS == sgtool.Darwin {
		hostOS = "macOS"
		ext = "zip"
	}
	hostArch := runtime.GOARCH
	binDir := sg.FromToolsDir(name, version)
	binary := filepath.Join(binDir, name)
	binURL := fmt.Sprintf(
		"https://github.com/cli/cli/releases/download/v%s/gh_%s_%s_%s.%s",
		version,
		version,
		hostOS,
		hostArch,
		ext,
	)
	opts := []sgtool.Opt{
		sgtool.WithDestinationDir(binDir),
		sgtool.WithRenameFile(fmt.Sprintf("gh_%s_%s_%s/bin/gh", version, hostOS, hostArch), name),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	}
	if hostOS == "macOS" {
		opts = append(opts, sgtool.WithUnzip())
	} else {
		opts = append(opts, sgtool.WithUntarGz())
	}
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		opts...,
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", name, err)
	}
	return nil
}
