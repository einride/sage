package sggovulncheck

import (
	"context"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"slices"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	name = "govulncheck"

	// renovate: datasource=go depName=golang.org/x/vuln
	version = "v1.1.4"
)

// Command returns an [*exec.Cmd] for govulncheck.
func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

// RunAll runs govulncheck on all the specified module paths, or all module paths from the current git root by default.
func RunAll(ctx context.Context, modulePaths ...string) error {
	var commands []*exec.Cmd
	if err := filepath.WalkDir(sg.FromGitRoot(), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "go.mod" {
			return nil
		}
		if len(modulePaths) > 0 {
			var shouldRunInModulePath bool
			if slices.Contains(modulePaths, path) {
				shouldRunInModulePath = true
			}
			if !shouldRunInModulePath {
				return nil
			}
		}
		relativePath, err := filepath.Rel(sg.FromGitRoot(), path)
		if err != nil {
			return err
		}
		cmd := Command(
			sg.AppendLoggerPrefix(ctx, fmt.Sprintf(" (%s): ", relativePath)),
			"-C",
			filepath.Dir(path),
			"./...",
		)
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

// PrepareCommand installs govulncheck using Go-version-aware caching.
// govulncheck uses go/packages to load and analyze dependencies. When the
// binary is built with an older Go version than what is currently installed,
// package loading can fail with errors like:
//
//	file requires newer Go version go1.N (application built with go1.M)
//
// This happens because dependencies may have //go:build go1.N constraints
// that are incompatible with the Go version embedded in a stale binary.
// Rebuilding when the Go version changes ensures compatibility.
func PrepareCommand(ctx context.Context) error {
	_, err := sgtool.GoInstallWithGoVersion(ctx, "golang.org/x/vuln/cmd/govulncheck", version)
	return err
}
