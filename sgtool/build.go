package sgtool

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.einride.tech/sage/sg"
)

func GoInstall(ctx context.Context, pkg, version string) (string, error) {
	executable := sg.FromToolsDir("go", pkg, version, filepath.Base(pkg))
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
