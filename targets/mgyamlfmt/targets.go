package mgyamlfmt

import (
	"bytes"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgpath"
	"gopkg.in/yaml.v3"
)

func FormatYaml() error {
	logger := mglog.Logger("format-yaml")
	logger.Info("formatting yaml files...")
	return filepath.WalkDir(mgpath.FromGitRoot("."), func(path string, d fs.DirEntry, err error) error {
		if filepath.Ext(path) == ".yml" || filepath.Ext(path) == ".yaml" {
			if err := formatFile(path); err != nil {
				return err
			}
		}
		return nil
	})
}

func formatFile(path string) error {
	node := yaml.Node{}
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	// Insert temporary comment to preserve empty lines.
	yamlFile = bytes.ReplaceAll(yamlFile, []byte("\n\n"), []byte("\n#preserveEmptyLine\n"))
	if err := yaml.Unmarshal(yamlFile, &node); err != nil {
		return err
	}
	var b bytes.Buffer
	encoder := yaml.NewEncoder(&b)
	encoder.SetIndent(2)
	if err := encoder.Encode(&node); err != nil {
		return err
	}
	// Remove temporary comment.
	indentPreserveComment := regexp.MustCompile("\n\\s+#preserveEmptyLine\n")
	out := b.String()
	out = indentPreserveComment.ReplaceAllString(out, "\n\n")
	out = strings.ReplaceAll(out, "\n#preserveEmptyLine\n", "\n\n")
	// Remove trailing empty lines
	out = strings.TrimSpace(out) + "\n"
	return ioutil.WriteFile(path, []byte(out), 0o600)
}
