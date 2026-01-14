// Package sghadolint is a Dockerfile linter that you can employ to check for Dockerfile best
// practices and common mistakes.
package sghadolint

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

// renovate: datasource=github-releases depName=hadolint/hadolint
const version = "2.12.1-beta"

//nolint:gochecknoglobals
var commandPath string

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, commandPath, args...)
}

func Run(ctx context.Context) error {
	cmd := sg.Command(ctx, "git", "ls-files", "--exclude-standard", "--cached", "--others", "--", "*Dockerfile*")
	var b bytes.Buffer
	cmd.Stdout = &b
	if err := cmd.Run(); err != nil {
		panic(fmt.Errorf("failed to list Dockerfiles: %w", err))
	}
	if b.String() == "" {
		// No Dockerfiles to lint, then there is no need to run hadolint.
		return nil
	}
	spaceless := strings.TrimSpace(b.String())
	dockerfiles := strings.Split(spaceless, "\n")
	return Command(ctx, dockerfiles...).Run()
}

func PrepareCommand(ctx context.Context) error {
	const toolName = "hadolint"
	binDir := sg.FromToolsDir(toolName, version)
	binary := filepath.Join(binDir, toolName)
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostArch == sgtool.AMD64 {
		hostArch = sgtool.X8664
	}
	hadolint := fmt.Sprintf("hadolint-%s-%s", strings.ToTitle(hostOS), hostArch)
	binURL := fmt.Sprintf(
		"https://github.com/hadolint/hadolint/releases/download/v%s/%s",
		version,
		hadolint,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(binDir),
		sgtool.WithRenameFile(fmt.Sprintf("%s/hadolint", hadolint), toolName),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", toolName, err)
	}
	commandPath = binary
	return nil
}
