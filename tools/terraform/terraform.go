package terraform

import (
	"fmt"

	"github.com/einride/mage-tools/tools"
	"github.com/go-playground/validator/v10"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var (
	tfConfig  TfConfig
	tfVersion string
)

type TfConfig struct {
	ServiceAccount string `validate:"required,email"`
	StateBucket    string `validate:"required"`
	VarFile        string `validate:"required"`
}

func SetTerraformVersion(v string) (string, error) {
	tfVersion = v
	return tfVersion, nil
}

func Setup(config TfConfig) {
	validate(config)
	tfConfig = config
}

func validate(config TfConfig) {
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		panic(err)
	}
}

func Init() {
	args := []string{
		"init",
		"-input=false",
		"-reconfigure",
		"-backend-config=bucket=" + tfConfig.StateBucket,
		"-backend-config=impersonate_service_account=" + tfConfig.ServiceAccount,
	}
	runTf(args)
}

func Plan() {
	args := []string{
		"plan",
		"-input=false",
		"-no-color",
		"-lock-timeout=120s",
		"-out=terraform.plan",
		"-var-file=" + tfConfig.VarFile,
	}
	runTf(args)
}

func Apply() {
	args := []string{
		"apply",
		"-input=false",
		"-no-color",
		"-lock-timeout=120s",
		"-auto-approve=true",
		"-var-file=" + tfConfig.VarFile,
	}
	runTf(args)
}

func Fmt() {
	args := []string{
		"fmt",
		"--recursive",
	}
	runTf(args)
}

func FmtCheck() {
	args := []string{
		"fmt",
		"--recursive",
		"--check",
	}
	runTf(args)
}

func Validate() {
	args := []string{"validate"}
	runTf(args)
}

func runTf(args []string) {
	mg.Deps(mg.F(tools.Terraform, tfVersion))
	fmt.Println("[terraform] running terraform...")
	err := sh.RunV(tools.TerraformPath, args...)
	if err != nil {
		panic(err)
	}
}
