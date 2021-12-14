package mgko

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/go-logr/logr"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgtool"
	"go.einride.tech/mage-tools/tools"
)

const version = "0.9.3"

var executable string

func PublishLocal(ctx context.Context) error {
	dockerTag, err := tag()
	if err != nil {
		return err
	}
	return ko(
		ctx,
		[]string{
			"publish",
			"--local",
			"--preserve-import-paths",
			"-t",
			dockerTag,
			"./cmd/server",
		},
	)
}

func Publish(ctx context.Context, repo string) error {
	_ = os.Setenv("KO_DOCKER_REPO", repo)
	dockerTag, err := tag()
	if err != nil {
		return err
	}
	return ko(
		ctx,
		[]string{
			"publish",
			"--preserve-import-paths",
			"-t",
			dockerTag,
			"./cmd/server",
		},
	)
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

func ko(ctx context.Context, args []string) error {
	logger := mglog.Logger("ko")
	ctx = logr.NewContext(ctx, logger)
	mg.CtxDeps(ctx, prepare)
	logger.Info("building ko...")
	return sh.RunV(executable, args...)
}

func prepare(ctx context.Context) error {
	const binaryName = "ko"

	hostOS := runtime.GOOS

	binDir := filepath.Join(tools.GetPath(), binaryName, version, "bin")
	binary := filepath.Join(binDir, binaryName)

	binURL := fmt.Sprintf(
		"https://github.com/google/ko/releases/download/v%s/ko_%s_%s_x86_64.tar.gz",
		version,
		version,
		hostOS,
	)

	if err := mgtool.FromRemote(
		ctx,
		binURL,
		mgtool.WithDestinationDir(binDir),
		mgtool.WithUntarGz(),
		mgtool.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}

	executable = binary
	return nil
}
