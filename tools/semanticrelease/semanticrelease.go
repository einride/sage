package semanticrelease

import (
	"github.com/einride/mage-tools/tools"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"path/filepath"
)

func Run(branch string, ci bool) error {
	mg.Deps(mg.F(tools.SemanticRelease, branch))
	path, err := filepath.Abs(tools.Path)
	if err != nil {
		return err
	}
	releaserc := filepath.Join(path, "semantic-release", ".releaserc.json")
	args := []string{
		"--extends",
		releaserc,
	}
	if ci {
		args = append(args, "--ci")
	}
	err = sh.RunV("semantic-release", args...)
	if err != nil {
		return err
	}
	return nil
}
