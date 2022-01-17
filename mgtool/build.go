package mgtool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mgpath"
)

func GoInstall(ctx context.Context, goPkg, version string) (string, error) {
	executable := filepath.Join(mgpath.Tools(), "go", goPkg, version, "bin", filepath.Base(goPkg))

	// Check if executable already exist
	if _, err := os.Stat(executable); err == nil {
		symlink, err := CreateSymlink(executable)
		if err != nil {
			return "", err
		}
		return symlink, nil
	}
	goPkgVer := fmt.Sprintf("%s@%s", goPkg, version)
	os.Setenv("GOBIN", filepath.Dir(executable))
	logr.FromContextOrDiscard(ctx).Info(fmt.Sprintf("Building %s...", goPkgVer))
	if err := sh.RunV("go", "install", goPkgVer); err != nil {
		return "", err
	}
	symlink, err := CreateSymlink(executable)
	if err != nil {
		return "", err
	}
	return symlink, nil
}
