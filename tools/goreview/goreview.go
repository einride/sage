package goreview

import (
	"github.com/einride/mage-tools/tools"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

func Goreview() error {
	mg.Deps(tools.Goreview)
	// TODO: the args should probably not be hardocded
	err := sh.RunV("goreview", "-c", "1", "./...")
	if err != nil {
		return err
	}
	return nil
}