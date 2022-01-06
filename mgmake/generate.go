package mgmake

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mglog"
)

type templateTargets struct {
	MakeTarget string
	MageTarget string
	Args       []string
}

// GenerateMakefile is a mage target that ...
func GenerateMakefile(makefile string) error {
	const defaultTargets = "mage_targets"
	genDir := filepath.Dir(makefile)
	mgDefaultTargets := fmt.Sprintf("%s.mk", filepath.Join(genDir, strcase.ToKebab(defaultTargets)))
	if makefile == mgDefaultTargets {
		return fmt.Errorf("%s has the same name as the default %s makefile", makefile, mgDefaultTargets)
	}
	mglog.Logger("generate-makefile").Info("generating Makefile...")
	targets, err := listTargets()
	if err != nil {
		return err
	}

	// Create map which holds variables for each makefile being generated
	mgMakefiles := make(map[string]string)

	for _, target := range targets {
		var f *os.File
		args, _ := getTargetArguments(target)
		if strings.Contains(target, ":") {
			// Create unique makefile if target is namespaced
			name := "mage_" + strcase.ToSnake(strings.Split(target, ":")[0])
			filename := fmt.Sprintf("%s.mk", filepath.Join(genDir, strcase.ToKebab(name)))
			f, err = os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
			if err != nil {
				return err
			}
			if _, ok := mgMakefiles[name]; !ok {
				mgMakefiles[name] = filename
			}
		} else {
			f, err = os.OpenFile(mgDefaultTargets, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
			if err != nil {
				return err
			}
			if _, ok := mgMakefiles[defaultTargets]; !ok {
				mgMakefiles[defaultTargets] = mgDefaultTargets
			}
		}
		templateTarget := templateTargets{
			MakeTarget: toMakeTarget(target),
			MageTarget: toMageTarget(target, toMakeVars(args)),
			Args:       toMakeVars(args),
		}
		t, _ := template.New("dynamic").Parse(`
.PHONY: {{.MakeTarget}}
{{.MakeTarget}}:{{range .Args}}
ifndef {{.}}
{{"\t"}}$(error missing argument {{.}}="...")
endif{{end}}
{{"\t"}}@$(mage) {{.MageTarget}}
`)
		err = t.Execute(f, templateTarget)
		if err != nil {
			return err
		}
	}
	err = os.WriteFile(makefile, createMakefileVariablesFromMap(mgMakefiles), 0o600)
	if err != nil {
		return err
	}
	return nil
}

func createMakefileVariablesFromMap(m map[string]string) []byte {
	makefileVariables := make([]string, 0, len(m))
	for key, value := range m {
		makefileVariables = append(makefileVariables, fmt.Sprintf("%s := %s", key, value))
	}
	return []byte(strings.Join(makefileVariables, "\n"))
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
	out, err := invokeMage([]string{"-l"})
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
			if strings.Contains(parts[0], "generateMakefile") {
				continue
			}

			targets = append(targets, parts[0])
		}
	}

	return targets, nil
}

func getTargetArguments(name string) ([]string, error) {
	out, err := invokeMage([]string{"-h", name})
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

func invokeMage(args []string) (string, error) {
	binary, err := os.Executable()
	if err != nil {
		return "", err
	}
	out, err := sh.Output(binary, args...)
	if err != nil {
		return "", err
	}
	return out, nil
}
