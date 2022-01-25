package mggosemanticrelease

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"go.einride.tech/mage-tools/mg"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

// nolint: gochecknoglobals
var commandPath string

func Command(ctx context.Context, args ...string) *exec.Cmd {
	mg.Deps(ctx, Prepare.GoSemanticRelease)
	return mgtool.Command(ctx, commandPath, args...)
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

type Prepare mgtool.Prepare

func (Prepare) GoSemanticRelease(ctx context.Context) error {
	const (
		binaryName = "gosemantic-release"
		version    = "2.18.0"
	)
	binDir := mgpath.FromToolsDir(binaryName, version)
	binary := filepath.Join(binDir, binaryName)
	var hostOS string
	switch strings.Split(runtime.GOOS, "/")[0] {
	case "linux":
		hostOS = "linux"
	case "darwin":
		hostOS = "darwin"
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	binURL := fmt.Sprintf(
		"https://github.com/go-semantic-release/semantic-release/releases/download/v%s/semantic-release_v%s_%s_amd64",
		version,
		version,
		hostOS,
	)
	if err := mgtool.FromRemote(
		ctx,
		binURL,
		mgtool.WithDestinationDir(binDir),
		mgtool.WithRenameFile("", binaryName),
		mgtool.WithSkipIfFileExists(binary),
		mgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	commandPath = binary
	return os.Chmod(binary, 0o755)
}
