package mggo

import (
	"os"
	"path/filepath"

	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgtool"
)

func GoTest() error {
	const toolName = "go"
	mglog.Logger("go-test").Info("running Go unit tests..")
	coverFile := filepath.Join(mgtool.GetPath(), toolName, "coverage", "go-test.txt")
	if err := os.MkdirAll(filepath.Dir(coverFile), 0o755); err != nil {
		return err
	}
	return sh.RunV(
		"go",
		"test",
		"-race",
		"-coverprofile",
		coverFile,
		"-covermode",
		"atomic",
		"./...",
	)
}

func GoModTidy() error {
	mglog.Logger("go-mod-tidy").Info("tidying Go module files...")
	return sh.RunV("go", "mod", "tidy", "-v")
}
