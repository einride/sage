package mgyamlfmt

import (
	"bytes"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"regexp"

	"go.einride.tech/mage-tools/mgpath"
	"gopkg.in/yaml.v3"
)

func FormatYAML() error {
	return filepath.WalkDir(mgpath.FromGitRoot("."), func(path string, d fs.DirEntry, err error) error {
		if filepath.Ext(path) == ".yml" || filepath.Ext(path) == ".yaml" {
			if err := formatFile(path); err != nil {
				return err
			}
		}
		return nil
	})
}

// PreserveEmptyLines adds a temporary #comment on each empty line in the provided byte array.
// CleanupPreserveEmptyLines can be used to clean up the temporary comments.
func PreserveEmptyLines(src []byte) []byte {
	return bytes.ReplaceAll(src, []byte("\n\n"), []byte("\n#preserveEmptyLine\n"))
}

// CleanupPreserveEmptyLines cleans up the temporary #comment added by PreserveEmptyLines.
func CleanupPreserveEmptyLines(src []byte) []byte {
	// Remove temporary comment.
	indentPreserveComment := regexp.MustCompile("\n\\s+#preserveEmptyLine\n")
	src = indentPreserveComment.ReplaceAll(src, []byte("\n\n"))
	src = bytes.ReplaceAll(src, []byte("\n#preserveEmptyLine\n"), []byte("\n\n"))
	// Remove trailing empty lines
	src = bytes.TrimSpace(src)
	src = append(src, []byte("\n")...)
	return src
}

func formatFile(path string) error {
	node := yaml.Node{}
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	yamlFile = PreserveEmptyLines(yamlFile)
	if err := yaml.Unmarshal(yamlFile, &node); err != nil {
		return err
	}
	var b bytes.Buffer
	encoder := yaml.NewEncoder(&b)
	encoder.SetIndent(2)
	if err := encoder.Encode(&node); err != nil {
		return err
	}
	return ioutil.WriteFile(path, CleanupPreserveEmptyLines(b.Bytes()), 0o600)
}
