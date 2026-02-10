package sgbiome

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	toolName = "biome"
	version  = "1.6.0"
)

func Format(ctx context.Context, paths ...string) error {
	sg.Deps(ctx, prepareCommand)
	args := make([]string, 0, 4+len(paths))
	args = append(args, "format", "--write", "--log-kind", "compact")
	binDir := sg.FromToolsDir(toolName, version)
	execDir := filepath.Join(binDir, toolName)
	args = append(args, paths...)

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
	binURL := "https://github.com/biomejs/biome/releases/download/cli%2F" +
		fmt.Sprintf("v%s/%s", version, toolNameWithArch)

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
