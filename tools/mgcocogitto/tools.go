package mgcocogitto

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-logr/logr"
	"github.com/magefile/mage/mg"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

const version = "4.0.1"

// nolint: gochecknoglobals
var commandPath string

func Command(ctx context.Context, args ...string) *exec.Cmd {
	ctx = logr.NewContext(ctx, mglog.Logger("cog"))
	mg.CtxDeps(ctx, Prepare.Cog)
	return mgtool.Command(commandPath, args...)
}

type Prepare mgtool.Prepare

func (Prepare) Cog(ctx context.Context) error {
	const toolName = "cocogitto"
	binDir := filepath.Join(mgpath.Tools(), toolName, version)
	binary := filepath.Join(binDir, "cog")
	var archiveName string
	switch strings.Split(runtime.GOOS, "/")[0] {
	case "linux":
		archiveName = fmt.Sprintf("cocogitto-%s-x86_64-unknown-linux-musl.tar.gz", version)
	case "darwin":
		archiveName = fmt.Sprintf("cocogitto-%s-x86_64-osx.tar.gz", version)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	binURL := fmt.Sprintf(
		"https://github.com/cocogitto/cocogitto/releases/download/%s/%s",
		version,
		archiveName,
	)
	if err := mgtool.FromRemote(
		ctx,
		binURL,
		mgtool.WithDestinationDir(binDir),
		mgtool.WithUntarGz(),
		mgtool.WithSkipIfFileExists(binary),
		mgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", toolName, err)
	}
	commandPath = binary
	return os.Chmod(binary, 0o755)
}
