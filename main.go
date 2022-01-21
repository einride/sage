package main

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
	"go.einride.tech/mage-tools/tools/mggo"
	"go.einride.tech/mage-tools/tools/mgyamlfmt"
	"gopkg.in/yaml.v3"
)

var (
	//go:embed example/.mage/magefile.go
	magefile []byte
	//go:embed example/.github/dependabot.yml
	dependabotYaml []byte
	// nolint: gochecknoglobals
	mageDir = mgpath.FromGitRoot(mgpath.MageDir)
)

func main() {
	logger := mglog.Logger("mage-tools")
	usage := func() {
		logger.Info(`Usage:
	init	to initialize mage-tools`)
		os.Exit(0)
	}
	if len(os.Args) <= 1 {
		usage()
	}
	switch os.Args[1] {
	case "init":
		if err := initMageTools(); err != nil {
			log.Fatalf(err.Error())
		}
	default:
		usage()
	}
}

func initMageTools() error {
	logger := mglog.Logger("init")
	logger.Info("initializing mage-tools...")

	if mgpath.FromWorkDir(".") != mgpath.FromGitRoot(".") {
		return fmt.Errorf("can only be generated in git root directory")
	}

	if err := os.Mkdir(mageDir, 0o755); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(mageDir, "magefile.go"), magefile, 0o600); err != nil {
		return err
	}

	_, err := os.Stat("Makefile")
	if err == nil {
		const mm = "Makefile.old"
		logger.Info(fmt.Sprintf("Makefile already exists, renaming  Makefile to %s", mm))
		if err := os.Rename("Makefile", mm); err != nil {
			return err
		}
	}
	cmd := mgtool.Command("go", "mod", "init", "mage-tools")
	cmd.Dir = mageDir
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = mggo.Command("mod", "tidy")
	cmd.Dir = mageDir
	if err := cmd.Run(); err != nil {
		return err
	}
	gitIgnore, err := os.OpenFile(".gitignore", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer gitIgnore.Close()
	relToolsPath, err := filepath.Rel(mgpath.FromGitRoot("."), mgpath.FromTools())
	if err != nil {
		return err
	}
	if _, err := gitIgnore.WriteString(fmt.Sprintf("%s\n", relToolsPath)); err != nil {
		return err
	}
	if err := addToDependabot(); err != nil {
		return err
	}
	// Generate make targets
	cmd = mgtool.Command("go", "run", "go.einride.tech/mage-tools/cmd/build")
	cmd.Dir = mageDir
	if err := cmd.Run(); err != nil {
		return err
	}
	logger.Info(`
Mage-tools has been successfully initialized!

To get started, have a look at the magefile.go in the .mage directory,
and look at https://github.com/einride/mage-tools#readme to learn more
`)
	return nil
}

type dependabot struct {
	PackageEcosystem string `yaml:"package-ecosystem"`
	Directory        string `yaml:"directory"`
	Schedule         struct {
		Interval string `yaml:"interval"`
	} `yaml:"schedule"`
}

func addToDependabot() error {
	dependabotYamlPath := filepath.Join(".github", "dependabot.yml")
	currentConfig, err := ioutil.ReadFile(dependabotYamlPath)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(dependabotYamlPath), 0o755); err != nil {
			return err
		}
		err = os.WriteFile(dependabotYamlPath, dependabotYaml, 0o600)
		if err != nil {
			return err
		}
		return nil
	}
	dependabotMageConfig := dependabot{
		PackageEcosystem: "gomod",
		Directory:        ".mage/",
		Schedule: struct {
			Interval string `yaml:"interval"`
		}{Interval: "daily"},
	}
	marshalDependabot, err := yaml.Marshal(&dependabotMageConfig)
	if err != nil {
		return err
	}
	mageNode := yaml.Node{}
	currentConfig = mgyamlfmt.PreserveEmptyLines(currentConfig)
	if err := yaml.Unmarshal(marshalDependabot, &mageNode); err != nil {
		return err
	}
	dependabotNode := yaml.Node{}
	if err := yaml.Unmarshal(currentConfig, &dependabotNode); err != nil {
		return err
	}
	var updatesIdx int
	for i, k := range dependabotNode.Content[0].Content {
		if k.Value == "updates" {
			updatesIdx = i + 1
			break
		}
	}
	if updatesIdx == 0 {
		return fmt.Errorf("could not find updates key in dependabot.yml")
	}
	dependabotNode.Content[0].Content[updatesIdx].Content =
		append(dependabotNode.Content[0].Content[updatesIdx].Content, mageNode.Content[0])

	var b bytes.Buffer
	encoder := yaml.NewEncoder(&b)
	encoder.SetIndent(2)
	if err := encoder.Encode(&dependabotNode); err != nil {
		return err
	}
	return os.WriteFile(dependabotYamlPath, mgyamlfmt.CleanupPreserveEmptyLines(b.Bytes()), 0o600)
}
