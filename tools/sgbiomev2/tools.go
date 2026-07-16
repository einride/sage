package sgbiomev2

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"slices"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	toolName = "biome"
	version  = "2.5.4"
)

func Format(ctx context.Context, flags []string, paths ...string) error {
	sg.Deps(ctx, prepareCommand)
	binDir := sg.FromToolsDir(toolName, version)
	execDir := filepath.Join(binDir, toolName)
	args := slices.Concat([]string{"format"}, flags, paths)

	if err := sg.Command(
		ctx,
		execDir,
		args...,
	).Run(); err != nil {
		return fmt.Errorf("running biome %v", err)
	}

	return nil
}

func prepareCommand(ctx context.Context) error {
	binDir := sg.FromToolsDir(toolName, version)
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	toolNameWithArch := fmt.Sprintf("%s-%s-%s", toolName, runtime.GOOS, arch)
	binary := filepath.Join(binDir, toolName, toolNameWithArch)
	binURL := "https://github.com/biomejs/biome/releases/download/%40biomejs%2Fbiome%40" +
		fmt.Sprintf("%s/%s", version, toolNameWithArch)

	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(binDir),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
		sgtool.WithRenameFile(toolNameWithArch, toolName),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", toolName, err)
	}

	return nil
}
