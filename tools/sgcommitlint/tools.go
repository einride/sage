package sgcommitlint

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-logr/logr"
	"go.einride.tech/sage/mgtool"
	"go.einride.tech/sage/sg"
)

const packageJSONContent = `{
  "devDependencies": {
    "@commitlint/cli": "15.0.0",
    "@commitlint/config-conventional": "15.0.0"
  }
}`

const commitlintFileContent = `module.exports = {
  extends: ['@commitlint/config-conventional'],
  ignores: [
    // ignore dependabot messages
    (message) => /^Bumps \[.+]\(.+\) from .+ to .+\.$/m.test(message),
  ],
};`

// nolint: gochecknoglobals
var (
	commandPath  string
	commitlintrc = sg.FromToolsDir("commitlint", ".commitlintrc.js")
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, commandPath, args...)
}

func LintCommand(ctx context.Context, branch string) *exec.Cmd {
	args := []string{
		"--config",
		commitlintrc,
		"--from",
		"origin/" + branch,
		"--to",
		"HEAD",
	}
	if err := sg.Command(ctx, "git", "fetch", "--tags").Run(); err != nil {
		panic(err)
	}
	return Command(ctx, args...)
}

func PrepareCommand(ctx context.Context) error {
	toolDir := sg.FromToolsDir("commitlint")
	binary := filepath.Join(toolDir, "node_modules", ".bin", "commitlint")
	packageJSON := filepath.Join(toolDir, "package.json")
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(commitlintrc, []byte(commitlintFileContent), 0o600); err != nil {
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
	return sg.Command(
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
