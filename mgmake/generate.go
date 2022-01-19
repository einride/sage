package mgmake

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"github.com/iancoleman/strcase"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
)

const (
	GenMakefilesTarget = "genMakefiles"
	defaultNamespace   = "default"
)

// nolint: gochecknoglobals
var (
	executable string
	makefiles  = make(map[string]makefile)
)

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

func GenMakefiles(exec string) error {
	if len(makefiles) == 0 {
		return fmt.Errorf("no makefiles to generate, see https://github.com/einride/mage-tools#readme for more info")
	}
	executable = exec
	targets, err := listTargets()
	if err != nil {
		return err
	}
	buffers, err := generateMakeTargets(targets)
	if err != nil {
		return err
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
					return fmt.Errorf("can't create target for makefile, %s already exist", ns)
				}
				mkPath, err := filepath.Rel(mgpath.FromGitRoot("."), filepath.Dir(mk.Path))
				if err != nil {
					return err
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
			if err := createMakefile(mk.Path, mk.DefaultTarget, buf.Bytes()); err != nil {
				return err
			}
		}
	}
	return nil
}

func createMakefile(makefilePath, target string, data []byte) error {
	includePath, err := filepath.Rel(filepath.Dir(makefilePath), mgpath.FromWorkDir("."))
	if err != nil {
		return err
	}
	if target != "" {
		target = fmt.Sprintf("\n\n.DEFAULT_GOAL := %s", toMakeTarget(target))
	}
	cmd := mgtool.Command("go", "list", "-m")
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

func generateMakeTargets(targets []string) (map[string]*bytes.Buffer, error) {
	buffers := make(map[string]*bytes.Buffer)
	for _, target := range targets {
		var b *bytes.Buffer
		var ns string
		if strings.Contains(target, ":") {
			ns = toMakeTarget(strings.Split(target, ":")[0])
		} else {
			ns = defaultNamespace
		}
		if _, ok := buffers[ns]; ok {
			b = buffers[ns]
		} else {
			b = bytes.NewBuffer(make([]byte, 0))
		}
		args, _ := getTargetArguments(target)
		templateTarget := templateTarget{
			MakeTarget: toMakeTarget(target),
			MageTarget: toMageTarget(target, toMakeVars(args)),
			Args:       toMakeVars(args),
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
func toMakeVars(args []string) []string {
	makeVars := make([]string, 0, len(args))
	for _, arg := range args {
		arg = strcase.ToSnake(arg)
		arg = strings.ReplaceAll(arg, "<", "")
		arg = strings.ReplaceAll(arg, ">", "")
		makeVars = append(makeVars, arg)
	}
	return makeVars
}

// toMakeTarget converts input to make target format.
func toMakeTarget(str string) string {
	const delimiter = ":"
	output := strcase.ToKebab(str)
	// Remove namespace if defined. We only use namespace for generating unique includes
	if strings.Contains(output, delimiter) {
		output = strings.Join(strings.Split(output, delimiter)[1:], delimiter)
	}
	return strings.ToLower(output)
}

// toMageTarget converts input to mageTarget with makeVars as arguments.
func toMageTarget(target string, args []string) string {
	for _, arg := range args {
		arg = fmt.Sprintf("$(%s)", arg)
		target += fmt.Sprintf(" %s", arg)
	}
	return target
}

func listTargets() ([]string, error) {
	out, err := invokeMage("-l")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) > 0 {
		// Remove "Targets: " lines
		if strings.HasPrefix(lines[0], "Targets:") {
			lines = lines[1:]
		}
		// If a default is set remove the last line informing the
		// default target
		if strings.Contains(lines[len(lines)-1], "* default") {
			lines = lines[:len(lines)-1]
		}
	}

	var targets []string
	for _, l := range lines {
		parts := strings.Fields(l)
		if len(parts) > 0 {
			// Remove spaces and default mark (*)
			parts[0] = strings.TrimRight(strings.TrimSpace(parts[0]), "*")

			// Remove this mage target from the output
			if strings.Contains(parts[0], GenMakefilesTarget) {
				continue
			}
			targets = append(targets, parts[0])
		}
	}

	return targets, nil
}

func getTargetArguments(name string) ([]string, error) {
	out, err := invokeMage("-h", name)
	if err != nil {
		return nil, err
	}

	// Removes Usage: mage COMMAND and adds remaining arguments to a list.
	args := strings.Fields(strings.ReplaceAll(out, "\n", ""))[3:]
	if len(args) == 0 {
		return nil, nil
	}

	return args, nil
}

func invokeMage(args ...string) (string, error) {
	cmd := mgtool.Command(executable, args...)
	var b bytes.Buffer
	cmd.Stdout = &b
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return b.String(), nil
}
