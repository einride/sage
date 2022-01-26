package mgbuf

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	"go.einride.tech/mage-tools/mg"
	"go.einride.tech/mage-tools/mgtool"
)

const version = "1.0.0-rc10"

// nolint: gochecknoglobals
var commandPath string

func Command(ctx context.Context, args ...string) *exec.Cmd {
	mg.Deps(ctx, PrepareCommand)
	return mg.Command(ctx, commandPath, args...)
}

func PrepareCommand(ctx context.Context) error {
	const binaryName = "buf"
	binDir := mg.FromToolsDir(binaryName, version, "bin")
	binary := filepath.Join(binDir, binaryName)
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostArch == mgtool.AMD64 {
		hostArch = mgtool.X8664
	}
	binURL := fmt.Sprintf(
		"https://github.com/bufbuild/buf/releases/download/v%s/buf-%s-%s",
		version,
		hostOS,
		hostArch,
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
	return nil
}
