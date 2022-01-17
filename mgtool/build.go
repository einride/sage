package mgtool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mgpath"
)

func GoInstall(ctx context.Context, pkg, version string) (string, error) {
	executable := filepath.Join(mgpath.Tools(), "go", pkg, version, filepath.Base(pkg))
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
	if err := sh.RunWithV(
		map[string]string{"GOBIN": filepath.Dir(executable)},
		"go",
		"install",
		pkgVersion,
	); err != nil {
		return "", err
	}
	symlink, err := CreateSymlink(executable)
	if err != nil {
		return "", err
	}
	return symlink, nil
}

func GoInstallWithModfile(ctx context.Context, pkg, file string) (string, error) {
	cleanup := mgpath.ChangeWorkDir(filepath.Dir(file))
	defer cleanup()
	version, err := sh.Output("go", "list", "-f", "{{.Module.Version}}", pkg)
	if err != nil {
		return "", err
	}
	version = strings.TrimSpace(version)
	if version == "" {
		return "", fmt.Errorf("failed to determine version of package %s", pkg)
	}
	executable := filepath.Join(mgpath.Tools(), "go", pkg, version, filepath.Base(pkg))
	// Check if executable already exist
	if _, err := os.Stat(executable); err == nil {
		symlink, err := CreateSymlink(executable)
		if err != nil {
			return "", err
		}
		return symlink, nil
	}
	logr.FromContextOrDiscard(ctx).Info("building", "pkg", pkg)
	if err := sh.RunWithV(
		map[string]string{"GOBIN": filepath.Dir(executable)},
		"go",
		"install",
		pkg,
	); err != nil {
		return "", err
	}
	symlink, err := CreateSymlink(executable)
	if err != nil {
		return "", err
	}
	return symlink, nil
}
