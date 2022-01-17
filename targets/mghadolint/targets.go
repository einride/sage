// Package mghadolint is a Dockerfile linter that you can employ to check for Dockerfile best
// practices and common mistakes.
package mghadolint

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-logr/logr"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

// version can be found here: https://github.com/hadolint/hadolint
const version = "2.8.0"

// nolint: gochecknoglobals
var executable string

type Prepare mgtool.Prepare

func (Prepare) Hadolint(ctx context.Context) error {
	return prepare(ctx)
}

func Hadolint(ctx context.Context) error {
	ctx = logr.NewContext(ctx, mglog.Logger("hadolint"))
	mg.CtxDeps(ctx, mg.F(prepare))
	logr.FromContextOrDiscard(ctx).Info("running...")
	dockerfilesRaw, err := sh.Output("git", "ls-files", "--exclude-standard", "--cached", "--others", "--", "*Dockerfile*")
	if err != nil {
		return fmt.Errorf("failed to list Dockerfiles: %w", err)
	}
	if dockerfilesRaw == "" {
		// No Dockerfiles to lint, then there is no need to run hadolint.
		return nil
	}
	dockerfiles := strings.Split(dockerfilesRaw, "\n")
	return sh.RunV(executable, dockerfiles...)
}

func prepare(ctx context.Context) error {
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
	executable = binary
	return nil
}
