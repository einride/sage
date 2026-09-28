package sgclangformat

import (
	"context"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"runtime"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	toolName = "clang-format"
	version  = "14"
	release  = "master-796e77c"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(toolName), args...)
}

func FormatProtoCommand(ctx context.Context, args ...string) *exec.Cmd {
	const protoStyle = "--style={BasedOnStyle: Google, ColumnLimit: 0, Language: Proto}"
	return Command(ctx, append([]string{"-i", protoStyle}, args...)...)
}

// FormatProto formats all proto files in the repo.
func FormatProto(ctx context.Context) error {
	var protoFiles []string
	if err := filepath.WalkDir(sg.FromGitRoot(), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".proto" {
			protoFiles = append(protoFiles, path)
		}
		return nil
	}); err != nil {
		return err
	}
	return FormatProtoCommand(ctx, protoFiles...).Run()
}

func PrepareCommand(ctx context.Context) error {
	var osArch string
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		osArch = "linux-amd64"
	case "darwin/amd64":
		osArch = "macosx-amd64"
	case "darwin/arm64":
		osArch = "macos-arm-arm64"
	default:
		return fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	toolDir := sg.FromToolsDir(toolName, version, release)
	binary := filepath.Join(toolDir, toolName)
	asset := fmt.Sprintf("%s-%s_%s", toolName, version, osArch)
	binURL := fmt.Sprintf(
		"https://github.com/muttleyxd/clang-tools-static-binaries/releases/download/%s/%s",
		release,
		asset,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(toolDir),
		sgtool.WithRenameFile(asset, toolName),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", toolName, err)
	}
	return nil
}
