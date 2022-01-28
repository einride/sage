package sgterraform

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	version    = "1.1.4"
	binaryName = "terraform"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(binaryName), args...)
}

func PrepareCommand(ctx context.Context) error {
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	binaryDir := sg.FromBinDir(binaryName)
	binary := filepath.Join(binaryDir, binaryName)
	terraform := fmt.Sprintf("terraform_%s_%s_%s", version, hostOS, hostArch)
	binURL := fmt.Sprintf(
		"https://releases.hashicorp.com/terraform/%s/%s.zip",
		version,
		terraform,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(binaryDir),
		sgtool.WithUnzip(),
		sgtool.WithRenameFile(fmt.Sprintf("%s/terraform", terraform), binaryName),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}

	return nil
}
