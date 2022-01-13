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

	"github.com/magefile/mage/mage"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
	"go.einride.tech/mage-tools/targets/mgyamlfmt"
	"gopkg.in/yaml.v3"
)

var (
	//go:embed example/.mage/tools.mk
	toolsMk []byte
	//go:embed example/.mage/magefile.go
	magefile []byte
	//go:embed example/Makefile
	makefile []byte
	//go:embed example/.mage/mgmake_gen.go
	mgmake []byte
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
	case "gen":
		if err := gen(); err != nil {
			log.Fatalf(err.Error())
		}
	default:
		usage()
	}
}

func gen() error {
	mglog.Logger("gen").Info("generating makefiles...")
	executable := filepath.Join(mgpath.Tools(), "mgmake", "magefile")
	if err := mgtool.RunInDir("git", mageDir, "clean", "-fdx", filepath.Dir(executable)); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(mageDir, mgpath.MakeGenGo), mgmake, 0o600); err != nil {
		return err
	}
	if err := mgtool.RunInDir("go", mageDir, "mod", "tidy"); err != nil {
		return err
	}
	if exit := mage.ParseAndRun(os.Stdout, os.Stderr, os.Stdin, []string{"-compile", executable}); exit != 0 {
		return fmt.Errorf("faild to compile magefile binary")
	}
	return mgtool.RunInDir(executable, mageDir, mgpath.GenMakefilesTarget, executable)
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

	if err := os.WriteFile(filepath.Join(mageDir, mgpath.ToolsMk), toolsMk, 0o600); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(mageDir, "magefile.go"), magefile, 0o600); err != nil {
		return err
	}

	_, err := os.Stat("Makefile")
	if err != nil {
		// Write Makefile
		if err := os.WriteFile("Makefile", makefile, 0o600); err != nil {
			return err
		}
	} else {
		const mm = "Makefile.MAGE"
		logger.Info(fmt.Sprintf("Makefile already exist, writing to %s", mm))
		if err := os.WriteFile(mm, makefile, 0o600); err != nil {
			return err
		}
	}
	if err := mgtool.RunInDir("go", mageDir, []string{"mod", "init", "mage-tools"}...); err != nil {
		return err
	}
	if err := mgtool.RunInDir("go", mageDir, []string{"mod", "tidy"}...); err != nil {
		return err
	}
	gitIgnore, err := os.OpenFile(".gitignore", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer gitIgnore.Close()
	if _, err := gitIgnore.WriteString(mgpath.Tools()); err != nil {
		return err
	}
	if err := addToDependabot(); err != nil {
		return err
	}
	// TODO: Output some documentation, next steps after init, and useful links.
	logger.Info("mage-tools initialized!")
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
	currentConfig, cleanup := mgyamlfmt.PreserveEmptyLines(currentConfig)
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
	return os.WriteFile(dependabotYamlPath, cleanup(b.Bytes()), 0o600)
}
