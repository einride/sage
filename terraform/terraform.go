package terraform

import (
	"fmt"
	"github.com/einride/mage-tools"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var DevConfig TfConfig
var ProdConfig TfConfig

type TfConfig struct {
	ServiceAccount string
	StateBucket    string
	Upgrade        string
	Refresh        string
	VarFile        string
}

func SetupDev(config TfConfig) error {
	DevConfig = config
	if DevConfig.Refresh == "" {
		DevConfig.Refresh = "-refresh-only=false"
	}
	if DevConfig.Upgrade == "" {
		DevConfig.Upgrade = "-upgrade=false"
	}
	return nil
}

func SetupProd(config TfConfig) error {
	ProdConfig = config
	if ProdConfig.Refresh == "" {
		ProdConfig.Refresh = "-refresh-only=false"
	}
	if ProdConfig.Upgrade == "" {
		ProdConfig.Upgrade = "-upgrade=false"
	}
	return nil
}

func Init(config TfConfig) {
	mg.Deps(tools.Terraform)
	backendConfigBucket := "-backend-config=bucket=" + config.StateBucket
	backendConfigServiceAccount := "-backend-config=impersonate_service_account=" + config.ServiceAccount
	fmt.Println("Initing...")
	out, _ := sh.Output(
		"terraform",
		"init",
		"-input=false",
		"-reconfigure",
		backendConfigBucket,
		backendConfigServiceAccount,
		config.Upgrade,
	)
	fmt.Println(out)
}

func Plan(config TfConfig) {
	varFile := "-var-file=" + config.VarFile
	mg.Deps(tools.Terraform)
	fmt.Println("Planning...")
	out, _ := sh.Output(
		"terraform",
		"plan",
		"-input=false",
		"-no-color",
		"-lock-timeout=120s",
		varFile,
		"-out=terraform.plan",
		config.Refresh,
	)
	fmt.Println(out)
}

func Apply(config TfConfig) {
	varFile := "-var-file=" + config.VarFile
	mg.Deps(tools.Terraform)
	fmt.Println("Applying...")
	out, _ := sh.Output(
		"terraform",
		"apply",
		"-input=false",
		"-no-color",
		"-lock-timeout=120s",
		"-auto-approve=true",
		varFile,
		config.Refresh,
	)
	fmt.Println(out)
}

func InitDev() {
	Init(DevConfig)
}

func InitDevUpgrade() {
	DevConfig.Upgrade = "-upgrade=true"
	mg.Deps(InitDev)
}

func InitProd() {
	Init(ProdConfig)
}

func InitProdUpgrade() {
	DevConfig.Upgrade = "-upgrade=true"
	mg.Deps(InitProd)
}

func PlanDev() {
	Plan(DevConfig)
}

func PlanRefreshDev() {
	DevConfig.Refresh = "-refresh-only=true"
	mg.Deps(PlanDev)
}

func PlanProd() {
	Plan(ProdConfig)
}

func PlanRefreshProd() {
	DevConfig.Refresh = "-refresh-only=true"
	mg.Deps(PlanProd)
}

func ApplyDev() {
	Apply(DevConfig)
}

func ApplyRefreshDev() {
	DevConfig.Refresh = "-refresh-only=true"
	mg.Deps(ApplyDev)
}

func ApplyProd() {
	Apply(ProdConfig)
}

func ApplyRefreshProd() {
	DevConfig.Refresh = "-refresh-only=true"
	mg.Deps(ApplyProd)
}
