package main

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/tools/sgyamlfmt"
	"gopkg.in/yaml.v3"
)

var (
	//go:embed example/.sage/sagefile.go
	sagefile []byte
	//go:embed example/.github/dependabot.yml
	dependabotYaml []byte
)

func main() {
	ctx := logr.NewContext(context.Background(), sg.NewLogger("sage"))
	logger := logr.FromContextOrDiscard(ctx)
	logger.Info("initializing sage...")
	if sg.FromWorkDir() != sg.FromGitRoot() {
		panic("can only be generated in git root directory")
	}
	if err := os.Mkdir(sg.FromSageDir(), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(sg.FromSageDir(), "sagefile.go"), sagefile, 0o600); err != nil {
		panic(err)
	}
	_, err := os.Stat(sg.FromGitRoot("Makefile"))
	if err == nil {
		const mm = "Makefile.old"
		logger.Info(fmt.Sprintf("Makefile already exists, renaming  Makefile to %s", mm))
		if err := os.Rename(sg.FromGitRoot("Makefile"), sg.FromGitRoot(mm)); err != nil {
			panic(err)
		}
	}
	cmd := sg.Command(ctx, "go", "mod", "init", "sage")
	cmd.Dir = sg.FromSageDir()
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	cmd = sg.Command(ctx, "go", "mod", "tidy")
	cmd.Dir = sg.FromSageDir()
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	gitIgnore, err := os.OpenFile(".gitignore", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		panic(err)
	}
	defer gitIgnore.Close()
	relToolsPath, err := filepath.Rel(sg.FromGitRoot("."), sg.FromToolsDir())
	if err != nil {
		panic(err)
	}
	if _, err := gitIgnore.WriteString(fmt.Sprintf("%s\n", relToolsPath)); err != nil {
		panic(err)
	}
	if err := addToDependabot(); err != nil {
		panic(err)
	}
	// Generate make targets
	cmd = sg.Command(ctx, "go", "run", "go.einride.tech/sage/cmd/build")
	cmd.Dir = sg.FromSageDir()
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	logger.Info(`
sage has been successfully initialized!

To get started, have a look at the sagefile.go in the .sage directory,
and look at https://github.com/einride/sage#readme to learn more
`)
}

type dependabot struct {
	PackageEcosystem string `yaml:"package-ecosystem"`
	Directory        string `yaml:"directory"`
	Schedule         struct {
		Interval string `yaml:"interval"`
	} `yaml:"schedule"`
}

func addToDependabot() error {
	dependabotYamlPath := sg.FromGitRoot(".github", "dependabot.yml")
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
	dependabotSageConfig := dependabot{
		PackageEcosystem: "gomod",
		Directory:        ".sage",
		Schedule: struct {
			Interval string `yaml:"interval"`
		}{Interval: "daily"},
	}
	marshalDependabot, err := yaml.Marshal(&dependabotSageConfig)
	if err != nil {
		return err
	}
	var sageNode yaml.Node
	currentConfig = sgyamlfmt.PreserveEmptyLines(currentConfig)
	if err := yaml.Unmarshal(marshalDependabot, &sageNode); err != nil {
		return err
	}
	var dependabotNode yaml.Node
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
		append(dependabotNode.Content[0].Content[updatesIdx].Content, sageNode.Content[0])
	var b bytes.Buffer
	encoder := yaml.NewEncoder(&b)
	encoder.SetIndent(2)
	if err := encoder.Encode(&dependabotNode); err != nil {
		return err
	}
	return os.WriteFile(dependabotYamlPath, sgyamlfmt.CleanupPreserveEmptyLines(b.Bytes()), 0o600)
}
