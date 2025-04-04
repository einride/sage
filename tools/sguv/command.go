package sguv

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
	name    = "uv"
	version = "0.6.12"

	// Runtime OS constants.
	windows = "windows"
	linux   = "linux"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	toolDir := sg.FromToolsDir(name)
	binDir := filepath.Join(toolDir, version, "bin")
	binary := filepath.Join(binDir, name)
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH

	// Map Go arch to uv arch
	switch hostArch {
	case sgtool.ARM64:
		hostArch = "aarch64"
	case sgtool.AMD64:
		hostArch = "x86_64"
	case "386":
		hostArch = "i686"
	}

	var fileExt string
	switch hostOS {
	case sgtool.Darwin:
		hostOS = "apple-darwin"
		fileExt = ".tar.gz"
	case windows:
		hostOS = "pc-windows-msvc"
		fileExt = ".zip"
	case linux:
		hostOS = "unknown-linux-gnu"
		fileExt = ".tar.gz"
	}

	filename := fmt.Sprintf("uv-%s-%s", hostArch, hostOS)
	binURL := fmt.Sprintf(
		"https://github.com/astral-sh/uv/releases/download/%s/%s%s",
		version,
		filename,
		fileExt,
	)

	options := []sgtool.Opt{
		sgtool.WithDestinationDir(binDir),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	}

	switch fileExt {
	case ".tar.gz":
		options = append(options,
			sgtool.WithUntarGz(),
			sgtool.WithRenameFile(fmt.Sprintf("%s/uv", filename), name),
		)
	case ".zip":
		options = append(options,
			sgtool.WithUnzip(),
			sgtool.WithRenameFile(fmt.Sprintf("%s/uv.exe", filename), name),
		)
	}

	if err := sgtool.FromRemote(ctx, binURL, options...); err != nil {
		return fmt.Errorf("unable to download %s: %w", name, err)
	}

	return nil
}
