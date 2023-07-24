// Deprecated: tfsec is deprecated and has been replaced by trivy.
//
// See sgtrivy package for a replacement.
package sgtfsec

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
	version = "1.28.1"
	name    = "tfsec"
)

func CheckCommand(ctx context.Context, tfvars ...string) *exec.Cmd {
	args := []string{
		"--concise-output",
	}
	for _, tfvar := range tfvars {
		args = append(args, "--tfvars-file", tfvar)
	}
	return Command(ctx, args...)
}

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	toolDir := sg.FromToolsDir(name, version)
	binary := filepath.Join(toolDir, name)
	arch := runtime.GOARCH
	if arch == sgtool.X8664 {
		arch = sgtool.AMD64
	}
	binURL := fmt.Sprintf(
		"https://github.com/aquasecurity/tfsec/releases/download/v%s/tfsec_%s_%s_%s.tar.gz",
		version,
		version,
		runtime.GOOS,
		arch,
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
