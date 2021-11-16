package semanticrelease

import (
	"fmt"
	"path/filepath"

	"github.com/einride/mage-tools/tools"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
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
	fmt.Println("[semantic-release] creating release...")
	err = sh.RunV(tools.SemanticReleasePath, args...)
	if err != nil {
		return err
	}
	return nil
}
