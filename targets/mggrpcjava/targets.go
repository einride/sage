package mggrpcjava

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-logr/logr"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

const version = "1.33.0"

// nolint: gochecknoglobals
var executable string

type Prepare mgtool.Prepare

func (Prepare) ProtocGenGrpcJava(ctx context.Context) error {
	return prepare(ctx)
}

func ProtocGenGrpcJava(ctx context.Context) error {
	ctx = logr.NewContext(ctx, mglog.Logger("protoc-gen-grpc-java"))
	mg.CtxDeps(ctx, prepare)
	logr.FromContextOrDiscard(ctx).Info("running...")
	return sh.RunV(executable)
}

func prepare(ctx context.Context) error {
	const binaryName = "protoc-gen-grpc-java"

	binDir := filepath.Join(mgpath.Tools(), "grpc-java", version, "bin")
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
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	executable = binary
	return nil
}
