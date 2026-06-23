package sgrumdl

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	name = "rumdl"

	// renovate: datasource=github-releases depName=rvben/rumdl
	version = "0.2.22"
)

//go:embed rumdl.toml
var config []byte

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	args = setDefaultArgs(args)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

// setDefaultArgs to format the repository using the bundled config. rumdl
// formats the current directory recursively and respects .gitignore.
func setDefaultArgs(args []string) []string {
	if len(args) != 0 {
		return args
	}
	return []string{"fmt", "--config", configPath(), "."}
}

func configPath() string {
	return sg.FromToolsDir(name, "rumdl.toml")
}

func PrepareCommand(ctx context.Context) error {
	toolDir := sg.FromToolsDir(name, version)
	binary := filepath.Join(toolDir, name)
	arch := runtime.GOARCH
	switch arch {
	case sgtool.AMD64:
		arch = sgtool.X8664
	case sgtool.ARM64:
		arch = "aarch64"
	}
	target := fmt.Sprintf("%s-unknown-linux-gnu", arch)
	if runtime.GOOS == sgtool.Darwin {
		target = fmt.Sprintf("%s-apple-darwin", arch)
	}
	binURL := fmt.Sprintf(
		"https://github.com/rvben/rumdl/releases/download/v%s/rumdl-v%s-%s.tar.gz",
		version,
		version,
		target,
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
	if err := writeConfig(); err != nil {
		return fmt.Errorf("unable to write %s config: %w", name, err)
	}
	return nil
}

// writeConfig writes the bundled config to the tools directory so it can be
// passed to rumdl with the --config flag.
func writeConfig() error {
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, config, 0o600)
}
