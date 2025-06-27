package sgshellcheck

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
	"go.einride.tech/sage/tools/sgxz"
)

const (
	version = "0.10.0"
	name    = "shellcheck"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

// Run shellcheck on all files ending with .sh and .bash in the repo.
func Run(ctx context.Context) error {
	var inputFiles []string
	if err := filepath.WalkDir(sg.FromGitRoot(), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		switch filepath.Ext(d.Name()) {
		case ".sh", ".bash":
			inputFiles = append(inputFiles, path)
		}
		return nil
	}); err != nil {
		return err
	}
	return Command(ctx, inputFiles...).Run()
}

func PrepareCommand(ctx context.Context) error {
	sg.Deps(ctx, sgxz.PrepareCommand)
	const binaryName = "shellcheck"
	toolDir := sg.FromToolsDir(binaryName)
	binDir := filepath.Join(toolDir, version, "bin")
	binary := filepath.Join(binDir, binaryName)
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostArch == sgtool.AMD64 {
		hostArch = sgtool.X8664
	}
	if hostOS == sgtool.Darwin && hostArch == sgtool.ARM64 {
		hostArch = sgtool.X8664
	}
	shellcheck := fmt.Sprintf("shellcheck-v%s", version)
	binURL := fmt.Sprintf(
		"https://github.com/koalaman/shellcheck/releases/download/v%s/%s",
		version,
		fmt.Sprintf("%s.%s.%s.tar.xz", shellcheck, hostOS, hostArch),
	)

	decompressedArchive := filepath.Join(toolDir, fmt.Sprintf("%s.%s.%s.tar", shellcheck, hostOS, hostArch))
	compressedArchive := decompressedArchive + ".xz"

	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(toolDir),
		sgtool.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	// Manual check for existence since we can't use sgtool here
	if _, err := os.Stat(binary); errors.Is(err, os.ErrNotExist) {
		if err := sgxz.Command(ctx, "-d", compressedArchive).Run(); err != nil {
			return fmt.Errorf("unable to decompress %s: %w", compressedArchive, err)
		}
	}
	if err := sgtool.FromLocal(
		ctx,
		decompressedArchive,
		sgtool.WithUntar(),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithDestinationDir(binDir),
		sgtool.WithSymlink(binary),
		sgtool.WithRenameFile(fmt.Sprintf("%s/shellcheck", shellcheck), binaryName),
	); err != nil {
		return fmt.Errorf("unable to extract %s: %w", decompressedArchive, err)
	}
	return os.RemoveAll(decompressedArchive)
}
