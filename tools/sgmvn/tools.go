package sgmvn

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	name    = "mvn"
	version = "3.8.6"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	toolDir := sg.FromToolsDir(name)
	binDir := filepath.Join(toolDir, version, "bin")
	binary := filepath.Join(binDir, "apache-maven-"+version, "bin", name)
	if err := sgtool.FromRemote(
		ctx,
		fmt.Sprintf(
			"https://dlcdn.apache.org/maven/maven-3/%s/binaries/apache-maven-%s-bin.tar.gz",
			version,
			version,
		),
		sgtool.WithDestinationDir(binDir),
		sgtool.WithUntarGz(),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", name, err)
	}
	return nil
}
