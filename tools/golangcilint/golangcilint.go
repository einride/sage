package golangcilint

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/einride/mage-tools/tools"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const defaultConfig = `run:
  timeout: 10m
  skip-dirs:
    - gen

linters:
  enable-all: true
  disable:
    - dupl # allow duplication
    - funlen # allow long functions
    - gomnd # allow some magic numbers
    - wsl # unwanted amount of whitespace
    - godox # allow TODOs
    - interfacer # deprecated by the author for having too many false positives
    - gocognit # allow higher cognitive complexity
    - testpackage # unwanted convention
    - nestif # allow deep nesting
    - unparam # allow constant parameters
    - goerr113 # allow "dynamic" errors
    - nlreturn # don't enforce newline before return
    - paralleltest # TODO: fix issues and enable
    - exhaustivestruct # don't require exhaustive struct fields
    - wrapcheck # don't require wrapping everywhere
`

var version string

func SetGolangciLintVersion(v string) (string, error) {
	version = v
	return version, nil
}

func GolangciLint() error {
	mg.Deps(mg.F(tools.GolangciLint, version))
	configPath := ".golangci.yml"
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		configPath := filepath.Join(tools.Path, "golangci-lint", ".golangci.yml")
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0o644); err != nil {
			return err
		}
	}
	fmt.Println("[golangci-lint] linting Go code with golangci-lint...")
	if err := sh.RunV(tools.GolangciLintPath, "run", "-c", configPath); err != nil {
		return err
	}
	return nil
}
