package mgapilinter

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	"go.einride.tech/mage-tools/mg"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

const version = "1.29.3"

// nolint: gochecknoglobals
var commandPath string

func Command(ctx context.Context, args ...string) *exec.Cmd {
	mg.Deps(ctx, Prepare.APILinter)
	return mgtool.Command(ctx, commandPath, args...)
}

type Prepare mgtool.Prepare

func (Prepare) APILinter(ctx context.Context) error {
	const binaryName = "api-linter"
	hostOS := runtime.GOOS
	binDir := mgpath.FromToolsDir(binaryName, version, "bin")
	binary := filepath.Join(binDir, binaryName)
	binURL := fmt.Sprintf(
		"https://github.com/googleapis/api-linter/releases/download/v%s/api-linter-%s-%s-amd64.tar.gz",
		version,
		version,
		hostOS,
	)
	if err := mgtool.FromRemote(
		ctx,
		binURL,
		mgtool.WithDestinationDir(binDir),
		mgtool.WithUntarGz(),
		mgtool.WithSkipIfFileExists(binary),
		mgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	commandPath = binary
	return nil
}
