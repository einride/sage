//go:build mage
// +build mage

package main

import (
	"github.com/magefile/mage/mg"

	// mage:import
	"go.einride.tech/mage-tools/targets/mgcommitlint"

	// mage:import
	"go.einride.tech/mage-tools/targets/mggitverifynodiff"
)

func All() {
	mg.Deps(
		mg.F(mgcommitlint.Commitlint, "main"),
	)
	mg.SerialDeps(
		mggitverifynodiff.GitVerifyNoDiff,
	)
}
