// +build mage

package main

import (
	// mage:import
	_ "github.com/einride/mage-tools/make"
	// mage:import
	_ "github.com/einride/mage-tools/tools/golangci_lint"
	"github.com/einride/mage-tools/tools"
)

func init() {
	tools.Path = ".tools"
}
