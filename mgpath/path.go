package mgpath

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	ToolsMk            = "tools.mk"
	MakeGenGo          = "mgmake_gen.go"
	GenMakefilesTarget = "genMakefiles"
	MageDir            = ".mage"
	ToolsDir           = MageDir + "/tools"
)

// nolint: gochecknoglobals
var mgToolsPath = FromGitRoot(ToolsDir)

func FromWorkDir(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return filepath.Join(cwd, path)
}

func FromGitRoot(path string) string {
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

// Tools returns the base to where tools are downloaded and installed.
func Tools() string {
	if mgToolsPath == "" {
		panic("No tools path set")
	}
	return mgToolsPath
}
