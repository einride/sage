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
// Usage: make check-tool-versions tool=NAME apply=true verify=true pr=true
//
// Parameters:
//   - tool: tool name, or "all" to check all tools
//   - apply: "true" to update version in file, "false" for dry-run
//   - verify: "true" to verify new version installs correctly, "false" to skip
//   - pr: "true" to create PR after update, "false" to skip
//
// Examples:
//
//	make check-tool-versions tool=all apply=false verify=false pr=false   # check all (dry-run)
//	make check-tool-versions tool=buf apply=true verify=true pr=false     # update + verify
//	make check-tool-versions tool=buf apply=true verify=true pr=true      # update + verify + PR
func CheckToolVersions(ctx context.Context, tool, apply, verify, pr string) error {
	applyBool := apply == "true"
	verifyBool := verify == "true"
	prBool := pr == "true"

	if prBool && !applyBool {
		return fmt.Errorf("pr=true requires apply=true")
	}
	if verifyBool && !applyBool {
		return fmt.Errorf("verify=true requires apply=true")
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

				// Verify the new version can be installed
				if verifyBool && result.Tool.Verify != nil {
					sg.Logger(ctx).Printf("%s: verifying new version...\n", result.Tool.Name)
					if err := result.Tool.Verify(ctx); err != nil {
						// Revert the version change on failure
						sg.Logger(ctx).Printf("%s: verify failed, reverting: %v\n", result.Tool.Name, err)
						revertErr := version.UpdateVersion(
							result.Tool.FilePath,
							result.LatestVersion,
							result.Tool.CurrentVersion,
						)
						if revertErr != nil {
							return fmt.Errorf("revert %s failed: %w", result.Tool.Name, revertErr)
						}
						return fmt.Errorf("verify %s failed: %w", result.Tool.Name, err)
					}
					sg.Logger(ctx).Printf("%s: verified successfully\n", result.Tool.Name)
				}

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
