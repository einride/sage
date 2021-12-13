package googleprotoscrubber

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/einride/mage-tools/file"
	"github.com/einride/mage-tools/tools"
)

var Binary string

func GoogleProtoScrubber(version string) error {
	const binaryName = "google-cloud-proto-scrubber"
	const defaultVersion = "1.1.0"

	if version == "" {
		version = defaultVersion
	} else {
		supportedVersions := []string{"1.1.0"}
		if err := tools.IsSupportedVersion(supportedVersions, version, binaryName); err != nil {
			return err
		}
	}
	binDir := filepath.Join(tools.GetPath(), binaryName, version)
	binary := filepath.Join(binDir, binaryName)
	Binary = binary

	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostArch == tools.AMD64 {
		hostArch = tools.X8664
	}

	binURL := fmt.Sprintf(
		"https://github.com/einride/google-cloud-proto-scrubber"+
			"/releases/download/v%s/google-cloud-proto-scrubber_%s_%s_%s.tar.gz",
		version,
		version,
		hostOS,
		hostArch,
	)

	if err := file.FromRemote(
		binURL,
		file.WithName(filepath.Base(binary)),
		file.WithDestinationDir(binDir),
		file.WithUntarGz(),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}

	if err := os.Chmod(binary, 0o755); err != nil {
		return fmt.Errorf("unable to make %s executable: %w", binaryName, err)
	}

	return nil
}
