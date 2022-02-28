package sgclangformat

import (
	"context"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	toolName = "clang-format"
	version  = "v1.6.0"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(toolName), args...)
}

func FormatProtoCommand(ctx context.Context, args ...string) *exec.Cmd {
	const protoStyle = "--style={BasedOnStyle: Google, ColumnLimit: 0, Language: Proto}"
	return Command(ctx, append([]string{"-i", protoStyle}, args...)...)
}

// FormatProto formats all proto files under the current working directory.
func FormatProto(ctx context.Context, args ...string) error {
	var protoFiles []string
	if err := filepath.WalkDir(sg.FromWorkDir(), func(path string, d fs.DirEntry, err error) error {
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
	switch strings.Split(runtime.GOOS, "/")[0] {
	case "linux":
		osArch = "linux_x64"
	case sgtool.Darwin:
		osArch = "darwin_x64"
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	toolDir := sg.FromToolsDir(toolName, version)
	binary := filepath.Join(toolDir, toolName)
	binURL := fmt.Sprintf(
		"https://github.com/angular/clang-format/blob/%s/bin/%s/clang-format?raw=true",
		version,
		osArch,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(toolDir),
		sgtool.WithRenameFile("clang-format?raw=true", toolName),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", toolName, err)
	}
	return nil
}
