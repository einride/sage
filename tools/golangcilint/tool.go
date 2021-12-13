package golangcilint

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/einride/mage-tools/file"
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

var (
	version string
	Binary  string
)

func SetGolangciLintVersion(v string) (string, error) {
	version = v
	return version, nil
}

func GolangciLint() error {
	mg.Deps(mg.F(golangciLint, version))
	configPath := ".golangci.yml"
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		configPath := filepath.Join(tools.GetPath(), "golangci-lint", ".golangci.yml")
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0o644); err != nil {
			return err
		}
	}
	fmt.Println("[golangci-lint] linting Go code with golangci-lint...")
	return sh.RunV(Binary, "run", "-c", configPath)
}

func golangciLint(version string) error {
	const binaryName = "golangci-lint"
	const defaultVersion = "1.42.1"

	if version == "" {
		version = defaultVersion
	} else {
		supportedVersions := []string{"1.42.1"}
		if err := tools.IsSupportedVersion(supportedVersions, version, binaryName); err != nil {
			return err
		}
	}
	toolDir := filepath.Join(tools.GetPath(), binaryName)
	binDir := filepath.Join(toolDir, version, "bin")
	binary := filepath.Join(binDir, binaryName)
	Binary = binary

	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	golangciLint := fmt.Sprintf("golangci-lint-%s-%s-%s", version, hostOS, hostArch)

	binURL := fmt.Sprintf(
		"https://github.com/golangci/golangci-lint/releases/download/v%s/%s.tar.gz",
		version,
		golangciLint,
	)

	if err := file.FromRemote(
		binURL,
		file.WithName(filepath.Base(binary)),
		file.WithDestinationDir(binDir),
		file.WithUntarGz(),
		file.WithRenameFile(fmt.Sprintf("%s/golangci-lint", golangciLint), binaryName),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	return nil
}
