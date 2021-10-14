package terraform

import (
	"fmt"
	"github.com/einride/mage-tools"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

func GhReviewTerraformPlan(prNumber string, gcpProject string) {
	terraformPlanFile := "terraform.plan"
	mg.Deps(
		tools.Terraform,
		tools.GHComment,
		mg.F(tools.FileExist, terraformPlanFile),
	)

	comment, _ := sh.Output(
		"terraform",
		"show",
		"-no-color",
		terraformPlanFile,
	)
	comment = fmt.Sprintf("```"+"hcl\n%s\n"+"```", comment)
	ghComment := fmt.Sprintf(`
<div>
<img align="right" width="120" src="https://www.terraform.io/assets/images/logo-text-8c3ba8a6.svg" />
<h2>Terraform Plan (%s)</h2>
</div>

%s
`, gcpProject, comment)

	out, _ := sh.Output(
		"ghcomment",
		"--pr",
		prNumber,
		"--signkey",
		gcpProject,
		"--comment",
		ghComment,
	)
	fmt.Println(out)
}
