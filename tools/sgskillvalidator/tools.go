package sgskillvalidator

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	// renovate: datasource=github-releases depName=agent-ecosystem/skill-validator
	version = "1.5.4"
	name    = "skill-validator"
)

// Command returns an *exec.Cmd for the skill-validator binary.
func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

// PrepareCommand downloads and installs the skill-validator binary.
func PrepareCommand(ctx context.Context) error {
	toolDir := sg.FromToolsDir(name, version)
	binary := filepath.Join(toolDir, name)
	binURL := fmt.Sprintf(
		"https://github.com/agent-ecosystem/skill-validator/releases/download/v%s/%s_%s_%s_%s.tar.gz",
		version,
		name,
		version,
		runtime.GOOS,
		runtime.GOARCH,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(toolDir),
		sgtool.WithUntarGz(),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", name, err)
	}
	return nil
}
