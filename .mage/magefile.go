//go:build mage
// +build mage

package main

import (
	"github.com/magefile/mage/mg"
	"go.einride.tech/mage-tools/mgmake"
	"go.einride.tech/mage-tools/mgpath"

	// mage:import
	"go.einride.tech/mage-tools/targets/mgmarkdownfmt"

	// mage:import
	"go.einride.tech/mage-tools/targets/mgconvco"

	// mage:import
	"go.einride.tech/mage-tools/targets/mggo"

	// mage:import
	"go.einride.tech/mage-tools/targets/mggitverifynodiff"

	// mage:import
	"go.einride.tech/mage-tools/targets/mggolangcilint"

	// mage:import
	"go.einride.tech/mage-tools/targets/mggoreview"
)

func init() {
	mgmake.GenerateMakefiles(
		mgmake.Makefile{
			Path:          mgpath.FromGitRoot("Makefile"),
			DefaultTarget: All,
		},
	)
}

func All() {
	mg.Deps(
		mg.F(mgconvco.ConvcoCheck, "origin/main..HEAD"),
		mggolangcilint.GolangciLint,
		mggoreview.Goreview,
		mgmarkdownfmt.FormatMarkdown,
	)
	mg.SerialDeps(
		mggo.GoModTidy,
		mggitverifynodiff.GitVerifyNoDiff,
	)
}
