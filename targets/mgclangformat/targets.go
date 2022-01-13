package mgclangformat

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgpath"
)

const version = "1.6.0"

// nolint: gochecknoglobals
var executable string

func ClangFormat(path string) error {
	logger := mglog.Logger("clang-format")
	mg.Deps(prepare)
	protoFiles, err := mgpath.FileExtensionsIn(".proto", path)
	if err != nil {
		return err
	}
	args := []string{"-i", "--style={BasedOnStyle: Google, ColumnLimit: 0, Language: Proto}", "--verbose"}
	args = append(args, protoFiles...)
	logger.Info("formatting proto files...")
	return sh.Run(executable, args...)
}

func prepare() error {
	// Check if npm is installed
	if err := sh.Run("npm", "version"); err != nil {
		return err
	}

	var archiveName string
	switch strings.Split(runtime.GOOS, "/")[0] {
	case "linux":
		archiveName = "linux_x64"
	case "darwin":
		archiveName = "darwin_x64"
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	toolDir := filepath.Join(mgpath.Tools(), "clang-format")
	binary := filepath.Join(toolDir, "node_modules", "clang-format", "bin", archiveName, "clang-format")

	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return err
	}
	if err := sh.Run(
		"npm",
		"--silent",
		"install",
		"--prefix",
		toolDir,
		"--no-save",
		"--no-audit",
		"clang-format@"+version,
	); err != nil {
		return err
	}
	executable = binary
	return nil
}
