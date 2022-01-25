package mggo

import (
	"context"
	"go.einride.tech/mage-tools/mg"
	"os"
	"os/exec"
	"path/filepath"
)

func TestCommand(ctx context.Context) *exec.Cmd {
	coverFile := mg.FromToolsDir("go", "coverage", "go-test.txt")
	if err := os.MkdirAll(filepath.Dir(coverFile), 0o755); err != nil {
		panic(err)
	}
	return mg.Command(
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
