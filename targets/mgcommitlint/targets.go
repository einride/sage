package mgcommitlint

import (
	"context"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgtool"
)

const packageJSONContent = `{
  "devDependencies": {
    "@commitlint/cli": "^11.0.0",
    "@commitlint/config-conventional": "^11.0.0"
  }
}`

const commitlintFileContent = `module.exports = {
  extends: ['@commitlint/config-conventional'],
  ignores: [
    // ignore dependabot messages
    (message) => /^Bumps \[.+]\(.+\) from .+ to .+\.$/m.test(message),
  ],
};`

var executable string

func Commitlint(ctx context.Context, branch string) error {
	logger := mglog.Logger("commitlint")
	ctx = logr.NewContext(ctx, logger)
	commitlintrc := filepath.Join(mgtool.GetPath(), "commitlint", ".commitlintrc.js")
	mg.CtxDeps(ctx, mg.F(prepare, commitlintrc))
	args := []string{
		"--config",
		commitlintrc,
		"--from",
		"origin/" + branch,
		"--to",
		"HEAD",
	}
	logr.FromContextOrDiscard(ctx).Info("linting commit messages...")
	if err := sh.Run("git", "fetch", "--tags"); err != nil {
		return err
	}
	return sh.RunV(executable, args...)
}

func prepare(ctx context.Context, commitlintrc string) error {
	// Check if npm is installed
	if err := sh.Run("npm", "version"); err != nil {
		return err
	}

	toolDir := filepath.Join(mgtool.GetPath(), "commitlint")
	binary := filepath.Join(toolDir, "node_modules", ".bin", "commitlint")
	packageJSON := filepath.Join(toolDir, "package.json")

	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(commitlintrc, []byte(commitlintFileContent), 0o644); err != nil {
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
