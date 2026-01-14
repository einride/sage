package sggcov2lcov

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
	name = "gcov2lcov"
	// renovate: datasource=github-releases depName=jandelgado/gcov2lcov
	version = "1.0.6"
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
	var hostOS string
	switch strings.Split(runtime.GOOS, "/")[0] {
	case "linux":
		hostOS = "linux-amd64"
	case sgtool.Darwin:
		hostOS = "darwin-amd64"
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	binURL := fmt.Sprintf(
		"https://github.com/jandelgado/gcov2lcov/releases/download/v%s/gcov2lcov-%s.tar.gz",
		version,
		hostOS,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(binDir),
		sgtool.WithRenameFile("bin/"+name+"-"+hostOS, name),
		sgtool.WithUntarGz(),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", name, err)
	}
	return os.Chmod(binary, 0o755)
}
