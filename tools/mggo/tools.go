package mggo

import (
	"os"
	"os/exec"
	"path/filepath"

	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

func TestCommand() *exec.Cmd {
	coverFile := mgpath.FromTools("go", "coverage", "go-test.txt")
	if err := os.MkdirAll(filepath.Dir(coverFile), 0o755); err != nil {
		panic(err)
	}
	return mgtool.Command(
		"go",
		"test",
		"-shuffle",
		"on",
		"-race",
		"-coverprofile",
		coverFile,
		"-covermode",
		"atomic",
		"./...",
	)
}
