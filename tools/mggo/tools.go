package mggo

import (
	"os"
	"os/exec"
	"path/filepath"

	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

func GoTest() *exec.Cmd {
	coverFile := filepath.Join(mgpath.Tools(), "go", "coverage", "go-test.txt")
	if err := os.MkdirAll(filepath.Dir(coverFile), 0o755); err != nil {
		panic(err)
	}
	return mgtool.Command("go", "test",
		"-race",
		"-coverprofile",
		coverFile,
		"-covermode",
		"atomic",
		"./...",
	)
}

func GoModTidy() *exec.Cmd {
	return mgtool.Command("go", "mod", "tidy", "-v")
}
