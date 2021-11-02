package terraform

import (
	"github.com/einride/mage-tools/tools"
	"github.com/go-playground/validator/v10"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var devConfig TfConfig
var prodConfig TfConfig
var tfVersion string

type TfConfig struct {
	ServiceAccount string `validate:"required,email"`
	StateBucket    string `validate:"required"`
	Upgrade        bool
	Refresh        bool
	VarFile        string `validate:"required"`
}

func SetVersion(v string) (error, string) {
	tfVersion = v
	return nil, tfVersion
}

func SetupDev(config TfConfig) {
	validate(config)
	devConfig = config
}

func SetupProd(config TfConfig) {
	validate(config)
	prodConfig = config
}

func validate(config TfConfig) {
	validate := validator.New()
	err := validate.Struct(config)
	if err != nil {
		panic(err)
	}
}

func Init(config TfConfig) {
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
	runTf(args)
}

func Plan(config TfConfig) {
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
	runTf(args)
}

func Apply(config TfConfig) {
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
	runTf(args)
}

func InitDev() {
	Init(devConfig)
}

func InitDevUpgrade() {
	devConfig.Upgrade = true
	mg.Deps(InitDev)
}

func InitProd() {
	Init(prodConfig)
}

func InitProdUpgrade() {
	devConfig.Upgrade = true
	mg.Deps(InitProd)
}

func PlanDev() {
	Plan(devConfig)
}

func PlanRefreshDev() {
	devConfig.Refresh = true
	mg.Deps(PlanDev)
}

func PlanProd() {
	Plan(prodConfig)
}

func PlanRefreshProd() {
	devConfig.Refresh = true
	mg.Deps(PlanProd)
}

func ApplyDev() {
	Apply(devConfig)
}

func ApplyRefreshDev() {
	devConfig.Refresh = true
	mg.Deps(ApplyDev)
}

func ApplyProd() {
	Apply(prodConfig)
}

func ApplyRefreshProd() {
	devConfig.Refresh = true
	mg.Deps(ApplyProd)
}

func Sops(file string) {
	mg.Deps(tools.Sops)
	err := sh.RunV("sops", file)
	if err != nil {
		panic(err)
	}
}

func runTf(args []string) {
	mg.Deps(mg.F(tools.Terraform, tfVersion))
	err := sh.RunV("terraform", args...)
	if err != nil {
		panic(err)
	}
}
