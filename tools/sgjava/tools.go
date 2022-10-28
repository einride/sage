package sgjava

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	name    = "java"
	version = "17.0.5+8"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	toolDir := sg.FromToolsDir(name)
	binDir := filepath.Join(toolDir, version, "bin")
	binary := filepath.Join(binDir, "jdk-"+version, "bin", name)
	if err := sgtool.FromRemote(
		ctx,
		fmt.Sprintf(
			"https://github.com/adoptium/temurin17-binaries/releases/download/jdk-%s/OpenJDK17U-jdk_x64_linux_hotspot_%s.tar.gz",
			version,
			strings.Replace(version, "+", "_", 1),
		),
		sgtool.WithDestinationDir(binDir),
		sgtool.WithUntarGz(),
		sgtool.WithSkipIfFileExists(filepath.Join(binDir, "jdk-"+version, "bin", name)),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", name, err)
	}
	return nil
}
