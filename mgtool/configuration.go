package mgtool

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	AMD64 = "amd64"
	X8664 = "x86_64"
)

// Path This should only be used to set a custom value.
// Targets should use path() instead which performs
// validation on whether a path is set.
var mgToolPath = GetGitRootPath(".mage/tools")

func GetCWDPath(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return filepath.Join(cwd, path)
}

func GetGitRootPath(path string) string {
	// We use exec.command here because this command runs in a global,
	// which is set up before mage has configured logging resulting in unwanted log prints
	output := &bytes.Buffer{}
	c := exec.Command("git", []string{"rev-parse", "--show-toplevel"}...)
	c.Env = os.Environ()
	c.Stderr = os.Stderr
	c.Stdout = output
	c.Stdin = os.Stdin

	if err := c.Run(); err != nil {
		panic(err)
	}
	return filepath.Join(strings.TrimSpace(output.String()), path)
}

func GetPath() string {
	if mgToolPath == "" {
		panic("No tools path set")
	}
	return mgToolPath
}

func SetPath(p string) {
	mgToolPath = p
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
