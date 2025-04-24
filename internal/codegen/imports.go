package codegen

import (
	"bytes"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// imports manages imports for a generated file.
type imports struct {
	packageNames     map[string]string
	usedPackageNames map[string]bool
}

// newImports creates a new imports set.
func newImports() *imports {
	return &imports{
		packageNames:     map[string]string{},
		usedPackageNames: map[string]bool{},
	}
}

// Import includes the provided identifier in the imports returns a package identifier.
func (i *imports) Import(path string) string {
	if packageName, ok := i.packageNames[path]; ok {
		return packageName
	}
	packageName := importPathToAssumedName(path)
	for n, orig := 1, packageName; i.usedPackageNames[packageName]; n++ {
		packageName = orig + strconv.Itoa(n)
	}
	i.packageNames[path] = packageName
	i.usedPackageNames[packageName] = true
	return packageName
}

// Bytes returns the generated code bytes for the imports.
func (i *imports) Bytes() []byte {
	if len(i.packageNames) == 0 {
		return nil
	}
	type pkg struct {
		importPath, packageName string
	}
	stdPkgs := make([]pkg, 0, len(i.packageNames))
	nonStdPkgs := make([]pkg, 0, len(i.packageNames))
	for importPath, packageName := range i.packageNames {
		if strings.Contains(importPath, ".") && packageName == path.Base(importPath) {
			nonStdPkgs = append(nonStdPkgs, pkg{importPath: importPath, packageName: packageName})
		} else {
			stdPkgs = append(stdPkgs, pkg{importPath: importPath, packageName: packageName})
		}
	}
	sort.Slice(stdPkgs, func(i, j int) bool {
		return stdPkgs[i].importPath < stdPkgs[j].importPath
	})
	sort.Slice(nonStdPkgs, func(i, j int) bool {
		return nonStdPkgs[i].importPath < nonStdPkgs[j].importPath
	})
	var b bytes.Buffer
	_ = b.WriteByte('\n')
	_, _ = b.WriteString("import (")
	for _, stdPkg := range stdPkgs {
		_, _ = b.WriteString(strconv.Quote(stdPkg.importPath))
		_ = b.WriteByte('\n')
	}
	if len(nonStdPkgs) > 0 {
		_ = b.WriteByte('\n')
	}
	for _, nonStdPkg := range nonStdPkgs {
		if nonStdPkg.packageName == path.Base(nonStdPkg.importPath) {
			_, _ = b.WriteString(strconv.Quote(nonStdPkg.importPath))
			_ = b.WriteByte('\n')
		} else {
			_, _ = b.WriteString(nonStdPkg.packageName)
			_ = b.WriteByte(' ')
			_, _ = b.WriteString(strconv.Quote(nonStdPkg.importPath))
			_ = b.WriteByte('\n')
		}
	}
	_ = b.WriteByte(')')
	return b.Bytes()
}

// importPathToAssumedName is copy-pasted from golang.org/x/tools.
func importPathToAssumedName(importPath string) string {
	base := path.Base(importPath)
	if strings.HasPrefix(base, "v") {
		if _, err := strconv.Atoi(base[1:]); err == nil {
			dir := path.Dir(importPath)
			if dir != "." {
				base = path.Base(dir)
			}
		}
	}
	base = strings.TrimPrefix(base, "go-")
	if i := strings.IndexFunc(base, func(r rune) bool {
		//nolint:staticcheck // De Morgan's law intentionally not applied
		return !('a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || '0' <= r && r <= '9' || r == '_' ||
			r >= utf8.RuneSelf && (unicode.IsLetter(r) || unicode.IsDigit(r)))
	}); i >= 0 {
		base = base[:i]
	}
	return base
}
