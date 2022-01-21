package mgcommitlint

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/magefile/mage/mg"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
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
	commitlintrc = mgpath.FromTools("commitlint", ".commitlintrc.js")
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	mg.CtxDeps(ctx, mg.F(Prepare.Commitlint, commitlintrc))
	return mgtool.Command(ctx, commandPath, args...)
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
	if err := mgtool.Command(ctx, "git", "fetch", "--tags").Run(); err != nil {
		panic(err)
	}
	return Command(ctx, args...)
}

type Prepare mgtool.Prepare

func (Prepare) Commitlint(ctx context.Context) error {
	toolDir := mgpath.FromTools("commitlint")
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
