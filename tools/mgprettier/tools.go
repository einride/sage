package mgprettier

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-logr/logr"
	"go.einride.tech/mage-tools/mg"
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
	commandPath string
	prettierrc  = mgpath.FromToolsDir("prettier", ".prettierrc.js")
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	mg.Deps(ctx, Prepare.Prettier)
	return mgtool.Command(ctx, commandPath, args...)
}

func FormatMarkdownCommand(ctx context.Context) *exec.Cmd {
	args := []string{
		"--config",
		prettierrc,
		"--write",
		"**/*.md",
		"!" + mgpath.FromMageDir(),
	}
	return Command(ctx, args...)
}

func FormatYAML(ctx context.Context) *exec.Cmd {
	args := []string{
		"--config",
		prettierrc,
		"--write",
		"**/*.y*ml",
		"!" + mgpath.FromMageDir(),
	}
	return Command(ctx, args...)
}

type Prepare mgtool.Prepare

func (Prepare) Prettier(ctx context.Context) error {
	toolDir := mgpath.FromToolsDir("prettier")
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
	commandPath = symlink
	logr.FromContextOrDiscard(ctx).Info("installing packages...")
	return mgtool.Command(
		ctx,
		"npm",
		"--silent",
		"install",
		"--prefix",
		toolDir,
		"--no-save",
		"--no-audit",
		"--ignore-script",
	).Run()
}
