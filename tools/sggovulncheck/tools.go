package sggovulncheck

import (
	"context"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	name    = "govulncheck"
	version = "v1.1.3"
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
			for _, modulePath := range modulePaths {
				if modulePath == path {
					shouldRunInModulePath = true
					break
				}
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

func PrepareCommand(ctx context.Context) error {
	_, err := sgtool.GoInstall(ctx, "golang.org/x/vuln/cmd/govulncheck", version)
	return err
}
