//go:build mage
// +build mage

package main

import (
	"context"

	"github.com/magefile/mage/mg"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgmake"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/mgtool"
	"go.einride.tech/mage-tools/tools/mgconvco"
	"go.einride.tech/mage-tools/tools/mggit"
	"go.einride.tech/mage-tools/tools/mggo"
	"go.einride.tech/mage-tools/tools/mggolangcilint"
	"go.einride.tech/mage-tools/tools/mggoreview"
	"go.einride.tech/mage-tools/tools/mgmarkdownfmt"
	"go.einride.tech/mage-tools/tools/mgyamlfmt"
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
		ConvcoCheck,
		GolangciLint,
		Goreview,
		GoTest,
		FormatMarkdown,
		FormatYAML,
	)
	mg.SerialDeps(
		GoModTidy,
		GitVerifyNoDiff,
	)
}

func FormatYAML(ctx context.Context) error {
	ctx = mglog.NewContext(ctx, "format-yaml")
	logr.FromContextOrDiscard(ctx).Info("formatting YAML files...")
	return mgyamlfmt.FormatYAML(ctx)
}

func GoModTidy(ctx context.Context) error {
	ctx = mglog.NewContext(ctx, "go-mod-tidy")
	logr.FromContextOrDiscard(ctx).Info("tidying Go module files...")
	return mgtool.Command(ctx, "go", "mod", "tidy", "-v").Run()
}

func GoTest(ctx context.Context) error {
	ctx = mglog.NewContext(ctx, "go-test")
	logr.FromContextOrDiscard(ctx).Info("running Go unit tests..")
	return mggo.TestCommand(ctx).Run()
}

func Goreview(ctx context.Context) error {
	ctx = mglog.NewContext(ctx, "goreview")
	logr.FromContextOrDiscard(ctx).Info("running...")
	return mggoreview.Command(ctx, "-c", "1", "./...").Run()
}

func GolangciLint(ctx context.Context) error {
	ctx = mglog.NewContext(ctx, "golangci-lint")
	logr.FromContextOrDiscard(ctx).Info("running...")
	return mggolangcilint.RunCommand(ctx).Run()
}

func FormatMarkdown(ctx context.Context) error {
	ctx = mglog.NewContext(ctx, "format-markdown")
	logr.FromContextOrDiscard(ctx).Info("formatting Markdown files...")
	return mgmarkdownfmt.Command(ctx, "-w", ".").Run()
}

func ConvcoCheck(ctx context.Context) error {
	ctx = mglog.NewContext(ctx, "convco-check")
	logr.FromContextOrDiscard(ctx).Info("checking git commits...")
	return mgconvco.Command(ctx, "check", "origin/main..HEAD").Run()
}

func GitVerifyNoDiff(ctx context.Context) error {
	ctx = mglog.NewContext(ctx, "git-verify-no-diff")
	logr.FromContextOrDiscard(ctx).Info("verifying that git has no diff...")
	return mggit.VerifyNoDiff(ctx)
}
