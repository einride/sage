package tools

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/einride/mage-tools/file"
	"github.com/magefile/mage/sh"
)

func GoTool(name string, goPkg string) error {
	toolDir, err := filepath.Abs(Path)
	if err != nil {
		return err
	}
	binDir := filepath.Join(toolDir, name, "bin")
	binary := filepath.Join(binDir, name)

	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH")))

	// Check if binary already exist
	if file.Exists(binary) == nil {
		return nil
	}

	os.Setenv("GOBIN", binDir)
	fmt.Printf("Building %s...\n", goPkg)
	if err := sh.Run("go", "install", goPkg); err != nil {
		fmt.Println("Failed")
		return err
	}
	fmt.Println("OK")
	return nil
}
