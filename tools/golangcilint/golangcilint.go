package golangcilint

import (
	"fmt"
	"github.com/einride/mage-tools/tools"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

func GolangciLint() error {
	mg.Deps(tools.GolangciLint)
	fmt.Println("[golangci-lint] linting Go code with golangci-lint...")
	err := sh.RunV("golangci-lint", "run")
	if err != nil {
		return err
	}
	return nil
}
