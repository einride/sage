package tools

import (
	"os"
	"path/filepath"
)

// This should only be used to set a custom value.
// Targets should use toolsPath() instead which performs
// validation on whether a path is set
var ToolsPath = cwdPath("tools")

func cwdPath(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return filepath.Join(cwd, path)
}

func toolsPath() string {
	if ToolsPath == "" {
		panic("No tools path set")
	}
	return ToolsPath
}
