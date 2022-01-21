package mggo

import (
	"os"
	"os/exec"
	"path/filepath"

	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

func Command(args ...string) *exec.Cmd {
	return mgtool.Command("go", args...)
}

func TestCommand() *exec.Cmd {
	coverFile := mgpath.FromTools("go", "coverage", "go-test.txt")
	if err := os.MkdirAll(filepath.Dir(coverFile), 0o755); err != nil {
		panic(err)
	}
	return Command(
		"test",
		"-race",
		"-coverprofile",
		coverFile,
		"-covermode",
		"atomic",
		"./...",
	)
}
