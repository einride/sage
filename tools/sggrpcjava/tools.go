package sggrpcjava

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const version = "1.45.1"

//nolint:gochecknoglobals
var commandPath string

func Command(ctx context.Context) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, commandPath)
}

func PrepareCommand(ctx context.Context) error {
	const binaryName = "protoc-gen-grpc-java"
	// TODO: Releases up to 1.75.0 ship x86_64 under osx-aarch_64. Bump to 1.76.0+ (universal) and remove this check.
	if runtime.GOOS == sgtool.Darwin && runtime.GOARCH == sgtool.ARM64 {
		return fmt.Errorf("%s %s has no arm64 macOS binary", binaryName, version)
	}
	binDir := sg.FromToolsDir("grpc-java", version, "bin")
	binary := filepath.Join(binDir, binaryName)
	hostOS := runtime.GOOS
	if hostOS == sgtool.Darwin {
		hostOS = "osx"
	}
	hostArch := runtime.GOARCH
	if hostArch == sgtool.AMD64 {
		hostArch = sgtool.X8664
	}
	if hostArch == sgtool.ARM64 {
		hostArch = "aarch_64"
	}
	binURL := fmt.Sprintf(
		"https://repo1.maven.org/maven2/io/grpc/%s/%s/%s-%s-%s-%s.exe",
		binaryName,
		version,
		binaryName,
		version,
		hostOS,
		hostArch,
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
	commandPath = binary
	return nil
}
