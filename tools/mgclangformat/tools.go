package mgclangformat

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/magefile/mage/mg"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

const version = "1.6.0"

// nolint: gochecknoglobals
var commandPath string

func Command(ctx context.Context, args ...string) *exec.Cmd {
	mg.Deps(Prepare.ClangFormat)
	return mgtool.Command(ctx, commandPath, args...)
}

func FormatProtoCommand(ctx context.Context, args ...string) *exec.Cmd {
	const protoStyle = "--style={BasedOnStyle: Google, ColumnLimit: 0, Language: Proto}"
	return Command(ctx, append([]string{"-i", protoStyle}, args...)...)
}

type Prepare mgtool.Prepare

func (Prepare) ClangFormat(ctx context.Context) error {
	var archiveName string
	switch strings.Split(runtime.GOOS, "/")[0] {
	case "linux":
		archiveName = "linux_x64"
	case "darwin":
		archiveName = "darwin_x64"
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	toolDir := mgpath.FromTools("clang-format")
	binary := filepath.Join(toolDir, "node_modules", "clang-format", "bin", archiveName, "clang-format")

	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return err
	}
	if err := mgtool.Command(
		ctx,
		"npm",
		"--silent",
		"install",
		"--prefix",
		toolDir,
		"--no-save",
		"--no-audit",
		"clang-format@"+version,
	).Run(); err != nil {
		return err
	}
	symlink, err := mgtool.CreateSymlink(binary)
	if err != nil {
		return err
	}
	commandPath = symlink
	return nil
}
