package mgmake

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"github.com/iancoleman/strcase"
	"go.einride.tech/mage-tools/internal/codegen"
	"go.einride.tech/mage-tools/mg"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

const defaultNamespace = "default"

// nolint: gochecknoglobals
var makefiles = make(map[string]makefile)

type Makefile struct {
	Namespace     interface{}
	Path          string
	DefaultTarget interface{}
}

type makefile struct {
	Path          string
	DefaultTarget string
}

type templateTarget struct {
	MakeTarget string
	MageTarget string
	Args       []string
}

// GenerateMakefiles define which makefiles should be created by go.einride.tech/cmd/build.
func GenerateMakefiles(mks ...Makefile) {
	for _, i := range mks {
		if i.Path == "" {
			panic("Path needs to be defined")
		}
		namespace := defaultNamespace
		if i.Namespace != nil {
			namespace = reflect.TypeOf(i.Namespace).Name()
		}
		var defaultTarget string
		if i.DefaultTarget != nil {
			defaultTarget = runtime.FuncForPC(reflect.ValueOf(i.DefaultTarget).Pointer()).Name()
			defaultTarget = strings.TrimPrefix(defaultTarget, "main.")
			defaultTarget = strings.TrimPrefix(defaultTarget, namespace+".")
			defaultTarget = strings.Split(defaultTarget, "-")[0]
			for _, r := range defaultTarget {
				if !unicode.IsLetter(r) {
					panic(fmt.Sprintf("Invalid default target %s", defaultTarget))
				}
			}
		}
		makefiles[toMakeTarget(namespace)] = makefile{Path: i.Path, DefaultTarget: defaultTarget}
	}
}

// compile uses the go tool to compile the files into an executable at path.
func compile(magePath, compileTo string, gofiles []string) error {
	// strip off the path since we're setting the path in the build command
	for i := range gofiles {
		gofiles[i] = filepath.Base(gofiles[i])
	}
	// nolint: gosec
	c := exec.Command("go", append([]string{"build", "-o", compileTo}, gofiles...)...)
	c.Env = os.Environ()
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	c.Dir = magePath
	if err := c.Run(); err != nil {
		return fmt.Errorf("error compiling magefiles: %w", err)
	}
	return nil
}

// GenMakefiles should only be used by go.einride.tech/cmd/build.
func GenMakefiles(ctx context.Context) {
	if len(makefiles) == 0 {
		panic("no makefiles to generate, see https://github.com/einride/mage-tools#readme for more info")
	}
	mageDir := mgpath.FromGitRoot(mgpath.MageDir)
	var mageFiles []string
	if err := filepath.WalkDir(mageDir, func(path string, d fs.DirEntry, err error) error {
		if filepath.Ext(path) == ".go" {
			if filepath.Base(path) != "mgmake_gen.go" {
				mageFiles = append(mageFiles, filepath.Base(path))
			}
		}
		return nil
	}); err != nil {
		panic(err)
	}
	targets, err := mg.Package(mageDir)
	if err != nil {
		panic(err)
	}
	sort.Sort(targets.Funcs)
	// compile binary
	executable := mgpath.FromToolsDir(mgpath.MagefileBinary)
	mainFilename := mgpath.FromMageDir("generating_magefile.go")
	mainFile := codegen.NewFile(codegen.FileConfig{
		Filename:    mainFilename,
		Package:     targets.DocPkg.Name,
		GeneratedBy: "go.einride.tech/mage-tools",
	})
	if err := generateMainFile2(targets.DocPkg, mainFile); err != nil {
		panic(err)
	}
	mainFileContent, err := mainFile.Content()
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(mainFilename, mainFileContent, 0o600); err != nil {
		panic(err)
	}
	//defer os.Remove(mainFilename)
	if err := compile(
		mgpath.FromMageDir(),
		executable,
		append(mageFiles, mainFilename),
	); err != nil {
		panic(err)
	}
	buffers, err := generateMakeTargets(targets.Funcs)
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
				if strings.Contains(defaultBuf.String(), fmt.Sprintf(".PHONY: %s\n", ns)) {
					panic(fmt.Errorf("can't create target for makefile, %s already exist", ns))
				}
				mkPath, err := filepath.Rel(mgpath.FromGitRoot("."), filepath.Dir(mk.Path))
				if err != nil {
					panic(err)
				}
				mkTarget := []byte(fmt.Sprintf(`.PHONY: %s
%s:
	make -C %s

`, ns, ns, mkPath))
				buffers[defaultNamespace] = bytes.NewBuffer(append(defaultBuf.Bytes(), mkTarget...))
			}
		}
	}
	// Write non-root makefiles
	for _, ns := range namespaces {
		if buf, ok := buffers[ns]; ok {
			mk := makefiles[ns]
			if err := createMakefile(ctx, mk.Path, mk.DefaultTarget, buf.Bytes()); err != nil {
				panic(err)
			}
		}
	}
}

