package sggosemanticrelease

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
	name    = "go-semantic-release"
	version = "2.24.0"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func ReleaseFromCloudBuildCommand(ctx context.Context, ci bool, repo string) *exec.Cmd {
	args := []string{
		"--allow-initial-development-versions",
		"--allow-no-changes",
		"--ci-condition",
		"default",
		"--provider-opt",
	}
	args = append(args, "slug="+repo)
	if !ci {
		args = append(args, "--dry")
	}
	return Command(ctx, args...)
}

func PrepareCommand(ctx context.Context) error {
	binDir := sg.FromToolsDir(name, version)
	binary := filepath.Join(binDir, name)
	var hostOS string
	switch strings.Split(runtime.GOOS, "/")[0] {
	case "linux":
		hostOS = "linux"
	case sgtool.Darwin:
		hostOS = sgtool.Darwin
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	binURL := fmt.Sprintf(
		"https://github.com/go-semantic-release/semantic-release/releases/download/v%s/semantic-release_v%s_%s_%s",
		version,
		version,
		hostOS,
		runtime.GOARCH,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(binDir),
		sgtool.WithRenameFile("", name),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", name, err)
	}
	return os.Chmod(binary, 0o755)
}
