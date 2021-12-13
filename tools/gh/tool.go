package gh

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/einride/mage-tools/file"
	"github.com/einride/mage-tools/tools"
)

var (
	ghVersion string
	Binary    string
)

func SetGhVersion(v string) (string, error) {
	ghVersion = v
	return ghVersion, nil
}

func GH(version string) error {
	const binaryName = "gh"
	const defaultVersion = "2.2.0"

	if version == "" {
		version = defaultVersion
	} else {
		supportedVersions := []string{"2.2.0"}
		if err := tools.IsSupportedVersion(supportedVersions, version, binaryName); err != nil {
			return err
		}
	}

	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH

	dir := filepath.Join(tools.GetPath(), binaryName)
	binDir := filepath.Join(dir, version, "bin")
	binary := filepath.Join(binDir, binaryName)
	Binary = binary

	binURL := fmt.Sprintf(
		"https://github.com/cli/cli/releases/download/v%s/gh_%s_%s_%s.tar.gz",
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
		file.WithRenameFile(fmt.Sprintf("gh_%s_%s_%s/bin/gh", version, hostOS, hostArch), binaryName),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}

	return nil
}
