// Package mghadolint is a Dockerfile linter that you can employ to check for Dockerfile best
// practices and common mistakes.
package mghadolint

import (
	"bytes"
	"context"
	"fmt"
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

// version can be found here: https://github.com/hadolint/hadolint
const version = "2.8.0"

// nolint: gochecknoglobals
var commandPath string

func Command(ctx context.Context, args ...string) *exec.Cmd {
	ctx = logr.NewContext(ctx, mglog.Logger("hadolint"))
	mg.CtxDeps(ctx, Prepare.Hadolint)
	return mgtool.Command(commandPath, args...)
}

func LintCommand(ctx context.Context) *exec.Cmd {
	cmd := mgtool.Command("git", "ls-files", "--exclude-standard", "--cached", "--others", "--", "*Dockerfile*")
	var b bytes.Buffer
	cmd.Stdout = &b
	if err := cmd.Run(); err != nil {
		panic(fmt.Errorf("failed to list Dockerfiles: %w", err))
	}
	if b.String() == "" {
		// No Dockerfiles to lint, then there is no need to run hadolint.
		return nil
	}
	dockerfiles := strings.Split(b.String(), "\n")
	return Command(ctx, dockerfiles...)
}

type Prepare mgtool.Prepare

func (Prepare) Hadolint(ctx context.Context) error {
	const binaryName = "hadolint"
	toolDir := filepath.Join(mgpath.Tools(), binaryName)
	binDir := filepath.Join(toolDir, version, "bin")
	binary := filepath.Join(binDir, binaryName)
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostArch == mgtool.AMD64 {
		hostArch = mgtool.X8664
	}
	hadolint := fmt.Sprintf("hadolint-%s-%s", strings.ToTitle(hostOS), hostArch)
	binURL := fmt.Sprintf(
		"https://github.com/hadolint/hadolint/releases/download/v%s/%s",
		version,
		hadolint,
	)
	if err := mgtool.FromRemote(
		ctx,
		binURL,
		mgtool.WithDestinationDir(binDir),
		mgtool.WithRenameFile(fmt.Sprintf("%s/hadolint", hadolint), binaryName),
		mgtool.WithSkipIfFileExists(binary),
		mgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	commandPath = binary
	return nil
}