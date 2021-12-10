// +build mage

package main

import (
	"github.com/magefile/mage/mg"

	// mage:import
	_ "github.com/einride/mage-tools/maketargets"
	// mage:import
	"github.com/einride/mage-tools/tools/goreview"
	// mage:import
	"github.com/einride/mage-tools/tools/golangcilint"
	// mage:import
	"github.com/einride/mage-tools/tools/commitlint"
	// mage:import semantic-release
	_ "github.com/einride/mage-tools/tools/semanticrelease"
	// mage:import
	"github.com/einride/mage-tools/tools/gitverifynodiff"
	// mage:import
	"github.com/einride/mage-tools/tools/common"
)

func All() {
	mg.Deps(
		mg.F(commitlint.Commitlint, "main"),
		golangcilint.GolangciLint,
		goreview.Goreview,
	)
	mg.SerialDeps(
		common.GoModTidy,
		gitverifynodiff.GitVerifyNoDiff,
	)
}
