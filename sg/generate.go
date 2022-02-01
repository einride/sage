package sg

import (
	"context"
	"fmt"
	"go/doc"
	"go/parser"
	"go/token"
	"os"

	"go.einride.tech/sage/internal/codegen"
)

// GenerateMakefiles defines which Makefiles should be generated.
func GenerateMakefiles(mks ...Makefile) {
	ctx := WithLogger(context.Background(), NewLogger("sage"))
	Logger(ctx).Println("building binary and generating Makefiles...")
	if len(mks) == 0 {
		panic("no makefiles to generate, see https://github.com/einride/sage#readme for more info")
	}
	pkgs, err := parser.ParseDir(token.NewFileSet(), FromSageDir(), nil, parser.ParseComments)
	if err != nil {
		panic(fmt.Errorf("failed to parse directory: %v", err))
	}
	if len(pkgs) != 1 {
		panic(fmt.Errorf("parser returned unexpected number of packages: %d", len(pkgs)))
	}
	var pkg *doc.Package
	for _, p := range pkgs {
		pkg = doc.New(p, "./", 0)
	}
	// update .gitignore file
	const gitignoreContent = ".gitignore\ntools/\nbin/\nbuild/\n"
	if err := os.WriteFile(FromSageDir(".gitignore"), []byte(gitignoreContent), 0o600); err != nil {
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
	initFileContent, err := initFile.GoContent()
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
	compileCmd := Command(ctx, "go", "build", "-o", FromBinDir(sageFileBinary), ".")
	compileCmd.Dir = FromSageDir()
	if err := compileCmd.Run(); err != nil {
		panic(fmt.Errorf("error compiling sagefiles: %w", err))
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
		if err := os.WriteFile(v.Path, mk.RawContent(), 0o600); err != nil {
			panic(err)
		}
	}
}
