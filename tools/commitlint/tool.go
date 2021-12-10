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

	commitlintrc := filepath.Join(tools.Path, "commitlint", ".commitlintrc.js")

	commitlintFileContent := `module.exports = {
  extends: ['@commitlint/config-conventional'],
  rules: {
    // Treat as warning until Dependabot supports commitlint.
    // https://github.com/dependabot/dependabot-core/issues/2445
    "body-max-line-length": [1, "always", 100],
  }
};`
	fr, err := os.Create(commitlintrc)
	if err != nil {
		return err
	}
	defer fr.Close()

	if _, err = fr.WriteString(commitlintFileContent); err != nil {
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
	err = sh.Run("git", "fetch", "--tags")
	if err != nil {
		return err
	}
	err = sh.RunV(Binary, args...)
	if err != nil {
		return err
	}
	return nil
}

func commitlint() error {
	// Check if npm is installed
	if err := sh.Run("npm", "version"); err != nil {
		return err
	}

	toolDir := filepath.Join(tools.Path, "commitlint")
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

	fp, err := os.Create(packageJSON)
	if err != nil {
		return err
	}
	defer fp.Close()

	if _, err = fp.WriteString(packageFileContent); err != nil {
		return err
	}

	Binary = binary

	fmt.Println("[commitlint] installing packages...")
	err = sh.Run(
		"npm",
		"--silent",
		"install",
		"--prefix",
		toolDir,
		"--no-save",
		"--no-audit",
		"--ignore-script",
	)
	if err != nil {
		return err
	}
	return nil
}
