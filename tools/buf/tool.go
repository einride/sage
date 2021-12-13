package buf

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/einride/mage-tools/file"
	"github.com/einride/mage-tools/tools"
)

var Binary string

func Buf(version string) error {
	const binaryName = "buf"
	const defaultVersion = "0.55.0"

	if version == "" {
		version = defaultVersion
	} else {
		supportedVersions := []string{"0.55.0"}
		if err := tools.IsSupportedVersion(supportedVersions, version, binaryName); err != nil {
			return err
		}
	}

	binDir := filepath.Join(tools.GetPath(), binaryName, version, "bin")
	binary := filepath.Join(binDir, binaryName)
	Binary = binary

	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostArch == tools.AMD64 {
		hostArch = tools.X8664
	}

	binURL := fmt.Sprintf(
		"https://github.com/bufbuild/buf/releases/download/v%s/buf-%s-%s",
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
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}

	return nil
}
