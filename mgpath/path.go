package mgpath

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
)

const (
	MageDir        = ".mage"
	ToolsDir       = "tools"
	BinDir         = "bin"
	MagefileBinary = "bin/magefile"
)

func FromWorkDir(pathElems ...string) string {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return filepath.Join(append([]string{cwd}, pathElems...)...)
}

func FromGitRoot(pathElems ...string) string {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	repo, err := git.PlainOpenWithOptions(cwd, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		panic(err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		panic(err)
	}
	return filepath.Join(append([]string{wt.Filesystem.Root()}, pathElems...)...)
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
