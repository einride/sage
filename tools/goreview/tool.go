package goreview

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/einride/mage-tools/file"
	"github.com/einride/mage-tools/tools"
	"github.com/einride/mage-tools/tools/gh"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var (
	version string
	Binary  string
)

func SetGoReviewVersion(v string) (string, error) {
	version = v
	return version, nil
}

func Goreview() error {
	mg.Deps(mg.F(goreview, version))
	// TODO: the args should probably not be hardocded
	fmt.Println("[goreview] reviewing Go code for Einride-specific conventions...")
	return sh.RunV(Binary, "-c", "1", "./...")
}

func goreview(version string) error {
	mg.Deps(mg.F(gh.GH, version))
	const binaryName = "goreview"
	const defaultVersion = "0.18.0"

	if version == "" {
		version = defaultVersion
	} else {
		supportedVersions := []string{"0.18.0"}
		if err := tools.IsSupportedVersion(supportedVersions, version, binaryName); err != nil {
			return err
		}
	}

	binDir := filepath.Join(tools.GetPath(), binaryName, version, "bin")
	binary := filepath.Join(binDir, binaryName)
	Binary = binary

	// Check if binary already exist
	if file.Exists(binary) == nil {
		return nil
	}

	hostOS := strings.Title(runtime.GOOS)
	hostArch := runtime.GOARCH
	if hostArch == tools.AMD64 {
		hostArch = tools.X8664
	}
	goreviewVersion := "v" + version
	pattern := fmt.Sprintf("*%s_%s.tar.gz", hostOS, hostArch)
	archive := fmt.Sprintf("%s/goreview_%s_%s_%s.tar.gz", binDir, version, hostOS, hostArch)

	if err := sh.Run(
		gh.Binary,
		"release",
		"download",
		"--repo",
		"einride/goreview",
		goreviewVersion,
		"--pattern",
		pattern,
		"--dir",
		binDir,
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}

	if err := file.FromLocal(
		archive,
		file.WithName(filepath.Base(binary)),
		file.WithDestinationDir(binDir),
		file.WithUntarGz(),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}

	return nil
}
