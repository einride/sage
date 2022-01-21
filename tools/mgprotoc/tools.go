package mgprotoc

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/magefile/mage/mg"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

const version = "3.15.7"

// nolint: gochecknoglobals
var commandPath string

func Command(ctx context.Context, args ...string) *exec.Cmd {
	mg.CtxDeps(ctx, Prepare.Protoc)
	return mgtool.Command(ctx, commandPath, args...)
}

type Prepare mgtool.Prepare

func (Prepare) Protoc(ctx context.Context) error {
	const binaryName = "protoc"
	binDir := mgpath.FromTools(binaryName, version)
	binary := filepath.Join(binDir, "bin", binaryName)
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostArch == mgtool.AMD64 {
		hostArch = mgtool.X8664
	}
	binURL := fmt.Sprintf(
		"https://github.com/protocolbuffers/protobuf/releases/download/v%s/protoc-%s-%s-%s.zip",
		version,
		version,
		hostOS,
		hostArch,
	)
	if err := mgtool.FromRemote(
		ctx,
		binURL,
		mgtool.WithDestinationDir(binDir),
		mgtool.WithUnzip(),
		mgtool.WithSkipIfFileExists(binary),
		mgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	if err := os.RemoveAll(filepath.Join(binDir, "include")); err != nil {
		return err
	}
	commandPath = binary
	return nil
}
