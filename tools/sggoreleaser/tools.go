package sggoreleaser

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	name    = "goreleaser"
	version = "1.25.1"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	binDir := sg.FromToolsDir(name, version)
	binary := filepath.Join(binDir, name)
	var hostOS string
	switch strings.Split(runtime.GOOS, "/")[0] {
	case "linux":
		hostOS = "Linux"
	case "darwin":
		hostOS = "Darwin"
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x86_64"
	}
	binURL := fmt.Sprintf(
		"https://github.com/goreleaser/goreleaser/releases/download/v%s/goreleaser_%s_%s.tar.gz",
		version,
		hostOS,
		arch,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(binDir),
		sgtool.WithRenameFile("", name),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
		sgtool.WithUntarGz(),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", name, err)
	}
	return os.Chmod(binary, 0o755)
}
