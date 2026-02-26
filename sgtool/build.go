package sgtool

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"go.einride.tech/sage/sg"
)

var versionSuffixRegex = regexp.MustCompile(`/v\d+$`)

// trimVersionSuffix removes version suffixes like /v2, /v3, etc. from package names.
// This is needed because Go modules v2+ include the major version in the import path,
// but the installed binary name should not include this suffix.
func trimVersionSuffix(pkg string) string {
	return versionSuffixRegex.ReplaceAllString(pkg, "")
}

func GoInstall(ctx context.Context, pkg, version string) (string, error) {
	executable := sg.FromToolsDir("go", pkg, version, filepath.Base(trimVersionSuffix(pkg)))
	// Check if executable already exist
	if _, err := os.Stat(executable); err == nil {
		symlink, err := CreateSymlink(executable)
		if err != nil {
			return "", err
		}
		return symlink, nil
	}
	pkgVersion := fmt.Sprintf("%s@%s", pkg, version)
	sg.Logger(ctx).Printf("building %s...", pkgVersion)
	cmd := sg.Command(ctx, "go", "install", pkgVersion)
	cmd.Env = append(cmd.Env, "GOBIN="+filepath.Dir(executable))
	if err := cmd.Run(); err != nil {
		return "", err
	}
	symlink, err := CreateSymlink(executable)
	if err != nil {
		return "", err
	}
	return symlink, nil
}

// GoInstallWithGoVersion is like GoInstall but includes the Go version in the
// cache key. Use this for tools whose behavior depends on the Go runtime version,
// such as go-licenses which uses build.Default.GOROOT to detect stdlib packages.
func GoInstallWithGoVersion(ctx context.Context, pkg, version string) (string, error) {
	executable := sg.FromToolsDir("go", pkg, version, runtime.Version(), filepath.Base(trimVersionSuffix(pkg)))
	// Check if executable already exists
	if _, err := os.Stat(executable); err == nil {
		symlink, err := CreateSymlink(executable)
		if err != nil {
			return "", err
		}
		return symlink, nil
	}
	pkgVersion := fmt.Sprintf("%s@%s", pkg, version)
	sg.Logger(ctx).Printf("building %s...", pkgVersion)
	cmd := sg.Command(ctx, "go", "install", pkgVersion)
	cmd.Env = append(cmd.Env, "GOBIN="+filepath.Dir(executable))
	if err := cmd.Run(); err != nil {
		return "", err
	}
	symlink, err := CreateSymlink(executable)
	if err != nil {
		return "", err
	}
	return symlink, nil
}

// GoInstallWithModfile builds and installs a go binary given the package and a path
// to the local go.mod file.
func GoInstallWithModfile(ctx context.Context, pkg, file string) (string, error) {
	cmd := sg.Command(ctx, "go", "list", "-f", "{{.Module.Version}}", pkg)
	cmd.Dir = filepath.Dir(file)
	var b bytes.Buffer
	cmd.Stdout = &b
	if err := cmd.Run(); err != nil {
		return "", err
	}
	version := strings.TrimSpace(b.String())
	if version == "" {
		return "", fmt.Errorf("failed to determine version of package %s", pkg)
	}

	var b2 bytes.Buffer
	cmd = sg.Command(ctx, "go", "list", "-f", "{{.Target}}", pkg)
	cmd.Stdout = &b2
	cmd.Dir = filepath.Dir(file)
	if err := cmd.Run(); err != nil {
		return "", err
	}
	commandName := filepath.Base(strings.TrimSpace(b2.String()))

	executable := sg.FromToolsDir("go", pkg, version, commandName)
	// Check if executable already exist
	if _, err := os.Stat(executable); err == nil {
		symlink, err := CreateSymlink(executable)
		if err != nil {
			return "", err
		}
		return symlink, nil
	}
	sg.Logger(ctx).Printf("building %s...", pkg)
	cmd = sg.Command(ctx, "go", "install", pkg+"@"+version)
	cmd.Dir = filepath.Dir(file)
	cmd.Env = append(cmd.Env, "GOBIN="+filepath.Dir(executable))
	if err := cmd.Run(); err != nil {
		return "", err
	}
	symlink, err := CreateSymlink(executable)
	if err != nil {
		return "", err
	}
	return symlink, nil
}
