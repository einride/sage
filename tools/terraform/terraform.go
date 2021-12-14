package terraform

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/go-playground/validator/v10"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mgtool"
	"go.einride.tech/mage-tools/tools"
)

var (
	tfConfig  TfConfig
	tfVersion string
	Binary    string
)

type TfConfig struct {
	ServiceAccount string `validate:"omitempty,email"`
	StateBucket    string `validate:"required"`
	VarFile        string `validate:"required"`
}

func SetTerraformVersion(v string) (string, error) {
	tfVersion = v
	return tfVersion, nil
}

func Setup(config TfConfig) error {
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return err
	}
	tfConfig = config
	return nil
}

func Init() error {
	args := []string{
		"init",
		"-input=false",
		"-reconfigure",
		"-backend-config=bucket=" + tfConfig.StateBucket,
	}
	if tfConfig.ServiceAccount != "" {
		args = append(args, "-backend-config=impersonate_service_account="+tfConfig.ServiceAccount)
	}
	return runTf(args)
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
	return runTf(args)
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
	return runTf(args)
}

func Fmt() error {
	args := []string{
		"fmt",
		"--recursive",
	}
	return runTf(args)
}

func FmtCheck() error {
	args := []string{
		"fmt",
		"--recursive",
		"--check",
	}
	return runTf(args)
}

func Validate() error {
	args := []string{"validate"}
	return runTf(args)
}

func runTf(args []string) error {
	mg.Deps(mg.F(terraform, tfVersion))
	fmt.Println("[terraform] running terraform...")
	return sh.RunV(Binary, args...)
}

func terraform(ctx context.Context, version string) error {
	const binaryName = "terraform"
	const defaultVersion = "1.0.0"

	if version == "" {
		version = defaultVersion
	} else {
		supportedVersions := []string{
			"1.0.0",
			"1.0.5",
		}
		if err := tools.IsSupportedVersion(supportedVersions, version, binaryName); err != nil {
			return err
		}
	}

	binDir := filepath.Join(tools.GetPath(), binaryName, version)
	binary := filepath.Join(binDir, binaryName)
	Binary = binary

	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH

	binURL := fmt.Sprintf(
		"https://releases.hashicorp.com/terraform/%s/terraform_%s_%s_%s.zip",
		version,
		version,
		hostOS,
		hostArch,
	)

	if err := mgtool.FromRemote(
		ctx,
		binURL,
		mgtool.WithDestinationDir(binDir),
		mgtool.WithUnzip(),
		mgtool.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	return nil
}
