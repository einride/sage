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

func Setup(config TfConfig) error {
	if err := validate(config); err != nil {
		return err
	}
	tfConfig = config
	return nil
}

func validate(config TfConfig) error {
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return err
	}
	return nil
}

func Init() error {
	args := []string{
		"init",
		"-input=false",
		"-reconfigure",
		"-backend-config=bucket=" + tfConfig.StateBucket,
		"-backend-config=impersonate_service_account=" + tfConfig.ServiceAccount,
	}
	if err := runTf(args); err != nil {
		return err
	}
	return nil
}

func Plan() error {
	args := []string{
		"plan",
		"-input=false",
		"-no-color",
		"-lock-timeout=120s",
		"-out=terraform.plan",
		"-var-file=" + tfConfig.VarFile,
	}
	if err := runTf(args); err != nil {
		return err
	}
	return nil
}

func Apply() error {
	args := []string{
		"apply",
		"-input=false",
		"-no-color",
		"-lock-timeout=120s",
		"-auto-approve=true",
		"-var-file=" + tfConfig.VarFile,
	}
	if err := runTf(args); err != nil {
		return err
	}
	return nil
}

func Fmt() error {
	args := []string{
		"fmt",
		"--recursive",
	}
	if err := runTf(args); err != nil {
		return err
	}
	return nil
}

func FmtCheck() error {
	args := []string{
		"fmt",
		"--recursive",
		"--check",
	}
	if err := runTf(args); err != nil {
		return err
	}
	return nil
}

func Validate() error {
	args := []string{"validate"}
	if err := runTf(args); err != nil {
		return err
	}
	return nil
}

func runTf(args []string) error {
	mg.Deps(mg.F(tools.Terraform, tfVersion))
	fmt.Println("[terraform] running terraform...")
	err := sh.RunV(tools.TerraformPath, args...)
	if err != nil {
		return err
	}
	return nil
}
