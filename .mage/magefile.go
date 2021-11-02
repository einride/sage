// +build mage

package main

import (
	"github.com/einride/mage-tools/tools"
	// mage:import
	_ "github.com/einride/mage-tools/make"
	// mage:import
	_ "github.com/einride/mage-tools/tools/golangcilint"
)

func init() {
	tools.Path = ".tools"
}
