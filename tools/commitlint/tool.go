package commitlint

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/einride/mage-tools/tools"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var Binary string

func Commitlint(branch string) error {
	mg.Deps(commitlint)

	commitlintrc := filepath.Join(tools.GetPath(), "commitlint", ".commitlintrc.js")

	commitlintFileContent := `module.exports = {
  extends: ['@commitlint/config-conventional'],
  rules: {
    // Treat as warning until Dependabot supports commitlint.
    // https://github.com/dependabot/dependabot-core/issues/2445
    "body-max-line-length": [1, "always", 100],
  }
};`
	if err := os.WriteFile(commitlintrc, []byte(commitlintFileContent), 0o644); err != nil {
		return err
	}

	args := []string{
		"--config",
		commitlintrc,
		"--from",
		"origin/" + branch,
		"--to",
		"HEAD",
	}
	fmt.Println("[commitlint] linting commit messages...")
	if err := sh.Run("git", "fetch", "--tags"); err != nil {
		return err
	}
	return sh.RunV(Binary, args...)
}

func commitlint() error {
	// Check if npm is installed
	if err := sh.Run("npm", "version"); err != nil {
		return err
	}

	toolDir := filepath.Join(tools.GetPath(), "commitlint")
	binary := filepath.Join(toolDir, "node_modules", ".bin", "commitlint")
	packageJSON := filepath.Join(toolDir, "package.json")

	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return err
	}

	packageFileContent := `{
  "devDependencies": {
    "@commitlint/cli": "^11.0.0",
    "@commitlint/config-conventional": "^11.0.0"
  }
}`

	if err := os.WriteFile(packageJSON, []byte(packageFileContent), 0o644); err != nil {
		return err
	}

	Binary = binary

	fmt.Println("[commitlint] installing packages...")
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
