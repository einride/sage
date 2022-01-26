package mgconvco

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"go.einride.tech/mage-tools/mg"
	"go.einride.tech/mage-tools/mgtool"
)

const version = "0.3.7"

// nolint: gochecknoglobals
var commandPath string

func Command(ctx context.Context, args ...string) *exec.Cmd {
	mg.Deps(ctx, PrepareCommand)
	return mg.Command(ctx, commandPath, args...)
}

func PrepareCommand(ctx context.Context) error {
	const toolName = "convco"
	binDir := mg.FromToolsDir(toolName, version)
	binary := filepath.Join(binDir, toolName)
	var hostOS string
	switch strings.Split(runtime.GOOS, "/")[0] {
	case "linux":
		hostOS = "ubuntu"
	case "darwin":
		hostOS = "macos"
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	binURL := fmt.Sprintf(
		"https://github.com/convco/convco/releases/download/v%s/convco-%s.zip",
		version,
		hostOS,
	)
	if err := mgtool.FromRemote(
		ctx,
		binURL,
		mgtool.WithDestinationDir(binDir),
		mgtool.WithUnzip(),
		mgtool.WithSkipIfFileExists(binary),
		mgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", toolName, err)
	}
	commandPath = binary
	return os.Chmod(binary, 0o755)
}
