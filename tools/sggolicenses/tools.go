package sggolicenses

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	name    = "go-licenses"
	version = "v1.6.0"
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
	var commands []*exec.Cmd
	if err := filepath.WalkDir(sg.FromGitRoot(), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "go.mod" {
			return nil
		}
		goModulePath, err := loadGoModulePath(ctx, path)
		if err != nil {
			return err
		}
		args := []string{
			"check",
			goModulePath,
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
	_, err := sgtool.GoInstall(ctx, "github.com/google/go-licenses", version)
	return err
}

func loadGoModulePath(ctx context.Context, goModFile string) (string, error) {
	var out bytes.Buffer
	cmd := sg.Command(ctx, "go", "mod", "edit", "-json")
	cmd.Dir = filepath.Dir(goModFile)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	var modFile struct {
		Module struct {
			Path string
		}
	}
	if err := json.Unmarshal(out.Bytes(), &modFile); err != nil {
		return "", err
	}
	if modFile.Module.Path == "" {
		return "", fmt.Errorf("no module path found for %s", goModFile)
	}
	return modFile.Module.Path, nil
}
