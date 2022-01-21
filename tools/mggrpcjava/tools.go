package mggrpcjava

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/magefile/mage/mg"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

const version = "1.33.0"

// nolint: gochecknoglobals
var commandPath string

func Command(ctx context.Context) *exec.Cmd {
	mg.CtxDeps(ctx, Prepare.ProtocGenGrpcJava)
	return mgtool.Command(ctx, commandPath)
}

type Prepare mgtool.Prepare

func (Prepare) ProtocGenGrpcJava(ctx context.Context) error {
	const binaryName = "protoc-gen-grpc-java"
	binDir := mgpath.FromTools("grpc-java", version, "bin")
	binary := filepath.Join(binDir, binaryName)
	// read the whole pom at once
	b, err := os.ReadFile("pom.xml")
	if err != nil {
		panic(err)
	}
	s := string(b)
	if !strings.Contains(s, fmt.Sprintf("<grpc.version>%s", version)) {
		return fmt.Errorf("pom.xml is out of sync with gRPC Java version - expecting %s", version)
	}
	hostOS := runtime.GOOS
	if hostOS == "darwin" {
		hostOS = "osx"
	}
	hostArch := runtime.GOARCH
	if hostArch == mgtool.AMD64 {
		hostArch = mgtool.X8664
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
	if err := mgtool.FromRemote(
		ctx,
		binURL,
		mgtool.WithDestinationDir(binDir),
		mgtool.WithRenameFile("", binaryName),
		mgtool.WithSkipIfFileExists(binary),
		mgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	commandPath = binary
	return nil
}
