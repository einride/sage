package sgyamlfmt

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

//go:embed yamlfmt.yaml
var DefaultConfig []byte

const (
	name              = "yamlfmt"
	version           = "0.5.0"
	defaultConfigName = ".yamlfmt"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

// Run, runs the google/yamlfmt tool.
// If no config file is found in the git root then it will run with the default config.
func Run(ctx context.Context, args ...string) error {
	defaultConfigPath := sg.FromToolsDir(name, defaultConfigName)
	if err := os.MkdirAll(filepath.Dir(defaultConfigPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(defaultConfigPath, DefaultConfig, 0o600); err != nil {
		return err
	}
	configPath := sg.FromGitRoot(defaultConfigName)
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		configPath = defaultConfigPath
	}

	cmd := Command(ctx, append([]string{"-conf", configPath}, args...)...)
	cmd.Dir = sg.FromGitRoot()
	return cmd.Run()
}

func PrepareCommand(ctx context.Context) error {
	binDir := sg.FromToolsDir(name, version)
	binary := filepath.Join(binDir, name)
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostArch == sgtool.AMD64 {
		hostArch = sgtool.X8664
	}

	binURL := fmt.Sprintf(
		"https://github.com/google/yamlfmt"+
			"/releases/download/v%s/yamlfmt_%s_%s_%s.tar.gz",
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
