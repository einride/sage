package grpcjava

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/einride/mage-tools/file"
	"github.com/einride/mage-tools/tools"
)

var Binary string

func GrpcJava(version string) error {
	const binaryName = "protoc-gen-grpc-java"
	const defaultVersion = "1.33.0"

	if version == "" {
		version = defaultVersion
	} else {
		supportedVersions := []string{"1.33.0"}
		if err := tools.IsSupportedVersion(supportedVersions, version, binaryName); err != nil {
			return err
		}
	}

	binDir := filepath.Join(tools.Path, "grpc-java", version, "bin")
	binary := filepath.Join(binDir, binaryName)
	Binary = binary

	// read the whole file at once
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
	if hostArch == tools.AMD64 {
		hostArch = tools.X8664
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

	if err := file.FromRemote(
		binURL,
		file.WithName(filepath.Base(binary)),
		file.WithDestinationDir(binDir),
		file.WithRenameFile("", binaryName),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download grpc-java: %w", err)
	}

	return nil
}
