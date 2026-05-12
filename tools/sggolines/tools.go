// Deprecated: sggolines is deprecated. golangci-lint v2 includes a built-in
// golines formatter, so a standalone tool is no longer needed.
//
// See sggolangcilint for a replacement.
package sggolines

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
	"go.einride.tech/sage/tools/sggofumpt"
)

const (
	name    = "golines"
	version = "0.12.2"
)

// Run golines on all Go files in the current git root with gofumpt as default formatter.
//
// Deprecated: Use sggolangcilint.Run instead; golangci-lint v2 has a built-in golines formatter.
func Run(ctx context.Context) error {
	sg.Deps(ctx, sggofumpt.PrepareCommand)
	return Command(
		ctx,
		"--base-formatter=gofumpt",
		"--ignore-generated",
		"--max-len=120",
		"--tab-len=1",
		"--write-output",
		".",
	).Run()
}

// Command returns an [*exec.Cmd] for golines.
//
// Deprecated: Use sggolangcilint.Command instead; golangci-lint v2 has a built-in golines formatter.
func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

// PrepareCommand downloads the golines binary and adds it to the PATH.
//
// Deprecated: Use sggolangcilint.PrepareCommand instead; golangci-lint v2 has a built-in golines formatter.
func PrepareCommand(ctx context.Context) error {
	binDir := sg.FromToolsDir(name, version)
	binary := filepath.Join(binDir, name)
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostOS == sgtool.Darwin {
		hostArch = "all"
	}
	binURL := fmt.Sprintf(
		"https://github.com/segmentio/golines/"+
			"releases/download/v%s/golines_%s_%s_%s.tar.gz",
		version,
		version,
		hostOS,
		hostArch,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(binDir),
		sgtool.WithUntarGz(),
		sgtool.WithRenameFile(fmt.Sprintf("golines_%s_%s_%s/golines", version, hostOS, hostArch), name),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", name, err)
	}
	if err := os.Chmod(binary, 0o755); err != nil {
		return fmt.Errorf("unable to make %s command: %w", name, err)
	}
	return nil
}
