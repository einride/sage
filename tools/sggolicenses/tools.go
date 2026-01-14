package sggolicenses

import (
	"context"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	name = "go-licenses"
	// renovate: datasource=go depName=github.com/google/go-licenses/v2
	version = "v2.0.1"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

// CheckDir checks for disallowed types of Go licenses in a specific directory.
// By default, Google's forbidden and restricted types are disallowed.
func CheckDir(ctx context.Context, directory string, disallowedTypes ...string) error {
	args := []string{
		"check",
		".",
		"--skip_headers",
		"--ignore",
		"github.com/einride",
		"--ignore",
		"go.einride.tech",
	}
	if len(disallowedTypes) > 0 {
		args = append(args, "--disallowed_types="+strings.Join(disallowedTypes, ","))
	} else {
		args = append(args, "--disallowed_types=forbidden,restricted")
	}
	cmd := Command(ctx, args...)
	cmd.Dir = directory
	return cmd.Run()
}

// Check for disallowed types of Go licenses.
// By default, Google's forbidden and restricted types are disallowed.
func Check(ctx context.Context, disallowedTypes ...string) error {
	var goModPaths []string
	if err := filepath.WalkDir(sg.FromGitRoot(), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "go.mod" {
			return nil
		}
		goModPaths = append(goModPaths, filepath.Dir(path))
		return nil
	}); err != nil {
		return err
	}

	args := []string{
		"check",
		"--skip_headers",
		"--include_tests",
		"--ignore", "github.com/einride",
		"--ignore", "go.einride.tech",
		"--ignore", "gotest.tools/v3", // go-licenses is unable to identify this license; Apache-2.0
	}
	if len(disallowedTypes) > 0 {
		args = append(args, "--disallowed_types="+strings.Join(disallowedTypes, ","))
	} else {
		args = append(args, "--disallowed_types=forbidden,restricted")
	}
	args = append(args, "./...")

	commands := make([]*exec.Cmd, 0, len(goModPaths))
	for _, path := range goModPaths {
		sg.Logger(ctx).Println("checking Go licenses for", path)
		cmd := Command(ctx, args...)
		cmd.Dir = path
		commands = append(commands, cmd)
		if err := cmd.Start(); err != nil {
			return err
		}
	}
	for _, cmd := range commands {
		if err := cmd.Wait(); err != nil {
			return err
		}
	}
	return nil
}

func PrepareCommand(ctx context.Context) error {
	_, err := sgtool.GoInstall(ctx, "github.com/google/go-licenses/v2", version)
	return err
}
