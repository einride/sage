package sops

import (
	"github.com/einride/mage-tools/tools"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

func Sops(file string) error {
	mg.Deps(tools.Sops)
	if err := sh.RunV("sops", file); err != nil {
		return err
	}
	return nil
}
