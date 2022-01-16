//go:build mage
// +build mage

package main

import (
	"github.com/magefile/mage/mg"
	"go.einride.tech/mage-tools/mgmake"
	"go.einride.tech/mage-tools/mgpath"

	// mage:import
	"go.einride.tech/mage-tools/targets/mgconvco"

	// mage:import
	"go.einride.tech/mage-tools/targets/mggitverifynodiff"

	// mage:import
	"go.einride.tech/mage-tools/targets/mgyamlfmt"
)

func init() {
	mgmake.GenerateMakefiles(
		mgmake.Makefile{
			Path:          mgpath.FromGitRoot("Makefile"),
			DefaultTarget: All,
		},
	)
}

func SuccessfulInit() {
	println(`
Mage-tools has been successfully initialized!

To get started, have a look at the magefile.go in the .mage directory,
and look at https://github.com/einride/mage-tools#readme to learn more
`)
}

func All() {
	mg.Deps(
		mg.F(mgconvco.ConvcoCheck, "origin/master..HEAD"),
		mgyamlfmt.FormatYaml,
	)
	mg.SerialDeps(
		mggitverifynodiff.GitVerifyNoDiff,
	)
}