func createMakefile(ctx context.Context, makefilePath, target string, data []byte) error {
	includePath, err := filepath.Rel(filepath.Dir(makefilePath), mgpath.FromGitRoot(mgpath.MageDir))
	if err != nil {
		return err
	}
	if target != "" {
		target = fmt.Sprintf("\n\n.DEFAULT_GOAL := %s", toMakeTarget(target))
	}
	cmd := mgtool.Command(ctx, "go", "list", "-m")
	var b bytes.Buffer
	cmd.Stdout = &b
	if err := cmd.Run(); err != nil {
		return err
	}
	mageDependencies := fmt.Sprintf("%s/go.mod %s/*.go", includePath, includePath)
	genCommand := fmt.Sprintf("cd %s && go run go.einride.tech/mage-tools/cmd/build", includePath)
	if strings.TrimSpace(b.String()) == "go.einride.tech/mage-tools" {
		mageDependencies = fmt.Sprintf("%s/go.mod $(shell find %s/.. -type f -name '*.go')", includePath, includePath)
		genCommand = fmt.Sprintf("cd %s && go run ../cmd/build", includePath)
	}
	codegen := []byte(fmt.Sprintf(
		`# Code generated by go.einride.tech/mage-tools. DO NOT EDIT.
# To learn more, see %s and https://github.com/einride/mage-tools.%s

magefile := %s

$(magefile): %s
	@%s

.PHONY: clean-mage-tools
clean-mage-tools:
	@git clean -fdx %s

`,
		// TODO: Refer to the source file that the default target or namespace comes from.
		filepath.Join(includePath, "magefile.go"),
		target,
		filepath.Join(includePath, mgpath.ToolsDir, mgpath.MagefileBinary),
		mageDependencies,
		genCommand,
		filepath.Join(includePath, mgpath.ToolsDir),
	))
	// Removes trailing empty line
	data = data[:len(data)-1]
	err = os.WriteFile(makefilePath, append(codegen, data...), 0o600)
	if err != nil {
		return err
	}
	return nil
}

func generateMakeTargets(targets mg.Functions) (map[string]*bytes.Buffer, error) {
	buffers := make(map[string]*bytes.Buffer)
	for _, target := range targets {
		var b *bytes.Buffer
		var ns string
		if target.Receiver != "" {
			ns = toMakeTarget(target.Receiver)
		} else {
			ns = defaultNamespace
		}
		if _, ok := buffers[ns]; ok {
			b = buffers[ns]
		} else {
			b = bytes.NewBuffer(make([]byte, 0))
		}
		templateTarget := templateTarget{
			MakeTarget: toMakeTarget(target.Name),
			MageTarget: toMageTarget(target.Name, target.Receiver, toMakeVars(target.Args)),
			Args:       toMakeVars(target.Args),
		}
		t, _ := template.New("dynamic").Parse(`.PHONY: {{.MakeTarget}}
{{.MakeTarget}}: $(magefile){{range .Args}}
ifndef {{.}}
{{"\t"}}$(error missing argument {{.}}="...")
endif{{end}}
{{"\t"}}@$(magefile) {{.MageTarget}}

`)
		err := t.Execute(b, templateTarget)
		if err != nil {
			return nil, err
		}
		buffers[ns] = b
	}
	return buffers, nil
}

// toMakeVars converts input to make vars.
func toMakeVars(args []mg.Arg) []string {
	makeVars := make([]string, 0, len(args))
	for _, arg := range args {
		name := strcase.ToSnake(arg.Name)
		makeVars = append(makeVars, name)
	}
	return makeVars
}

// toMakeTarget converts input to make target format.
func toMakeTarget(str string) string {
	output := strcase.ToKebab(str)
	return strings.ToLower(output)
}

// toMageTarget converts input to mageTarget with makeVars as arguments.
func toMageTarget(target, receiver string, args []string) string {
	if receiver != "" {
		target = fmt.Sprintf("%s:%s", receiver, target)
	}
	for _, arg := range args {
		arg = fmt.Sprintf("$(%s)", arg)
		target += fmt.Sprintf(" %s", arg)
	}
	return target
}
