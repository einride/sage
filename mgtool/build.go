package mgtool

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"
	"go.einride.tech/mage-tools/mgpath"
)

func GoInstall(ctx context.Context, pkg, version string) (string, error) {
	executable := mgpath.FromTools("go", pkg, version, filepath.Base(pkg))
	// Check if executable already exist
	if _, err := os.Stat(executable); err == nil {
		symlink, err := CreateSymlink(executable)
		if err != nil {
			return "", err
		}
		return symlink, nil
	}
	pkgVersion := fmt.Sprintf("%s@%s", pkg, version)
	logr.FromContextOrDiscard(ctx).Info("building...", "pkg", pkgVersion)
	cmd := Command("go", "install", pkgVersion)
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

func GoInstallWithModfile(ctx context.Context, pkg, file string) (string, error) {
	cmd := Command("go", "list", "-f", "{{.Module.Version}}", pkg)
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
	executable := mgpath.FromTools("go", pkg, version, filepath.Base(pkg))
	// Check if executable already exist
	if _, err := os.Stat(executable); err == nil {
		symlink, err := CreateSymlink(executable)
		if err != nil {
			return "", err
		}
		return symlink, nil
	}
	logr.FromContextOrDiscard(ctx).Info("building", "pkg", pkg)
	cmd = Command("go", "install", pkg)
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
