package sggcov2lcov

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
	name    = "gcov2lcov"
	version = "1.1.1"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	cmd := sg.Command(ctx, sg.FromBinDir(name), args...)
	// GOROOT need to be set, and as suggested here: https://github.com/jandelgado/gcov2lcov#goroot.
	goroot := sg.Output(sg.Command(ctx, "go", "env", "GOROOT"))
	cmd.Env = append(cmd.Env, "GOROOT="+goroot)
	return cmd
}

// Convert converts inFile in GCOV format to outFile in LCOV format.
func Convert(ctx context.Context, inFile, outFile string) error {
	return Command(ctx, "-infile="+inFile, "-outfile="+outFile).Run()
}

func PrepareCommand(ctx context.Context) error {
	binDir := sg.FromToolsDir(name, version)
	binary := filepath.Join(binDir, name)
	binURL := fmt.Sprintf(
		"https://github.com/jandelgado/gcov2lcov/releases/download/v%s/gcov2lcov_%s_%s_%s.tar.gz",
		version,
		version,
		runtime.GOOS,
		runtime.GOARCH,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(binDir),
		sgtool.WithUntarGz(),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", name, err)
	}
	return os.Chmod(binary, 0o755)
}
