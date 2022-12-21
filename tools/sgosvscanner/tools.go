package sgosvscanner

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

const (
	version    = "1.0.2"
	binaryName = "osv-scanner"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(binaryName), args...)
}

func PrepareCommand(ctx context.Context) error {
	binDir := sg.FromToolsDir(binaryName, version)
	binary := filepath.Join(binDir, binaryName)
	if err := sgtool.FromRemote(
		ctx,
		fmt.Sprintf(
			"https://github.com/google/osv-scanner/releases/download/v%s/osv-scanner_%s_%s_%s",
			version,
			version,
			runtime.GOOS,
			runtime.GOARCH,
		),
		sgtool.WithDestinationDir(binDir),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithRenameFile("", "osv-scanner"),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	if err := os.Chmod(binary, 0o755); err != nil {
		return fmt.Errorf("unable to make %s command: %w", binaryName, err)
	}
	return nil
}
