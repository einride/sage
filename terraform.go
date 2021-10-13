package tools

import (
	"fmt"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var TerraformUpgrade = "-upgrade=false"
var TerraformRefresh = "-refresh-only=false"

func TerraformInit(stateBucket string, impersonateServiceAccount string) {
	mg.Deps(Terraform)
	backendConfigBucket := "-backend-config=bucket=" + stateBucket
	backendConfigServiceAccount := "-backend-config=impersonate_service_account=" + impersonateServiceAccount
	fmt.Println("Initing...")
	out, _ := sh.Output(
		"terraform",
		"init",
		"-input=false",
		"-reconfigure",
		backendConfigBucket,
		backendConfigServiceAccount,
		TerraformUpgrade,
	)
	fmt.Println(out)
}

func TerraformPlan(configFile string) {
	varFile := "-var-file=" + configFile
	mg.Deps(Terraform)
	fmt.Println("Planning...")
	fmt.Print(TerraformRefresh)
	out, _ := sh.Output(
		"terraform",
		"plan",
		"-input=false",
		"-no-color",
		"-lock-timeout=120s",
		varFile,
		"-out=terraform.plan",
		TerraformRefresh,
	)
	fmt.Println(out)
}

func TerraformApply(configFile string) {
	varFile := "-var-file=" + configFile
	mg.Deps(Terraform)
	fmt.Println("Applying...")
	out, _ := sh.Output(
		"terraform",
		"apply",
		"-input=false",
		"-no-color",
		"-lock-timeout=120s",
		"-auto-approve=true",
		varFile,
		TerraformRefresh,
	)
	fmt.Println(out)
}
