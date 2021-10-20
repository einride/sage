package tools

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/magefile/mage/sh"
)

func GoTool(name string, goPkg string) {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	toolDir := filepath.Join(cwd, "tools", name)
	binDir := filepath.Join(toolDir, "bin")
	binary := filepath.Join(binDir, name)

	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH")))

	os.Setenv("GOBIN", binDir)
	fmt.Printf("Building %s...", goPkg)
	if err := sh.Run("go", "install", goPkg); err != nil {
		fmt.Println("Failed")
		panic(err)
	}
	fmt.Println("OK")
}
