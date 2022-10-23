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
	if hostOS == "osx" && hostArch == sgtool.ARM64 {
		hostArch = sgtool.X8664
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
