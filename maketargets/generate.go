package maketargets

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
	"github.com/magefile/mage/sh"
)

type templateTargets struct {
	MakeTarget string
	MageTarget string
	Args       []string
}

func GenerateMakefile(makefile string) error {
	fmt.Println("[generate-makefile] generating makefile...")
	targets, err := listTargets()
	if err != nil {
		return err
	}
	// Write makefile to disk
	f, err := os.Create(makefile)
	if err != nil {
		return err
	}

	defer f.Close()

	for _, target := range targets {
		args, _ := getTargetArguments(target)
		target := templateTargets{
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
		err = t.Execute(f, target)
		if err != nil {
			return err
		}
	}
	return nil
}

// toMakeVars converts input to make vars.
func toMakeVars(args []string) []string {
	makeVars := make([]string, 0)
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
	output := strcase.ToKebab(str)
	output = strings.ReplaceAll(output, ":", "-")
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
