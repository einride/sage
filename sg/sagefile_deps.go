package sg

import (
	"fmt"
	"go/build"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// SagefileDeps returns the source files the sagefile binary depends on.
// It uses go/build.ImportDir to resolve which .go files actually participate
// in the build for the current platform, rather than naively globbing all
// .go files (which would include test files and build-tag-excluded files).
//
// prefix is the relative path from the Makefile to the .sage directory
// (e.g. ".sage" for the root Makefile, "../../.sage" for a namespace
// Makefile in a subdirectory). Each returned path is prefixed with it so
// that Make can resolve the files from its working directory.
func SagefileDeps(prefix string) []string {
	if strings.Contains(prefix, " ") {
		fmt.Fprintf(os.Stderr, "sagefile --deps: prefix %q contains spaces, which Make cannot handle\n", prefix)
		return nil
	}
	sageDir := FromSageDir()
	pkg, err := build.ImportDir(sageDir, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sagefile --deps: %v\n", err)
		return nil
	}
	var deps []string
	for _, f := range pkg.GoFiles {
		deps = append(deps, filepath.Join(prefix, f))
	}
	deps = append(deps, filepath.Join(prefix, "go.mod"))
	deps = append(deps, filepath.Join(prefix, "go.sum"))
	// If go.mod has local replace directives (e.g. "replace foo => ../"),
	// changes to the replaced module's source should also trigger a rebuild.
	// This matters when developing sage itself, where .sage/go.mod points
	// back to the repo root via replace.
	for _, relTarget := range findLocalReplaces(filepath.Join(sageDir, "go.mod")) {
		absTarget := filepath.Join(sageDir, relTarget)
		_ = filepath.WalkDir(absTarget, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				name := d.Name()
				if name == "vendor" || name == "testdata" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
				rel, err := filepath.Rel(absTarget, path)
				if err != nil {
					return nil
				}
				deps = append(deps, filepath.Join(prefix, relTarget, rel))
			}
			return nil
		})
	}
	return deps
}

// findLocalReplaces parses a go.mod file and returns the target paths of
// replace directives that point to local directories (relative paths).
func findLocalReplaces(goModPath string) []string {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, line := range strings.Split(string(data), "\n") {
		// In go.mod, replace entries either start with "replace" (single-line)
		// or are tab-indented inside a replace() block.
		if !strings.HasPrefix(line, "replace") && !strings.HasPrefix(line, "\t") {
			continue
		}
		idx := strings.Index(line, "=>")
		if idx < 0 {
			continue
		}
		target := strings.TrimSpace(line[idx+2:])
		// Strip trailing version if present.
		if i := strings.IndexByte(target, ' '); i >= 0 {
			target = target[:i]
		}
		if strings.HasPrefix(target, "./") || strings.HasPrefix(target, "../") {
			dirs = append(dirs, target)
		}
	}
	return dirs
}
