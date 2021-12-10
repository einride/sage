package terraform

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/einride/mage-tools/file"
	"github.com/einride/mage-tools/tools"
	"github.com/go-playground/validator/v10"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var (
	tfConfig  TfConfig
	tfVersion string
	Binary    string
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
	mg.Deps(mg.F(terraform, tfVersion))
	fmt.Println("[terraform] running terraform...")
	return sh.RunV(Binary, args...)
}

func terraform(version string) error {
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

	binDir := filepath.Join(tools.Path, binaryName, version)
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

	if err := file.FromRemote(
		binURL,
		file.WithName(filepath.Base(binary)),
		file.WithDestinationDir(binDir),
		file.WithUnzip(),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	return nil
}
