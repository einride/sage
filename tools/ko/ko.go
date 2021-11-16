package ko

import (
	"fmt"
	"os"

	"github.com/einride/mage-tools/tools"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var version string

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
	mg.Deps(mg.F(tools.Ko, version))
	fmt.Println("[ko] info building ko...")
	if err := sh.RunV(tools.KoPath, args...); err != nil {
		return err
	}
	return nil
}
