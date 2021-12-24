//go:build mage
// +build mage

package main

import (
	"github.com/magefile/mage/mg"

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

	// mage:import
	"go.einride.tech/mage-tools/targets/mgprettier"
)

func All() {
	mg.Deps(
		mg.F(mgcommitlint.Commitlint, "main"),
		mggolangcilint.GolangciLint,
		mggoreview.Goreview,
		mgprettier.FormatMarkdown,
	)
	mg.SerialDeps(
		mggo.GoModTidy,
		mggitverifynodiff.GitVerifyNoDiff,
	)
}
