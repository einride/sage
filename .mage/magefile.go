//go:build mage
// +build mage

package main

import (
	"github.com/magefile/mage/mg"
	// mage:import
	_ "go.einride.tech/mage-tools/mgmake"

	// mage:import
	"go.einride.tech/mage-tools/targets/mgcommitlint"

	// mage:import
	"go.einride.tech/mage-tools/targets/mggo"

	// mage:import
	"go.einride.tech/mage-tools/targets/mggitverifynodiff"

	// mage:import
	"go.einride.tech/mage-tools/targets/mggolangcilint"

	// mage:import
	"go.einride.tech/mage-tools/targets/mggoreview"

	// mage:import
	_ "go.einride.tech/mage-tools/targets/mgsemanticrelease"
)

func All() {
	mg.Deps(
		mg.F(mgcommitlint.Commitlint, "main"),
		mggolangcilint.GolangciLint,
		mggoreview.Goreview,
	)
	mg.SerialDeps(
		mggo.GoModTidy,
		mggitverifynodiff.GitVerifyNoDiff,
	)
}
