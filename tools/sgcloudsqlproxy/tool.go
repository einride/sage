package sgcloudsqlproxy

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
	version    = "1.30.0"
	binaryName = "cloud_sql_proxy"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(binaryName), args...)
}

func PrepareCommand(ctx context.Context) error {
	binDir := sg.FromToolsDir(binaryName, version)
	binary := filepath.Join(binDir, binaryName)
	filename := fmt.Sprintf(
		"cloud_sql_proxy.%s.%s",
		runtime.GOOS,
		runtime.GOARCH,
	)
	binURL := fmt.Sprintf(
		"https://storage.googleapis.com/cloudsql-proxy/v%s/%s",
		version,
		filename,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(binDir),
		sgtool.WithRenameFile("", binaryName),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	return nil
}
