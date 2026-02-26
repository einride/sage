// Package sggolicenses provides sage integration for the go-licenses tool.
//
// Troubleshooting: If go-licenses fails with errors like "Package X does not
// have module info", this is a known incompatibility between go-licenses and
// Go toolchain version mismatches (https://github.com/google/go-licenses/pull/329).
// The go-licenses binary uses build.Default.GOROOT at runtime to detect stdlib
// packages — if your local Go version differs from the toolchain directive in
// go.mod, the GOROOT path won't match and stdlib packages are misidentified.
//
// To fix this, update your local Go installation to match the version in go.mod,
// or set GOTOOLCHAIN=go<version>+auto (e.g. GOTOOLCHAIN=go1.25.7+auto) to
// ensure the correct toolchain is used consistently.
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
	name    = "go-licenses"
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

// PrepareCommand installs go-licenses using Go-version-aware caching.
// go-licenses uses build.Default.GOROOT at runtime to detect stdlib packages,
// so the binary must be rebuilt when the Go version changes.
func PrepareCommand(ctx context.Context) error {
	_, err := sgtool.GoInstallWithGoVersion(ctx, "github.com/google/go-licenses/v2", version)
	return err
}
