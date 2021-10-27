package make

import (
	"bytes"
	"fmt"
	"github.com/einride/mage-tools/tools"
	"os"
	"regexp"
	"strings"

	"github.com/magefile/mage/mage"
)

func GenerateMakefile(makefile string) error {
	targets, err := listTargets()
	if err != nil {
		return err
	}
	// Write makefile to disk
	f, err := os.Create(tools.CwdPath(makefile))
	if err != nil {
		return err
	}

	defer f.Close()

	// This part will be written to the start of the makefile
	staticContent := fmt.Sprintf(
		`mage_cwd := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
make_dir := $(abspath $(dir $(firstword $(MAKEFILE_LIST))))

`,
	)
	_, err = f.WriteString(staticContent)
	if err != nil {
		return err
	}

	for _, i := range targets {
		makeFormat := toMakeFormat(strings.Fields(i)[0])
		dynamicContent := fmt.Sprintf(
			`.PHONY: %s
%s:
%s@cd $(mage_cwd) && $(mage) -w $(make_dir) %s

`, makeFormat, makeFormat, "\t", i)
		_, err = f.WriteString(dynamicContent)
		if err != nil {
			return err
		}
	}
	return nil
}

func listTargets() ([]string, error) {
	var b bytes.Buffer
	err := invokeMage(mage.Invocation{
		Stdout: &b,
		List:   true,
		Stderr: os.Stderr,
	})
	if err != nil {
		return []string{}, err
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
			if strings.Contains(parts[0], "printMakeTargets") {
				continue
			}
			// Get input arguments for mage target
			args, err := getTargetArguments(parts[0])
			if err != nil {
				return []string{}, err
			} else if args != "" {
				parts[0] = parts[0] + " " + args
			}

			targets = append(targets, parts[0])
		}
	}

	return targets, nil
}

func toMakeFormat(str string) string {
	matchFirstCap := regexp.MustCompile("([A-Z])([A-Z][a-z])")
	matchAllCap := regexp.MustCompile("([a-z0-9])([A-Z])")

	output := matchFirstCap.ReplaceAllString(str, "${1}-${2}")
	output = matchAllCap.ReplaceAllString(output, "${1}-${2}")
	output = strings.ReplaceAll(output, ":", "-")
	return strings.ToLower(output)
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
	for _, a := range lines {
		parts := strings.Fields(a)[2:]
		if len(parts) == 0 {
			continue
		}
		a = strings.Join(parts, " ")
		a = strings.ReplaceAll(a, "<", "$(")
		a = strings.ReplaceAll(a, ">", ")")
		arguments = a
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
