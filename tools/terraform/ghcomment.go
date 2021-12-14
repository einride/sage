package terraform

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mgtool"
	"go.einride.tech/mage-tools/tools"
	"go.einride.tech/mage-tools/tools/gh"
)

var (
	ghCommentVersion string
	CommentBinary    string
)

func SetGhCommentVersion(v string) (string, error) {
	ghCommentVersion = v
	return ghCommentVersion, nil
}

func GhReviewTerraformPlan(prNumber string, gcpProject string) error {
	terraformPlanFile := "terraform.plan"
	mg.Deps(
		mg.F(terraform, tfVersion),
		mg.F(comment, ghCommentVersion),
		mg.F(mgtool.Exists, terraformPlanFile),
	)

	comment, err := sh.Output(
		Binary,
		"show",
		"-no-color",
		terraformPlanFile,
	)
	if err != nil {
		return err
	}
	comment = fmt.Sprintf("```"+"hcl\n%s\n"+"```", comment)
	ghComment := fmt.Sprintf(`
<div>
<img align="right" width="120" src="https://www.terraform.io/assets/images/logo-text-8c3ba8a6.svg" />
<h2>Terraform Plan (%s)</h2>
</div>

%s
`, gcpProject, comment)

	fmt.Println("[ghcomment] commenting terraform plan on pr...")
	err = sh.RunV(
		CommentBinary,
		"--pr",
		prNumber,
		"--signkey",
		gcpProject,
		"--comment",
		ghComment,
	)
	if err != nil {
		return err
	}
	return nil
}

func comment(ctx context.Context, version string) error {
	mg.Deps(mg.F(gh.GH, version))
	const binaryName = "ghcomment"
	const defaultVersion = "0.2.1"

	if version == "" {
		version = defaultVersion
	} else {
		supportedVersions := []string{"0.2.1"}
		if err := tools.IsSupportedVersion(supportedVersions, version, binaryName); err != nil {
			return err
		}
	}

	binDir := filepath.Join(tools.GetPath(), binaryName, version, "bin")
	binary := filepath.Join(binDir, binaryName)
	CommentBinary = binary

	// Check if binary already exist
	if mgtool.Exists(binary) == nil {
		return nil
	}

	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	ghVersion := "v" + version
	pattern := fmt.Sprintf("*%s_%s.tar.gz", hostOS, hostArch)
	archive := fmt.Sprintf("%s/ghcomment_%s_%s_%s.tar.gz", binDir, version, hostOS, hostArch)

	if err := sh.Run(
		gh.Binary,
		"release",
		"download",
		"--repo",
		"einride/ghcomment",
		ghVersion,
		"--pattern",
		pattern,
		"--dir",
		binDir,
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}

	if err := mgtool.FromLocal(
		ctx,
		archive,
		mgtool.WithDestinationDir(binDir),
		mgtool.WithUntarGz(),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}

	return nil
}
