// +build mage

package main

import (
	// mage:import
	_ "github.com/einride/mage-tools/make"
	// mage:import
	_ "github.com/einride/mage-tools/golangci_lint"
	// mage:import
	_ "github.com/einride/mage-tools/goreview"
	"github.com/einride/mage-tools/tools"
)

func init() {
	tools.Path = ".tools"
}
