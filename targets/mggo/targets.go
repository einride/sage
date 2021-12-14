package mggo

import (
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mglog"
)

func GoTest() error {
	mglog.Logger("go-test").Info("running unit tests..")
	return sh.RunV("go", "test", "-race", "-cover", "./...")
}

func GoModTidy() error {
	mglog.Logger("go-mod-tidy").Info("tidying up mod file...")
	return sh.RunV("go", "mod", "tidy", "-v")
}
