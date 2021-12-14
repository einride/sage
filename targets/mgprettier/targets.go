package mgprettier

import (
	"context"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/tools"
)

const packageJSONContent = `{
  "devDependencies": {
    "prettier": "^2.4.1",
    "@einride/prettier-config": "^1.2.0"
  }
}`

var executable string

func FormatMarkdown(ctx context.Context) error {
	logger := mglog.Logger("prettier")
	ctx = logr.NewContext(ctx, logger)
	mg.CtxDeps(ctx, prepare)
	args := []string{
		"--write",
		"**/*.md",
		"!.tools",
	}
	logger.Info("formatting Markdown files...")
	return sh.RunV(executable, args...)
}

func prepare(ctx context.Context) error {
	// Check if npm is installed
	if err := sh.Run("npm", "version"); err != nil {
		return err
	}

	toolDir := filepath.Join(tools.GetPath(), "prettier")
	binary := filepath.Join(toolDir, "node_modules", ".bin", "prettier")
	packageJSON := filepath.Join(toolDir, "package.json")

	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return err
	}

	if err := os.WriteFile(packageJSON, []byte(packageJSONContent), 0o644); err != nil {
		return err
	}

	executable = binary
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
