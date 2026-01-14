package sgprotocgendecapcms

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
	// renovate: datasource=github-releases depName=einride/protobuf-decap-cms
	version    = "1.1.0"
	binaryName = "protoc-gen-decap-cms"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(binaryName), args...)
}

func PrepareCommand(ctx context.Context) error {
	binDir := sg.FromToolsDir(binaryName, version)
	binary := filepath.Join(binDir, binaryName)
	downloadURL := fmt.Sprintf(
		"https://github.com/einride/protobuf-decap-cms/releases/download/v%s/protobuf-decap-cms_%s_%s_%s.tar.gz",
		version,
		version,
		runtime.GOOS,
		runtime.GOARCH,
	)

	if err := sgtool.FromRemote(
		ctx,
		downloadURL,
		sgtool.WithDestinationDir(binDir),
		sgtool.WithUntarGz(),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	if err := os.Chmod(binary, 0o755); err != nil {
		return fmt.Errorf("unable to make %s command: %w", binaryName, err)
	}
	return nil
}
