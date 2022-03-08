package sgshfmt

import (
	"context"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"runtime"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	version = "3.4.2"
	name    = "shfmt"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

// Run shfmt on all files ending with .sh and .bash in the repo.
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
	return Command(ctx, append([]string{"-w", "-s"}, inputFiles...)...).Run()
}

func PrepareCommand(ctx context.Context) error {
	const binaryName = "shfmt"
	toolDir := sg.FromToolsDir(binaryName)
	binDir := filepath.Join(toolDir, version, "bin")
	binary := filepath.Join(binDir, binaryName)
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	shfmt := fmt.Sprintf("shfmt_v%s_%s_%s", version, hostOS, hostArch)
	binURL := fmt.Sprintf(
		"https://github.com/mvdan/sh/releases/download/v%s/%s",
		version,
		shfmt,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(binDir),
		sgtool.WithRenameFile(fmt.Sprintf("%s/shfmt", shfmt), binaryName),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	return nil
}
