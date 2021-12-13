package ko

import (
	"fmt"
	"os"
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

func SetKoVersion(v string) (string, error) {
	version = v
	return version, nil
}

func PublishLocal() error {
	dockerTag, err := tag()
	if err != nil {
		return err
	}
	err = publish(
		[]string{
			"publish",
			"--local",
			"--preserve-import-paths",
			"-t",
			dockerTag,
			"./cmd/server",
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func Publish(repo string) error {
	_ = os.Setenv("KO_DOCKER_REPO", repo)
	dockerTag, err := tag()
	if err != nil {
		return err
	}
	err = publish(
		[]string{
			"publish",
			"--preserve-import-paths",
			"-t",
			dockerTag,
			"./cmd/server",
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func tag() (string, error) {
	revision, err := sh.Output("git", "rev-parse", "--verify", "HEAD")
	if err != nil {
		return "", err
	}
	diff, err := sh.Output("git", "status", "--porcelain")
	if err != nil {
		return "", err
	}
	if diff != "" {
		revision += "-dirty"
	}
	_ = os.Setenv("DOCKER_TAG", revision)
	return revision, nil
}

func publish(args []string) error {
	mg.Deps(mg.F(ko, version))
	fmt.Println("[ko] info building ko...")
	if err := sh.RunV(Binary, args...); err != nil {
		return err
	}
	return nil
}

func ko(version string) error {
	const binaryName = "ko"
	const defaultVersion = "0.9.3"

	if version == "" {
		version = defaultVersion
	} else {
		supportedVersions := []string{"0.9.3"}
		if err := tools.IsSupportedVersion(supportedVersions, version, binaryName); err != nil {
			return err
		}
	}

	hostOS := runtime.GOOS

	binDir := filepath.Join(tools.GetPath(), binaryName, version, "bin")
	binary := filepath.Join(binDir, binaryName)
	Binary = binary

	binURL := fmt.Sprintf(
		"https://github.com/google/ko/releases/download/v%s/ko_%s_%s_x86_64.tar.gz",
		version,
		version,
		hostOS,
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

	return nil
}
