package protoc

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/einride/mage-tools/file"
	"github.com/einride/mage-tools/tools"
)

var Binary string

func Protoc(version string) error {
	const binaryName = "protoc"
	const defaultVersion = "3.15.7"

	if version == "" {
		version = defaultVersion
	} else {
		supportedVersions := []string{"3.15.7"}
		if err := tools.IsSupportedVersion(supportedVersions, version, binaryName); err != nil {
			return err
		}
	}

	binDir := filepath.Join(tools.Path, binaryName, version)
	binary := filepath.Join(binDir, "bin", binaryName)
	Binary = binary

	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostArch == tools.AMD64 {
		hostArch = tools.X8664
	}

	binURL := fmt.Sprintf(
		"https://github.com/protocolbuffers/protobuf/releases/download/v%s/protoc-%s-%s-%s.zip",
		version,
		version,
		hostOS,
		hostArch,
	)

	if err := file.FromRemote(
		binURL,
		file.WithName(filepath.Base(binary)),
		file.WithDestinationDir(binDir),
		file.WithUnzip(),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}

	if err := os.RemoveAll(filepath.Join(binDir, "include")); err != nil {
		return err
	}

	return nil
}
