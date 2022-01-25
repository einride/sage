package mg

import (
	"context"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"unicode"

	"go.einride.tech/mage-tools/internal/codegen"
)

const defaultNamespace = "default"

// nolint: gochecknoglobals
var makefiles = make(map[string]Makefile)

type Makefile struct {
	Namespace     interface{}
	Path          string
	DefaultTarget interface{}
}

func (m Makefile) namespaceName() string {
	if m.Namespace == nil {
		return defaultNamespace
	}
	return reflect.TypeOf(m.Namespace).Name()
}

func (m Makefile) defaultTargetName() string {
	if m.DefaultTarget == nil {
		return ""
	}
	result := runtime.FuncForPC(reflect.ValueOf(m.DefaultTarget).Pointer()).Name()
	result = strings.TrimPrefix(result, "main.")
	result = strings.TrimPrefix(result, m.namespaceName()+".")
	result = strings.Split(result, "-")[0]
	for _, r := range result {
		if !unicode.IsLetter(r) {
			panic(fmt.Sprintf("Invalid default target %s", result))
		}
	}
	return result
}

// GenerateMakefiles define which makefiles should be created by go.einride.tech/cmd/build.
func GenerateMakefiles(mks ...Makefile) {
	for _, i := range mks {
		if i.Path == "" {
			panic("Path needs to be defined")
		}
		makefiles[i.namespaceName()] = i
	}
}

// compile uses the go tool to compile the files into an executable at path.
func compile(ctx context.Context, files []string) error {
	for file := range files {
		files[file] = filepath.Base(files[file])
	}
	cmd := Command(ctx, "go", "build", "-o", FromToolsDir(MagefileBinary))
	cmd.Args = append(cmd.Args, files...)
	cmd.Dir = FromMageDir()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error compiling magefiles: %w", err)
	}
	return nil
}

// GenMakefiles should only be used by go.einride.tech/cmd/build.
func GenMakefiles(ctx context.Context) {
	if len(makefiles) == 0 {
		panic("no makefiles to generate, see https://github.com/einride/mage-tools#readme for more info")
	}
	mageDir := FromGitRoot(MageDir)
	var mageFiles []string
	if err := filepath.WalkDir(mageDir, func(path string, d fs.DirEntry, err error) error {
		if filepath.Ext(path) == ".go" {
			if filepath.Base(path) != MakeGenGo {
				mageFiles = append(mageFiles, path)
			}
		}
		return nil
	}); err != nil {
		panic(err)
	}
	pkg, err := parsePackage(mageDir)
	if err != nil {
		panic(err)
	}
	// compile binary
	mainFilename := FromMageDir("generating_magefile.go")
	mainFile := codegen.NewFile(codegen.FileConfig{
		Filename:    mainFilename,
		Package:     pkg.Name,
		GeneratedBy: "go.einride.tech/mage-tools",
	})
	if err := generateMainFile(mainFile, pkg); err != nil {
		panic(err)
	}
	mainFileContent, err := mainFile.Content()
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(mainFilename, mainFileContent, 0o600); err != nil {
		panic(err)
	}
	defer func() {
		_ = os.Remove(mainFilename)
	}()
	if err := compile(
		ctx,
		append(mageFiles, mainFilename),
	); err != nil {
		panic(err)
	}
	buffers, err := generateMakeTargets(pkg)
	if err != nil {
		panic(err)
	}

	namespaces := make([]string, 0, len(makefiles))
	for k := range makefiles {
		namespaces = append(namespaces, k)
	}
	sort.Strings(namespaces)

	// Add target for non-root makefile to default makefile
	for _, ns := range namespaces {
		if ns != defaultNamespace {
			mk := makefiles[ns]
			if defaultBuf, ok := buffers[defaultNamespace]; ok {
				if strings.Contains(string(defaultBuf), fmt.Sprintf(".PHONY: %s\n", ns)) {
					panic(fmt.Errorf("can't create target for makefile, %s already exist", ns))
				}
				mkPath, err := filepath.Rel(FromGitRoot("."), filepath.Dir(mk.Path))
				if err != nil {
					panic(err)
				}
				mkTarget := []byte(fmt.Sprintf(`
.PHONY: %s
%s:
	make -C %s

`, toMakeTarget(ns), toMakeTarget(ns), mkPath))
				buffers[defaultNamespace] = append(defaultBuf, mkTarget...)
			}
		}
	}
	// Write non-root makefiles
	for _, ns := range namespaces {
		if buf, ok := buffers[ns]; ok {
			mk := makefiles[ns]
			if err := os.WriteFile(mk.Path, buf[:len(buf)-1], 0o600); err != nil {
				panic(err)
			}
		}
	}
}

func generateMakeTargets(targets *doc.Package) (map[string][]byte, error) {
	buffers := make(map[string][]byte)
	for k, v := range makefiles {
		mk := codegen.NewMakefile(codegen.FileConfig{
			GeneratedBy: "go.einride.tech/mage-tools",
		})
		// nolint: gosec
		if err := generateMakefile(mk, targets, &v, k); err != nil {
			panic(err)
		}
		buffers[k] = mk.Bytes()
	}
	return buffers, nil
}

func parsePackage(path string) (*doc.Package, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, path, func(info fs.FileInfo) bool {
		return info.Name() != MakeGenGo
	}, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse directory: %v", err)
	}
	var pkg *ast.Package
	switch len(pkgs) {
	case 1:
		for _, p := range pkgs {
			pkg = p
		}
	case 0:
		return nil, fmt.Errorf("no importable packages found in %s", path)
	default:
		var names []string
		for name := range pkgs {
			names = append(names, name)
		}
		return nil, fmt.Errorf("multiple packages found in %s: %v", path, strings.Join(names, ", "))
	}
	return doc.New(pkg, "./", 0), nil
}
