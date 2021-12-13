package prettier

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/einride/mage-tools/tools"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var Binary string

func FormatMarkdown() error {
	mg.Deps(prettier)
	args := []string{
		"--write",
		"**/*.md",
		"!.tools",
	}
	fmt.Println("[prettier] formatting Markdown files...", args)
	return sh.RunV(Binary, args...)
}

func prettier() error {
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

	const packageFileContent = `{
  "devDependencies": {
    "prettier": "^2.4.1",
    "@einride/prettier-config": "^1.2.0"
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

	fmt.Println("[prettier] installing packages...")
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
