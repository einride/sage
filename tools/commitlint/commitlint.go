package commitlint

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/einride/mage-tools/tools"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

func Commitlint(branch string) error {
	mg.Deps(tools.Commitlint)
	path, err := filepath.Abs(tools.Path)
	if err != nil {
		return err
	}

	commitlintrc := filepath.Join(path, "commitlint", ".commitlintrc.js")

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
	err = sh.RunV("commitlint", args...)
	if err != nil {
		return err
	}
	return nil
}
