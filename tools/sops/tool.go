package sops

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/einride/mage-tools/file"
	"github.com/einride/mage-tools/tools"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var (
	version string
	Binary  string
)

func SetSopsVersion(v string) (string, error) {
	version = v
	return version, nil
}

func Sops(file string) error {
	mg.Deps(mg.F(sops, version))
	return sh.RunV(Binary, file)
}

func sops(version string) error {
	const binaryName = "sops"
	const defaultVersion = "3.7.1"

	if version == "" {
		version = defaultVersion
	} else {
		supportedVersions := []string{"3.7.1"}
		if err := tools.IsSupportedVersion(supportedVersions, version, binaryName); err != nil {
			return err
		}
	}

	binDir := filepath.Join(tools.Path, binaryName, version)
	binary := filepath.Join(binDir, binaryName)
	Binary = binary

	hostOS := runtime.GOOS

	binURL := fmt.Sprintf(
		"https://github.com/mozilla/sops/releases/download/v%s/sops-v%s.%s",
		version,
		version,
		hostOS,
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
