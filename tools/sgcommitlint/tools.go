package sgcommitlint

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const packageJSONContent = `{
  "devDependencies": {
    "@commitlint/cli": "17.0.3",
    "@commitlint/config-conventional": "17.0.3"
  }
}`

const commitlintFileContent = `module.exports = {
  extends: ["@commitlint/config-conventional"],
  ignores: [
    // ignore dependabot commits
    (message) => /chore\(deps(-dev)?\): bump/.test(message),
  ],
}`

const name = "commitlint"

//nolint: gochecknoglobals
var commitlintrc = sg.FromToolsDir("commitlint", ".commitlintrc.js")

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
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
	toolDir := sg.FromToolsDir(name)
	binary := filepath.Join(toolDir, "node_modules", ".bin", name)
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
	sg.Logger(ctx).Println("installing packages...")
	if err := sg.Command(
		ctx,
		"npm",
		"--silent",
		"install",
		"--prefix",
		toolDir,
		"--no-save",
		"--no-audit",
		"--ignore-script",
	).Run(); err != nil {
		return err
	}
	if _, err := sgtool.CreateSymlink(binary); err != nil {
		return err
	}
	return nil
}
