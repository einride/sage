package sgphrase

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const version = "2.5.3"

//nolint:gochecknoglobals
var commandPath string

// Phrase is used by the frontend guild for managing translations across Saga.
func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, commandPath, args...)
}

func PrepareCommand(ctx context.Context) error {
	const binaryName = "phrase"
	binDir := sg.FromToolsDir(binaryName, version)
	binary := filepath.Join(binDir, binaryName)
	hostOS := runtime.GOOS
	if hostOS == sgtool.Darwin {
		hostOS = "macosx"
	}
	hostArch := runtime.GOARCH
	binURL := fmt.Sprintf(
		"https://github.com/phrase/phrase-cli/releases/download/%s/phrase_%s_%s.tar.gz",
		version,
		hostOS,
		hostArch,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(binDir),
		sgtool.WithUntarGz(),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	if err := os.RemoveAll(filepath.Join(binDir, "include")); err != nil {
		return err
	}
	commandPath = binary
	return nil
}
