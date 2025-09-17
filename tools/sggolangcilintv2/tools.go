// This package is considered experimental and may change.

package sggolangcilintv2

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
	"text/template"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	name    = "golangci-lint"
	version = "2.4.0"

	RunRelativePathModeGitRoot    = "gitroot"
	RunRelativePathModeGomod      = "gomod"
	RunRelativePathModeCfg        = "cfg" // WARNING: not recommended; paths will be relative to .sage/tools/golangci-lint
	RunRelativePathModeWorkingDir = "wd"  // WARNING: not recommended according to official docs
)

type RunRelativePathMode string

//go:embed golangci.yml.tmpl
var configTemplate []byte

// Config holds values which will be applied to the template, ultimately resulting in the final golangci.yml file.
type Config struct {
	// The mode used to evaluate relative paths. It's used by exclusions, Go plugins and some linters.
	RunRelativePathMode RunRelativePathMode
	// Which file paths to exclude from issue reporting. The paths will still be analyzed.
	// Golangci-Lint will:
	// - replace "/" with the current OS file path separator to properly work on Windows.
	// - allow you to optionally use regexps here, like ".*\\.my\\.go$".
	// - treat paths relative to the setting of RunRelativePathMode.
	LintersExclusionsPaths []string
	// Which file paths to exclude from issue reporting. The paths will still be analyzed.
	// Golangci-Lint will:
	// - replace "/" with the current OS file path separator to properly work on Windows.
	// - allow you to optionally use regexps here, like ".*\\.my\\.go$".
	// - treat paths relative to the setting of RunRelativePathMode.
	FormattersExclusionsPaths []string
}

func Command(ctx context.Context, config Config, args ...string) *exec.Cmd {
	sg.Deps(ctx, func(ctx context.Context) error {
		return PrepareCommand(ctx, config)
	})
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func defaultConfigPath() string {
	return sg.FromToolsDir(name, ".golangci.yml")
}

func CommandRunInDirectory(ctx context.Context, config Config, directory string, args ...string) *exec.Cmd {
	configPath := filepath.Join(directory, ".golangci.yml")
	if _, err := os.Lstat(configPath); errors.Is(err, os.ErrNotExist) {
		configPath = defaultConfigPath()
	}
	cmdArgs := append([]string{"run", "--allow-parallel-runners", "-c", configPath}, args...)
	cmd := Command(ctx, config, cmdArgs...)
	cmd.Dir = directory
	return cmd
}

func CommandFmtInDirectory(ctx context.Context, config Config, directory string, args ...string) *exec.Cmd {
	configPath := filepath.Join(directory, ".golangci.yml")
	if _, err := os.Lstat(configPath); errors.Is(err, os.ErrNotExist) {
		configPath = defaultConfigPath()
	}
	cmdArgs := append([]string{"fmt", "-c", configPath}, args...)
	cmd := Command(ctx, config, cmdArgs...)
	cmd.Dir = directory
	return cmd
}

// Run GolangCI-Lint in every Go module from the root of the current git repo.
func Run(ctx context.Context, config Config, args ...string) error {
	var commands []*exec.Cmd
	if err := filepath.WalkDir(sg.FromGitRoot(), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "go.mod" {
			return nil
		}
		fileInfo, err := d.Info()
		if err != nil {
			return err
		}
		// ignore if it's an empty go.mod
		if fileInfo.Size() == 0 {
			return filepath.SkipAll
		}
		cmd := CommandRunInDirectory(ctx, config, filepath.Dir(path), args...)
		commands = append(commands, cmd)
		return cmd.Start()
	}); err != nil {
		return err
	}
	errs := make([]error, 0, len(commands))
	for _, cmd := range commands {
		errs = append(errs, cmd.Wait())
	}
	return errors.Join(errs...)
}

// Run GolangCI-Lint --fix in every Go module from the root of the current git repo.
func Fix(ctx context.Context, config Config, args ...string) error {
	var commands []*exec.Cmd
	if err := filepath.WalkDir(sg.FromGitRoot(), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "go.mod" {
			return nil
		}
		fileInfo, err := d.Info()
		if err != nil {
			return err
		}
		// ignore if it's an empty go.mod
		if fileInfo.Size() == 0 {
			return nil
		}
		cmd := Command(
			ctx,
			config,
			append([]string{"run", "--allow-serial-runners", "-c", defaultConfigPath(), "--fix"}, args...)...,
		)
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

// Run `golangci-lint fmt` in every Go module from the root of the current git repo.
// This writes the formatting to the files. Add the `--diff` argument if you don't want it to write to the files.
func Fmt(ctx context.Context, config Config, args ...string) error {
	var commands []*exec.Cmd
	if err := filepath.WalkDir(sg.FromGitRoot(), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "go.mod" {
			return nil
		}
		cmd := Command(ctx, config, append([]string{"fmt", "-c", defaultConfigPath()}, args...)...)
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

func PrepareCommand(ctx context.Context, config Config) error {
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
	err := CreateConfigFromTemplate(ctx, configPath, config)
	if err != nil {
		return fmt.Errorf("unable to create config file: %w", err)
	}

	return nil
}

func CreateConfigFromTemplate(ctx context.Context, outputPath string, config Config) error {
	if config.RunRelativePathMode == "" {
		config.RunRelativePathMode = RunRelativePathModeGitRoot
		sg.Logger(ctx).Printf("Using default relative path mode: %s\n", config.RunRelativePathMode)
	}
	tmpl, err := template.New("golangci.yml").Parse(string(configTemplate))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, config)
}
