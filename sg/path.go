package sg

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	sageDir        = ".sage"
	toolsDir       = "tools"
	binDir         = "bin"
	buildDir       = "build"
	sageFileBinary = "sagefile"
)

func FromWorkDir(pathElems ...string) string {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return filepath.Join(append([]string{cwd}, pathElems...)...)
}

func FromGitRoot(pathElems ...string) string {
	// We use exec.Command here because this command runs in a global,
	// which is set up before logging is configured, resulting in unwanted log prints.
	var output bytes.Buffer
	c := exec.Command("git", []string{"rev-parse", "--show-toplevel"}...)
	c.Env = os.Environ()
	c.Stderr = os.Stderr
	c.Stdout = &output
	c.Stdin = os.Stdin
	if err := c.Run(); err != nil {
		panic(err)
	}
	gitRoot := strings.TrimSpace(output.String())
	return filepath.Join(append([]string{gitRoot}, pathElems...)...)
}

// FromSageDir returns the path relative to where the sage files are kept.
func FromSageDir(pathElems ...string) string {
	return FromGitRoot(append([]string{sageDir}, pathElems...)...)
}

// FromToolsDir returns the path relative to where tools are downloaded and installed.
// Parent directories of the returned path will be automatically created.
func FromToolsDir(pathElems ...string) string {
	path := FromSageDir(append([]string{toolsDir}, pathElems...)...)
	ensureParentDir(path)
	return path
}

// FromBinDir returns the path relative to where tool binaries are installed.
// Parent directories of the returned path will be automatically created.
func FromBinDir(pathElems ...string) string {
	path := FromSageDir(append([]string{binDir}, pathElems...)...)
	ensureParentDir(path)
	return path
}

// FromBuildDir returns the path relative to where generated build files are installed.
// Parent directories of the returned path will be automatically created.
func FromBuildDir(pathElems ...string) string {
	path := FromSageDir(append([]string{buildDir}, pathElems...)...)
	ensureParentDir(path)
	return path
}

func ensureParentDir(path string) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		panic(err)
	}
}
