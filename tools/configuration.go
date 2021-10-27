package tools

import (
	"os"
	"path/filepath"
)

// Path This should only be used to set a custom value.
// Targets should use path() instead which performs
// validation on whether a path is set
var Path = CwdPath("tools")

func CwdPath(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return filepath.Join(cwd, path)
}

func path() string {
	if Path == "" {
		panic("No tools path set")
	}
	return Path
}
