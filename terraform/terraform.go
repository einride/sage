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
	Upgrade        bool
	Refresh        bool
	VarFile        string
}

func SetupDev(config TfConfig) error {
	DevConfig = config
	return nil
}

func SetupProd(config TfConfig) error {
	ProdConfig = config
	return nil
}

func Init(config TfConfig) {
	mg.Deps(tools.Terraform)
	args := []string{
		"init",
		"-input=false",
		"-reconfigure",
		"-backend-config=bucket=" + config.StateBucket,
		"-backend-config=impersonate_service_account=" + config.ServiceAccount,
	}
	if config.Upgrade {
		args = append(args, "-upgrade=true")
	}
	fmt.Println("Initing...")
	out, _ := sh.Output(
		"terraform",
		args...,
	)
	fmt.Println(out)
}

func Plan(config TfConfig) {
	mg.Deps(tools.Terraform)
	fmt.Println("Planning...")
	args := []string{
		"plan",
		"-input=false",
		"-no-color",
		"-lock-timeout=120s",
		"-out=terraform.plan",
		"-var-file=" + config.VarFile,
	}
	if config.Refresh {
		args = append(args, "-refresh-only=true")
	}
	out, _ := sh.Output(
		"terraform",
		args...,
	)
	fmt.Println(out)
}

func Apply(config TfConfig) {
	mg.Deps(tools.Terraform)
	fmt.Println("Applying...")
	args := []string{
		"apply",
		"-input=false",
		"-no-color",
		"-lock-timeout=120s",
		"-auto-approve=true",
		"-var-file=" + config.VarFile,
	}
	if config.Refresh {
		args = append(args, "-refresh-only=true")
	}
	out, _ := sh.Output(
		"terraform",
		args...,
	)
	fmt.Println(out)
}

func InitDev() {
	Init(DevConfig)
}

func InitDevUpgrade() {
	DevConfig.Upgrade = true
	mg.Deps(InitDev)
}

func InitProd() {
	Init(ProdConfig)
}

func InitProdUpgrade() {
	DevConfig.Upgrade = true
	mg.Deps(InitProd)
}

func PlanDev() {
	Plan(DevConfig)
}

func PlanRefreshDev() {
	DevConfig.Refresh = true
	mg.Deps(PlanDev)
}

func PlanProd() {
	Plan(ProdConfig)
}

func PlanRefreshProd() {
	DevConfig.Refresh = true
	mg.Deps(PlanProd)
}

func ApplyDev() {
	Apply(DevConfig)
}

func ApplyRefreshDev() {
	DevConfig.Refresh = true
	mg.Deps(ApplyDev)
}

func ApplyProd() {
	Apply(ProdConfig)
}

func ApplyRefreshProd() {
	DevConfig.Refresh = true
	mg.Deps(ApplyProd)
}
