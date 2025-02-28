package sggolangcilint

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	name    = "golangci-lint"
	version = "1.64.5"
)

//go:embed golangci.yml
var DefaultConfig []byte

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func defaultConfigPath() string {
	return sg.FromToolsDir(name, ".golangci.yml")
}

func CommandInDirectory(ctx context.Context, directory string, args ...string) *exec.Cmd {
	configPath := filepath.Join(directory, ".golangci.yml")
	if _, err := os.Lstat(configPath); errors.Is(err, os.ErrNotExist) {
		configPath = defaultConfigPath()
	}
	var excludeArg []string
	if directory == sg.FromSageDir() {
		excludeArg = append(excludeArg, "--exclude", "(is a global variable|is unused)")
	}
	cmdArgs := append([]string{"run", "--allow-parallel-runners", "-c", configPath}, args...)
	cmd := Command(ctx, append(cmdArgs, excludeArg...)...)
	cmd.Dir = directory
	return cmd
}

func joinErrorMessages(errs []error) error {
	var messages []string

	for _, err := range errs {
		if err != nil {
			messages = append(messages, err.Error())
		}
	}

	if len(messages) == 0 {
		return nil
	}

	return fmt.Errorf("multiple errors occurred:\n%s", strings.Join(messages, "\n"))
}

// Run GolangCI-Lint in every Go module from the root of the current git repo.
func Run(ctx context.Context, args ...string) error {
	var commands []*exec.Cmd
	if err := filepath.WalkDir(sg.FromGitRoot(), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "go.mod" {
			return nil
		}
		cmd := CommandInDirectory(ctx, filepath.Dir(path), args...)
		commands = append(commands, cmd)
		return cmd.Start()
	}); err != nil {
		return err
	}
	errs := make([]error, 0, len(commands))
	for _, cmd := range commands {
		errs = append(errs, cmd.Wait())
	}
	return joinErrorMessages(errs)
}

// Run GolangCI-Lint --fix in every Go module from the root of the current git repo.
func Fix(ctx context.Context, args ...string) error {
	var commands []*exec.Cmd
	if err := filepath.WalkDir(sg.FromGitRoot(), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "go.mod" {
			return nil
		}
		cmd := Command(ctx, append([]string{"run", "--allow-serial-runners", "-c", defaultConfigPath(), "--fix"}, args...)...)
		cmd.Dir = filepath.Dir(path)
		commands = append(commands, cmd)
		return cmd.Start()
	}); err != nil {
		return err
	}
	for _, cmd := range commands {
		if err := cmd.Wait(); err != nil {
			return err
		}
	}
	return nil
}

func PrepareCommand(ctx context.Context) error {
	toolDir := sg.FromToolsDir(name)
	binDir := filepath.Join(toolDir, version, "bin")
	binary := filepath.Join(binDir, name)
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	golangciLint := fmt.Sprintf("golangci-lint-%s-%s-%s", version, hostOS, hostArch)
	binURL := fmt.Sprintf(
		"https://github.com/golangci/golangci-lint/releases/download/v%s/%s.tar.gz",
		version,
		golangciLint,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(binDir),
		sgtool.WithUntarGz(),
		sgtool.WithRenameFile(fmt.Sprintf("%s/golangci-lint", golangciLint), name),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", name, err)
	}
	configPath := defaultConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(configPath, DefaultConfig, 0o600)
}
