package sggolicenses

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"runtime"
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

func checkDir(ctx context.Context, directory string, appendArgs []string) error {
	args := []string{
		"check",
		".",
		"--skip_headers",
		"--ignore",
		"github.com/einride",
		"--ignore",
		"go.einride.tech",
	}

	args = append(args, appendArgs...)

	cmd := Command(ctx, args...)
	cmd.Dir = directory
	// go-licenses tries to exclude standard library packages by checking if they are prefixed
	// with `runtime.GOROOT()`. However, if the go-licenses tool is not run with a GOROOT environment variable,
	// that call will return the GOROOT path used during build time of go-licenses. This typically works on Linux,
	// but on macOS with Homebrew, the GOROOT is version prefixed, which breaks as soon as Go is upgraded.
	// For example: /opt/homebrew/Cellar/go/1.19.4/libexec
	//
	// As a workaround, add the GOROOT environment variable to the result of `runtime.GOROOT()` called here.
	// This should work as the Sage binary is built on the same machine that executes it.
	// See: https://github.com/google/go-licenses/issues/149
	cmd.Env = append(cmd.Env, fmt.Sprintf("GOROOT=%s", runtime.GOROOT()))
	return cmd.Run()
}

// Check for disallowed types of Go licenses in a specific directory.
// By default, Google's forbidden and restricted types are disallowed.
func CheckDir(ctx context.Context, directory string, disallowedTypes ...string) error {
	var appendArgs []string
	if len(disallowedTypes) > 0 {
		appendArgs = append(appendArgs, "--disallowed_types="+strings.Join(disallowedTypes, ","))
	} else {
		appendArgs = append(appendArgs, "--disallowed_types=forbidden,restricted")
	}

	return checkDir(ctx, directory, appendArgs)
}

// Check for disallowed types of Go licenses in a specific directory.
// By default, Google's forbidden and restricted types are disallowed.
// Allows to ignore some dependencies by specifying their package path (ie "github.com/einride/sage").
func CheckDirWithIgnored(
	ctx context.Context,
	directory string,
	ignoredPackages []string,
	disallowedTypes ...string,
) error {
	appendArgs := make([]string, 0, len(ignoredPackages)+1)

	for _, ignored := range ignoredPackages {
		appendArgs = append(appendArgs, "--ignore")
		appendArgs = append(appendArgs, ignored)
	}

	if len(disallowedTypes) > 0 {
		appendArgs = append(appendArgs, "--disallowed_types="+strings.Join(disallowedTypes, ","))
	} else {
		appendArgs = append(appendArgs, "--disallowed_types=forbidden,restricted")
	}

	return checkDir(ctx, directory, appendArgs)
}

func check(ctx context.Context, appendArgs []string) error {
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

		args = append(args, appendArgs...)
		cmd := Command(ctx, args...)
		cmd.Dir = filepath.Dir(path)
		// go-licenses tries to exclude standard library packages by checking if they are prefixed
		// with `runtime.GOROOT()`. However, if the go-licenses tool is not run with a GOROOT environment variable,
		// that call will return the GOROOT path used during build time of go-licenses. This typically works on Linux,
		// but on macOS with Homebrew, the GOROOT is version prefixed, which breaks as soon as Go is upgraded.
		// For example: /opt/homebrew/Cellar/go/1.19.4/libexec
		//
		// As a workaround, add the GOROOT environment variable to the result of `runtime.GOROOT()` called here.
		// This should work as the Sage binary is built on the same machine that executes it.
		// See: https://github.com/google/go-licenses/issues/149
		cmd.Env = append(cmd.Env, fmt.Sprintf("GOROOT=%s", runtime.GOROOT()))
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

// Check for disallowed types of Go licenses.
// By default, Google's forbidden and restricted types are disallowed.
func Check(ctx context.Context, disallowedTypes ...string) error {
	var appendArgs []string
	if len(disallowedTypes) > 0 {
		appendArgs = append(appendArgs, "--disallowed_types="+strings.Join(disallowedTypes, ","))
	} else {
		appendArgs = append(appendArgs, "--disallowed_types=forbidden,restricted")
	}

	return check(ctx, appendArgs)
}

// Check for disallowed types of Go licenses.
// By default, Google's forbidden and restricted types are disallowed.
// Allows to ignore some dependencies by specifying their package path (ie "github.com/einride/sage").
func CheckWithIgnored(ctx context.Context, ignoredPackages []string, disallowedTypes ...string) error {
	appendArgs := make([]string, 0, len(ignoredPackages)+1)
	for _, ignored := range ignoredPackages {
		appendArgs = append(appendArgs, "--ignore")
		appendArgs = append(appendArgs, ignored)
	}

	if len(disallowedTypes) > 0 {
		appendArgs = append(appendArgs, "--disallowed_types="+strings.Join(disallowedTypes, ","))
	} else {
		appendArgs = append(appendArgs, "--disallowed_types=forbidden,restricted")
	}

	return check(ctx, appendArgs)
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
