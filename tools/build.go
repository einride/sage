package tools

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/magefile/mage/sh"
)

func GoTool(name string, goPkg string) error {
	toolDir, err := filepath.Abs(GetPath())
	if err != nil {
		return err
	}
	binDir := filepath.Join(toolDir, name, "bin")
	binary := filepath.Join(binDir, name)

	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH")))

	// Check if binary already exist
	if _, err := os.Stat(binary); err == nil {
		return nil
	}

	os.Setenv("GOBIN", binDir)
	fmt.Printf("Building %s...\n", goPkg)
	return sh.RunV("go", "install", goPkg)
}
