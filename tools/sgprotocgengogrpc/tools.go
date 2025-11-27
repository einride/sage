package sgprotocgengogrpc

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
	// Version is the protoc-gen-go-grpc version.
	Version = "1.4.0"
	// Name is the binary name.
	Name = "protoc-gen-go-grpc"
	// Repo is the GitHub repository.
	Repo = "grpc/grpc-go"
	// TagPattern matches version tags (monorepo pattern).
	TagPattern = `^cmd/protoc-gen-go-grpc/v(\d+\.\d+\.\d+)$`
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(Name), args...)
}

func PrepareCommand(ctx context.Context) error {
	binDir := sg.FromToolsDir(Name, Version)
	binary := filepath.Join(binDir, Name)
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostOS == sgtool.Darwin {
		hostArch = sgtool.AMD64
	}
	binURL := fmt.Sprintf(
		"https://github.com/grpc/grpc-go/releases/download/cmd/protoc-gen-go-grpc/v%s/protoc-gen-go-grpc.v%s.%s.%s.tar.gz",
		Version,
		Version,
		hostOS,
		hostArch,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(binDir),
		sgtool.WithUntarGz(),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", Name, err)
	}
	if err := os.Chmod(binary, 0o755); err != nil {
		return fmt.Errorf("unable to make %s command: %w", Name, err)
	}
	return nil
}
