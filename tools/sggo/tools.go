package sggo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"go.einride.tech/sage/sg"
)

func TestCommand(ctx context.Context) *exec.Cmd {
	coverFile := sg.FromBuildDir("go", "coverage", "go-test.txt")
	if err := os.MkdirAll(filepath.Dir(coverFile), 0o755); err != nil {
		panic(err)
	}
	return sg.Command(
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
