package sgtrivy

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

//go:embed trivyignore
var defaultConfig []byte

const (
	version = "0.58.1"
	name    = "trivy"
)

func defaultConfigPath() string {
	return sg.FromToolsDir(name, ".trivyignore")
}

// CheckTerraformCommand checks terraform configuration on the given dir
// for any known security misconfigurations.
// It includes a default .trivyignore.yaml which can be
// overridden by setting a .trivyignore.yaml in the git root.
func CheckTerraformCommand(ctx context.Context, dir string) *exec.Cmd {
	args := []string{
		"config",
		"--misconfig-scanners",
		"terraform,terraformplan-json,terraformplan-snapshot",
		"--exit-code",
		"1",
		dir,
	}

	configPath := sg.FromGitRoot(".trivyignore")
	if _, err := os.Lstat(configPath); errors.Is(err, os.ErrNotExist) {
		configPath = defaultConfigPath()
	}
	args = append(args, "--ignorefile", configPath)

	return Command(ctx, args...)
}

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	toolDir := sg.FromToolsDir(name, version)
	binary := filepath.Join(toolDir, name)
	var goos, goarch string
	switch runtime.GOOS {
	case "linux":
		goos = "Linux"
	case "darwin":
		goos = "macOS"
	default:
		return fmt.Errorf("unsupported OS in sgtrivy package %s", runtime.GOOS)
	}
	switch runtime.GOARCH {
	case sgtool.AMD64:
		goarch = "64bit"
	case sgtool.ARM64:
		goarch = "ARM64"
	default:
		return fmt.Errorf("unsupported ARCH in sgtrivy package %s", runtime.GOARCH)
	}

	binURL := fmt.Sprintf(
		"https://github.com/aquasecurity/trivy/releases/download/v%s/trivy_%s_%s-%s.tar.gz",
		version,
		version,
		goos,
		goarch,
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

	configPath := defaultConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(configPath, defaultConfig, 0o600)
}
