package mgprettier

import (
	"context"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

const packageJSONContent = `{
  "devDependencies": {
    "@einride/prettier-config": "1.2.0",
    "prettier": "2.5.0"
  }
}`

const prettierConfigContent = `module.exports = {
	...require("@einride/prettier-config"),
}`

// nolint: gochecknoglobals
var (
	executable string
	prettierrc = filepath.Join(mgpath.Tools(), "prettier", ".prettierrc.js")
)

type Prepare mgtool.Prepare

func (Prepare) Prettier(ctx context.Context) error {
	return prepare(ctx)
}

func FormatMarkdown(ctx context.Context) error {
	logger := mglog.Logger("prettier")
	ctx = logr.NewContext(ctx, logger)
	mg.CtxDeps(ctx, prepare)
	args := []string{
		"--config",
		prettierrc,
		"--write",
		"**/*.md",
		"!" + mgpath.MageDir,
	}
	logger.Info("formatting Markdown files...")
	return sh.RunV(executable, args...)
}

func FormatYAML(ctx context.Context) error {
	logger := mglog.Logger("prettier")
	ctx = logr.NewContext(ctx, logger)
	mg.CtxDeps(ctx, prepare)
	args := []string{
		"--config",
		prettierrc,
		"--write",
		"**/*.y*ml",
		"!" + mgpath.MageDir,
	}
	logger.Info("formatting YAML files...")
	return sh.RunV(executable, args...)
}

func prepare(ctx context.Context) error {
	// Check if npm is installed
	if err := sh.Run("npm", "version"); err != nil {
		return err
	}

	toolDir := filepath.Join(mgpath.Tools(), "prettier")
	binary := filepath.Join(toolDir, "node_modules", ".bin", "prettier")
	packageJSON := filepath.Join(toolDir, "package.json")

	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(prettierrc, []byte(prettierConfigContent), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(packageJSON, []byte(packageJSONContent), 0o600); err != nil {
		return err
	}
	symlink, err := mgtool.CreateSymlink(binary)
	if err != nil {
		return err
	}
	executable = symlink
	logr.FromContextOrDiscard(ctx).Info("installing packages...")
	return sh.Run(
		"npm",
		"--silent",
		"install",
		"--prefix",
		toolDir,
		"--no-save",
		"--no-audit",
		"--ignore-script",
	)
}
