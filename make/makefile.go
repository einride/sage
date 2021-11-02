package make

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
	"github.com/magefile/mage/mage"
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
		parts := strings.Fields(target)
		target := templateTargets{
			MakeTarget: toMakeTarget(parts[0]),
			MageTarget: target,
			Args:       toMakeVars(parts[1:]),
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

// toMakeVars converts input to make vars
func toMakeVars(args []string) []string {
	var list []string
	for _, arg := range args {
		arg = strcase.ToSnakeWithIgnore(arg, " ")
		arg = strings.ReplaceAll(arg, "$(", "")
		arg = strings.ReplaceAll(arg, ")", "")
		list = append(list, arg)
	}
	return list
}

// toMakeTarget converts input to make target format
func toMakeTarget(str string) string {
	output := strcase.ToKebab(str)
	output = strings.ReplaceAll(output, ":", "-")
	return strings.ToLower(output)
}

func listTargets() ([]string, error) {
	var b bytes.Buffer
	err := invokeMage(mage.Invocation{
		Stdout: &b,
		List:   true,
		Stderr: os.Stderr,
	})
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(b.String()), "\n")
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
			// Get input arguments for mage target
			args, err := getTargetArguments(parts[0])
			if err != nil {
				return nil, err
			} else if args != "" {
				parts[0] = parts[0] + " " + args
			}

			targets = append(targets, parts[0])
		}
	}

	return targets, nil
}

func getTargetArguments(name string) (string, error) {
	var b bytes.Buffer
	err := invokeMage(mage.Invocation{
		Stdout: &b,
		Stderr: os.Stderr,
		Help:   true,
		Args:   []string{name},
	})
	if err != nil {
		return "", err
	}

	lines := strings.Split(strings.TrimSpace(b.String()), "\n\n")
	if strings.HasPrefix(lines[0], "Usage:") {
		lines = lines[1:]
	}

	var arguments string
	for _, arg := range lines {
		parts := strings.Fields(arg)[2:]
		if len(parts) == 0 {
			continue
		}
		arg = strings.Join(parts, " ")
		arg = strcase.ToSnakeWithIgnore(arg, " ")
		arg = strings.ReplaceAll(arg, "<", "$(")
		arg = strings.ReplaceAll(arg, ">", ")")
		arguments = arg
	}

	return arguments, nil
}

func invokeMage(args mage.Invocation) error {
	out := mage.Invoke(args)
	if out != 0 {
		return fmt.Errorf("mage exited with status code %d", out)
	}
	return nil
}
