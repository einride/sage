package mg

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	MageDir        = ".mage"
	ToolsDir       = "tools"
	BinDir         = "bin"
	MagefileBinary = "bin/magefile"
	MakeGenGo      = "mgmake_gen.go"
)

func FromWorkDir(pathElems ...string) string {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return filepath.Join(append([]string{cwd}, pathElems...)...)
}

func FromGitRoot(pathElems ...string) string {
	// We use exec.command here because this command runs in a global,
	// which is set up before mage has configured logging resulting in unwanted log prints
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

// FromMageDir returns the path relative to the mage root.
func FromMageDir(pathElems ...string) string {
	return FromGitRoot(append([]string{MageDir}, pathElems...)...)
}

// FromToolsDir returns the path relative to where tools are downloaded and installed.
func FromToolsDir(pathElems ...string) string {
	return FromMageDir(append([]string{ToolsDir}, pathElems...)...)
}

// FromBinDir returns the path relative to where tool binaries are installed.
func FromBinDir(pathElems ...string) string {
	return FromToolsDir(append([]string{BinDir}, pathElems...)...)
}

func FindFilesWithExtension(path, ext string) []string {
	var files []string
	if err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if filepath.Ext(path) == ext {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		panic(err)
	}
	return files
}
