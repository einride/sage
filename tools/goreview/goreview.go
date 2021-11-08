package goreview

import (
	"fmt"

	"github.com/einride/mage-tools/tools"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

func Goreview() error {
	mg.Deps(tools.Goreview)
	// TODO: the args should probably not be hardocded
	fmt.Println("[goreview] reviewing Go code for Einride-specific conventions...")
	err := sh.RunV("goreview", "-c", "1", "./...")
	if err != nil {
		return err
	}
	return nil
}
