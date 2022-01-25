package main

import (
	"context"

	"github.com/go-logr/logr"
	"go.einride.tech/mage-tools/mg"
	"go.einride.tech/mage-tools/tools/mgconvco"
	"go.einride.tech/mage-tools/tools/mggit"
	"go.einride.tech/mage-tools/tools/mggo"
	"go.einride.tech/mage-tools/tools/mggolangcilint"
	"go.einride.tech/mage-tools/tools/mggoreview"
	"go.einride.tech/mage-tools/tools/mgmarkdownfmt"
	"go.einride.tech/mage-tools/tools/mgyamlfmt"
)

func init() {
	mg.GenerateMakefiles(
		mg.Makefile{
			Path:          mg.FromGitRoot("Makefile"),
			DefaultTarget: All,
		},
	)
}

func All(ctx context.Context) error {
	mg.Deps(ctx, ConvcoCheck, GolangciLint, GoReview, GoTest, FormatMarkdown, FormatYAML)
	mg.SerialDeps(ctx, GoModTidy, GitVerifyNoDiff)
	return nil
}

func FormatYAML(ctx context.Context) error {
	logr.FromContextOrDiscard(ctx).Info("formatting YAML files...")
	return mgyamlfmt.FormatYAML(ctx)
}

func GoModTidy(ctx context.Context) error {
	logr.FromContextOrDiscard(ctx).Info("tidying Go module files...")
	return mg.Command(ctx, "go", "mod", "tidy", "-v").Run()
}

func GoTest(ctx context.Context) error {
	logr.FromContextOrDiscard(ctx).Info("running Go tests...")
	return mggo.TestCommand(ctx).Run()
}

func GoReview(ctx context.Context) error {
	logr.FromContextOrDiscard(ctx).Info("reviewing Go files...")
	return mggoreview.Command(ctx, "-c", "1", "./...").Run()
}

func GolangciLint(ctx context.Context) error {
	logr.FromContextOrDiscard(ctx).Info("linting Go files...")
	return mggolangcilint.RunCommand(ctx).Run()
}

func FormatMarkdown(ctx context.Context) error {
	logr.FromContextOrDiscard(ctx).Info("formatting Markdown files...")
	return mgmarkdownfmt.Command(ctx, "-w", ".").Run()
}

func ConvcoCheck(ctx context.Context) error {
	logr.FromContextOrDiscard(ctx).Info("checking git commits...")
	return mgconvco.Command(ctx, "check", "origin/master..HEAD").Run()
}

func GitVerifyNoDiff(ctx context.Context) error {
	logr.FromContextOrDiscard(ctx).Info("verifying that git has no diff...")
	return mggit.VerifyNoDiff(ctx)
}
