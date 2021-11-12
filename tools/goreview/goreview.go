package goreview

import (
	"fmt"

	"github.com/einride/mage-tools/tools"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var version string

func SetGoReviewVersion(v string) (string, error) {
	version = v
	return version, nil
}

func Goreview() error {
	mg.Deps(mg.F(tools.Goreview, version))
	// TODO: the args should probably not be hardocded
	fmt.Println("[goreview] reviewing Go code for Einride-specific conventions...")
	err := sh.RunV("goreview", "-c", "1", "./...")
	if err != nil {
		return err
	}
	return nil
}
