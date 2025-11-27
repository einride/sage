package main

import (
	"context"
	"fmt"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sg/version"
	"go.einride.tech/sage/tools/sgbackstage"
	"go.einride.tech/sage/tools/sgconvco"
	"go.einride.tech/sage/tools/sggit"
	"go.einride.tech/sage/tools/sggo"
	"go.einride.tech/sage/tools/sggolangcilintv2"
	"go.einride.tech/sage/tools/sggolicenses"
	"go.einride.tech/sage/tools/sgmdformat"
	"go.einride.tech/sage/tools/sgyamlfmt"
)

func main() {
	sg.GenerateMakefiles(
		sg.Makefile{
			Path:          sg.FromGitRoot("Makefile"),
			DefaultTarget: Default,
		},
	)
}

func Default(ctx context.Context) error {
	sg.Deps(ctx, ConvcoCheck, GoLint, GoTest, FormatMarkdown, FormatYaml, BackstageValidate)
	sg.SerialDeps(ctx, GoModTidy)
	sg.SerialDeps(ctx, GoLicenses, GitVerifyNoDiff)
	return nil
}

func GoModTidy(ctx context.Context) error {
	sg.Logger(ctx).Println("tidying Go module files...")
	return sg.Command(ctx, "go", "mod", "tidy", "-v").Run()
}

func GoTest(ctx context.Context) error {
	sg.Logger(ctx).Println("running Go tests...")
	return sggo.TestCommand(ctx).Run()
}

func GoLint(ctx context.Context) error {
	sg.Logger(ctx).Println("linting Go files...")
	return sggolangcilintv2.Run(
		ctx,
		sggolangcilintv2.Config{RunRelativePathMode: sggolangcilintv2.RunRelativePathModeGitRoot},
	)
}

func GoLintFix(ctx context.Context) error {
	sg.Logger(ctx).Println("fixing Go files...")
	return sggolangcilintv2.Fix(
		ctx,
		sggolangcilintv2.Config{RunRelativePathMode: sggolangcilintv2.RunRelativePathModeGitRoot},
	)
}

func GoFormat(ctx context.Context) error {
	sg.Logger(ctx).Println("formatting Go files...")
	return sggolangcilintv2.Fmt(
		ctx,
		sggolangcilintv2.Config{RunRelativePathMode: sggolangcilintv2.RunRelativePathModeGitRoot},
	)
}

func GoLicenses(ctx context.Context) error {
	sg.Logger(ctx).Println("checking Go licenses...")
	return sggolicenses.Check(ctx)
}

func FormatMarkdown(ctx context.Context) error {
	sg.Logger(ctx).Println("formatting Markdown files...")
	return sgmdformat.Command(ctx).Run()
}

func FormatYaml(ctx context.Context) error {
	sg.Logger(ctx).Println("formatting YAML files...")
	return sgyamlfmt.Run(ctx)
}

func ConvcoCheck(ctx context.Context) error {
	sg.Logger(ctx).Println("checking git commits...")
	return sgconvco.Command(ctx, "check", "origin/master..HEAD").Run()
}

func BackstageValidate(ctx context.Context) error {
	sg.Logger(ctx).Println("validating Backstage files...")
	return sgbackstage.Validate(ctx)
}

func GitVerifyNoDiff(ctx context.Context) error {
	sg.Logger(ctx).Println("verifying that git has no diff...")
	return sggit.VerifyNoDiff(ctx)
}

// CheckToolVersions checks for outdated tool versions.
// Usage: make check-tool-versions tool=NAME apply=true pr=true
//
// Parameters:
//   - tool: tool name, or "all" to check all tools
//   - apply: "true" to update version in file, "false" for dry-run
//   - pr: "true" to create PR after update, "false" to skip
//
// Examples:
//
//	make check-tool-versions tool=all apply=false pr=false     # check all (dry-run)
//	make check-tool-versions tool=buf apply=false pr=false     # check single tool
//	make check-tool-versions tool=buf apply=true pr=false      # update version in file
//	make check-tool-versions tool=buf apply=true pr=true       # update + create PR
func CheckToolVersions(ctx context.Context, tool, apply, pr string) error {
	applyBool := apply == "true"
	prBool := pr == "true"

	if prBool && !applyBool {
		return fmt.Errorf("pr=true requires apply=true")
	}

	// Get tools to check (tool="all" means check all tools)
	var results []version.CheckResult
	if tool != "all" {
		result, err := version.CheckByName(ctx, tool)
		if err != nil {
			return err
		}
		results = []version.CheckResult{result}
	} else {
		results = version.CheckAll(ctx)
	}

	// Print results and optionally apply updates
	hasUpdates := false
	for _, result := range results {
		if result.Error != nil {
			sg.Logger(ctx).Printf("%s: error: %v\n", result.Tool.Name, result.Error)
			continue
		}

		if result.NeedsUpdate {
			hasUpdates = true
			sg.Logger(ctx).Printf("%s: %s -> %s [outdated]\n",
				result.Tool.Name, result.Tool.CurrentVersion, result.LatestVersion)

			if applyBool {
				if err := version.UpdateVersion(
					result.Tool.FilePath,
					result.Tool.CurrentVersion,
					result.LatestVersion,
				); err != nil {
					return fmt.Errorf("failed to update %s: %w", result.Tool.Name, err)
				}
				sg.Logger(ctx).Printf("%s: updated version in %s\n", result.Tool.Name, result.Tool.FilePath)

				if prBool {
					sg.Logger(ctx).Printf("%s: creating PR...\n", result.Tool.Name)
					if err := version.CreatePR(ctx, result.Tool, result.LatestVersion); err != nil {
						return fmt.Errorf("failed to create PR for %s: %w", result.Tool.Name, err)
					}
				}
			}
		} else {
			sg.Logger(ctx).Printf("%s: %s [up to date]\n", result.Tool.Name, result.Tool.CurrentVersion)
		}
	}

	if !hasUpdates && tool == "all" {
		sg.Logger(ctx).Println("All tools are up to date.")
	}

	return nil
}
