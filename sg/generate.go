package sg

import (
	"context"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"runtime"
	"strings"
	"unicode"

	"github.com/go-logr/logr"
	"go.einride.tech/sage/internal/codegen"
)

type Makefile struct {
	Namespace     interface{}
	Path          string
	DefaultTarget interface{}
}

func (m Makefile) namespaceName() string {
	if m.Namespace == nil {
		return ""
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

// GenerateMakefiles defines which Makefiles should be generated.
func GenerateMakefiles(mks ...Makefile) {
	ctx := logr.NewContext(context.Background(), NewLogger("sage"))
	logr.FromContextOrDiscard(ctx).Info("building binary and generating Makefiles...")
	genMakefiles(ctx, mks...)
}

// compile uses the go tool to compile the files into an executable at path.
func compile(ctx context.Context) error {
	cmd := Command(ctx, "go", "build", "-o", FromBinDir(SageFileBinary), ".")
	cmd.Dir = FromSageDir()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error compiling sagefiles: %w", err)
	}
	return nil
}

func genMakefiles(ctx context.Context, mks ...Makefile) {
	if len(mks) == 0 {
		panic("no makefiles to generate, see https://github.com/einride/sage#readme for more info")
	}
	pkg, err := parsePackage(FromGitRoot(SageDir))
	if err != nil {
		panic(err)
	}
	// generate init file before compiling
	initFilename := FromSageDir("generating_sagefile.go")
	initFile := codegen.NewFile(codegen.FileConfig{
		Filename:    initFilename,
		Package:     pkg.Name,
		GeneratedBy: "go.einride.tech/sage",
	})
	if err := generateInitFile(initFile, pkg); err != nil {
		panic(err)
	}
	initFileContent, err := initFile.Content()
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(initFilename, initFileContent, 0o600); err != nil {
		panic(err)
	}
	defer func() {
		_ = os.Remove(initFilename)
	}()
	// Compile binary
	if err := compile(ctx); err != nil {
		panic(err)
	}
	// Generate makefiles
	for _, v := range mks {
		if v.Path == "" {
			panic("Path needs to be defined")
		}
		mk := codegen.NewMakefile(codegen.FileConfig{
			GeneratedBy: "go.einride.tech/sage",
		})

		if err := generateMakefile(mk, pkg, v, mks...); err != nil {
			panic(err)
		}
		// Remove trailing whitespace with len
		if err := os.WriteFile(v.Path, mk.Bytes()[:len(mk.Bytes())-1], 0o600); err != nil {
			panic(err)
		}
	}
}

func parsePackage(path string) (*doc.Package, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, path, nil, parser.ParseComments)
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
