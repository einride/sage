package mggo

import (
	"context"
	"go.einride.tech/mage-tools/mg"
	"os"
	"os/exec"
	"path/filepath"

	"go.einride.tech/mage-tools/mgtool"
)

func TestCommand(ctx context.Context) *exec.Cmd {
	coverFile := mg.FromToolsDir("go", "coverage", "go-test.txt")
	if err := os.MkdirAll(filepath.Dir(coverFile), 0o755); err != nil {
		panic(err)
	}
	return mgtool.Command(
		ctx,
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
