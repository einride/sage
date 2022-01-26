package mgclangformat

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"go.einride.tech/mage-tools/mg"
	"go.einride.tech/mage-tools/mgtool"
)

const (
	toolName = "clang-format"
	version  = "v1.6.0"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	mg.Deps(ctx, PrepareCommand)
	return mg.Command(ctx, mg.FromBinDir(toolName), args...)
}

func FormatProtoCommand(ctx context.Context, args ...string) *exec.Cmd {
	const protoStyle = "--style={BasedOnStyle: Google, ColumnLimit: 0, Language: Proto}"
	return Command(ctx, append([]string{"-i", protoStyle}, args...)...)
}

func PrepareCommand(ctx context.Context) error {
	var osArch string
	switch strings.Split(runtime.GOOS, "/")[0] {
	case "linux":
		osArch = "linux_x64"
	case "darwin":
		osArch = "darwin_x64"
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	toolDir := mg.FromToolsDir(toolName, version)
	binary := filepath.Join(toolDir, toolName)
	binURL := fmt.Sprintf(
		"https://github.com/angular/clang-format/blob/%s/bin/%s/clang-format?raw=true",
		version,
		osArch,
	)
	if err := mgtool.FromRemote(
		ctx,
		binURL,
		mgtool.WithDestinationDir(toolDir),
		mgtool.WithRenameFile("clang-format?raw=true", toolName),
		mgtool.WithSkipIfFileExists(binary),
		mgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", toolName, err)
	}
	return nil
}
