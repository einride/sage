// +build mage

package main

import (
	"github.com/einride/mage-tools/tools"
	// mage:import
	_ "github.com/einride/mage-tools/make"
	// mage:import
	_ "github.com/einride/mage-tools/tools/golangcilint"
	// mage:import semantic-release
	_ "github.com/einride/mage-tools/tools/semanticrelease"
)

func init() {
	tools.Path = ".tools"
}
