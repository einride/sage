package mgpath

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	MakeGenGo          = "mgmake_gen.go"
	GenMakefilesTarget = "genMakefiles"
	MageDir            = ".mage"
	ToolsDir           = "tools"
	MagefileBinary     = "mgmake/magefile"
)

// nolint: gochecknoglobals
var mgToolsPath = FromGitRoot(filepath.Join(MageDir, ToolsDir))

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

func ChangeWorkDir(path string) func() {
	cwd := FromWorkDir(".")
	if err := os.Chdir(path); err != nil {
		panic(err)
	}
	return func() {
		if err := os.Chdir(cwd); err != nil {
			panic(err)
		}
	}
}

func FindFilesWithExtension(path, ext string) ([]string, error) {
	var files []string
	if err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if filepath.Ext(path) == ext {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return files, nil
}
