package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Path This should only be used to set a custom value.
// Targets should use path() instead which performs
// validation on whether a path is set.
var Path = cwdPath("tools")

func cwdPath(path string) string {
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

func IsSupportedVersion(versions []string, version string, name string) error {
	for _, a := range versions {
		if a == version {
			return nil
		}
	}
	return fmt.Errorf(
		"the following %s versions are supported: %s",
		name,
		strings.Join(versions, ", "),
	)
}