package mgko

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-logr/logr"
	"github.com/magefile/mage/mg"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

const version = "0.9.3"

// nolint: gochecknoglobals
var commandPath string

func Command(ctx context.Context, args ...string) *exec.Cmd {
	ctx = logr.NewContext(ctx, mglog.Logger("ko"))
	mg.CtxDeps(ctx, Prepare.Ko)
	return mgtool.Command(commandPath, args...)
}

func PublishLocalCommand(ctx context.Context) *exec.Cmd {
	dockerTag, err := tag()
	if err != nil {
		panic(err)
	}
	return Command(
		ctx,
		"publish",
		"--local",
		"--preserve-import-paths",
		"-t",
		dockerTag,
		"./cmd/server",
	)
}

func PublishCommand(ctx context.Context, repo string) *exec.Cmd {
	_ = os.Setenv("KO_DOCKER_REPO", repo)
	dockerTag, err := tag()
	if err != nil {
		panic(err)
	}
	return Command(
		ctx,
		"publish",
		"--preserve-import-paths",
		"-t",
		dockerTag,
		"./cmd/server",
	)
}

func tag() (string, error) {
	cmd := mgtool.Command("git", "rev-parse", "--verify", "HEAD")
	var b bytes.Buffer
	cmd.Stdout = &b
	if err := cmd.Run(); err != nil {
		return "", err
	}
	revision := strings.TrimSpace(b.String())
	cmd = mgtool.Command("git", "status", "--porcelain")
	var diff bytes.Buffer
	cmd.Stdout = &diff
	if err := cmd.Run(); err != nil {
		return "", err
	}
	if diff.String() != "" {
		revision += "-dirty"
	}
	_ = os.Setenv("DOCKER_TAG", revision)
	return revision, nil
}

type Prepare mgtool.Prepare

func (Prepare) Ko(ctx context.Context) error {
	const binaryName = "ko"

	hostOS := runtime.GOOS

	binDir := filepath.Join(mgpath.Tools(), binaryName, version, "bin")
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
		mgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}

	commandPath = binary
	return nil
}
