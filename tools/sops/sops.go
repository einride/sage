package sops

import (
	"github.com/einride/mage-tools/tools"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var version string

func SetSopsVersion(v string) (string, error) {
	version = v
	return version, nil
}

func Sops(file string) error {
	mg.Deps(mg.F(tools.Sops, version))
	if err := sh.RunV(tools.SopsPath, file); err != nil {
		return err
	}
	return nil
}
