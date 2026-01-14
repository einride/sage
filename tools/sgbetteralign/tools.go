package sgbetteralign

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
	"go.einride.tech/sage/tools/sgxz"
)

const (
	// renovate: datasource=github-releases depName=dkorunic/betteralign
	version = "0.2.5"
	name    = "betteralign"
)

// Note: Ignore structs using a comment with betteralign:ignore

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

// Run betteralign on all files ending with .sh and .bash in the repo.
func Run(ctx context.Context, apply bool) error {
	args := []string{}
	if apply {
		args = append(args, "-apply")
	}
	args = append(args, "./...")
	cmd := Command(ctx, args...)
	cmd.Dir = sg.FromGitRoot()
	return cmd.Run()
}

func PrepareCommand(ctx context.Context) error {
	sg.Deps(ctx, sgxz.PrepareCommand)
	toolDir := sg.FromToolsDir(name)
	binDir := filepath.Join(toolDir, version, "bin")
	binary := filepath.Join(binDir, name)
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostArch == sgtool.AMD64 {
		hostArch = sgtool.X8664
	}
	if hostOS == sgtool.Darwin {
		hostArch = "all"
	}
	betteralign := fmt.Sprintf("%s_%s", name, version)
	binURL := fmt.Sprintf(
		"https://github.com/dkorunic/betteralign/releases/download/v%s/%s",
		version,
		fmt.Sprintf("%s_%s_%s.tar.gz", betteralign, hostOS, hostArch),
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
	return nil
}
