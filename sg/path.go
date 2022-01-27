package sg

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	SageDir        = ".sage"
	ToolsDir       = "tools"
	BinDir         = "bin"
	SageFileBinary = "bin/sagefile"
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
	return FromGitRoot(append([]string{SageDir}, pathElems...)...)
}

// FromToolsDir returns the path relative to where tools are downloaded and installed.
func FromToolsDir(pathElems ...string) string {
	return FromSageDir(append([]string{ToolsDir}, pathElems...)...)
}

// FromBinDir returns the path relative to where tool binaries are installed.
func FromBinDir(pathElems ...string) string {
	return FromSageDir(append([]string{BinDir}, pathElems...)...)
}
