package sgprettier

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
    "@einride/prettier-config": "2.0.0",
    "prettier": "2.7.1"
  }
}`

const prettierConfigContent = `module.exports = {
	...require("@einride/prettier-config"),
}`

const name = "prettier"

//nolint:gochecknoglobals
var prettierrc = sg.FromToolsDir("prettier", ".prettierrc.js")

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func FormatMarkdownCommand(ctx context.Context) *exec.Cmd {
	args := []string{
		"--config",
		prettierrc,
		"--write",
		"**/*.md",
		"!" + sg.FromSageDir(),
	}
	return Command(ctx, args...)
}

func FormatYAML(ctx context.Context) *exec.Cmd {
	args := []string{
		"--config",
		prettierrc,
		"--write",
		"**/*.y*ml",
		"!" + sg.FromSageDir(),
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
	if err := os.WriteFile(prettierrc, []byte(prettierConfigContent), 0o600); err != nil {
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
