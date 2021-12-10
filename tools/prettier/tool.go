package prettier

import (
	"fmt"

	"github.com/einride/mage-tools/tools"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

func FormatMarkdown() error {
	mg.Deps(tools.Prettier)
	args := []string{
		"--write",
		"**/*.md",
		"!.tools",
	}
	fmt.Println("[prettier] formatting Markdown files...", args)
	return sh.RunV(tools.PrettierPath, args...)
}
