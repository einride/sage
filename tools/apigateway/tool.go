package apigateway

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/go-playground/validator/v10"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/targets/googlecloudprotoscrubber"
	"go.einride.tech/mage-tools/targets/mgbuf"
)

var gatewayConfig GatewayConfig

type GatewayConfig struct {
	ServiceAccount string `validate:"required,email"`
	ProtoTag       string `validate:"required"`
	ProtoRepo      string `validate:"required"`
	GenPath        string `validate:"required"`
	APIPb          string `validate:"required"`
	APIScrubbedPb  string `validate:"required"`
	APIConfigPath  string `validate:"required"`
	Region         string `validate:"required"`
	ServiceName    string `validate:"required"`
	GcpProject     string `validate:"required"`
}

func Setup(config GatewayConfig) error {
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return err
	}
	gatewayConfig = config
	return nil
}

func protoTagFile() error {
	fmt.Println("[proto-tag-file] touching tag file for einride/proto...")
	protoFile := filepath.Join(gatewayConfig.GenPath, "proto_tag."+gatewayConfig.ProtoTag)
	if _, err := os.Stat(protoFile); err == nil {
		return nil
	}
	err := os.MkdirAll(filepath.Dir(protoFile), 0o755)
	if err != nil {
		return err
	}
	return os.WriteFile(protoFile, []byte{}, 0o644)
}

func genAPI(ctx context.Context) error {
	mg.Deps(protoTagFile)
	fmt.Printf("[gen-api] generating API descriptor from %s...", gatewayConfig.ProtoRepo)
	return mgbuf.Buf(
		ctx,
		"build",
		fmt.Sprintf("%s#tag=%s", gatewayConfig.ProtoRepo, gatewayConfig.ProtoTag),
		"--as-tool-descriptor-set",
		"-o", gatewayConfig.APIPb,
	)
}

func genAPIScrubbed(ctx context.Context) error {
	mg.Deps(genAPI)
	input, err := ioutil.ReadFile(gatewayConfig.APIPb)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(gatewayConfig.APIScrubbedPb, input, 0o600)
	if err != nil {
		return err
	}
	return googlecloudprotoscrubber.GoogleCloudProtoScrubber(ctx, gatewayConfig.APIScrubbedPb)
}

func Generate() {
	mg.Deps(genAPIScrubbed)
}

func ValidateEndpoints() error {
	mg.Deps(Generate)
	fmt.Printf("[validate-endpoints] validating endpoints config in %s...", gatewayConfig.GcpProject)
	return sh.RunV(
		"gcloud",
		"endpoints",
		"services",
		"deploy",
		"--validate-only",
		"--project",
		gatewayConfig.GcpProject,
		"--impersonate-service-account",
		gatewayConfig.ServiceAccount,
		gatewayConfig.APIConfigPath,
		gatewayConfig.APIScrubbedPb,
	)
}

func DeployEndpoint() error {
	mg.Deps(Generate)
	fmt.Printf("[deploy-endpoints] deploying endpoints config to %s...", gatewayConfig.GcpProject)
	return sh.RunV(
		"gcloud",
		"endpoints",
		"services",
		"deploy",
		"--project",
		gatewayConfig.GcpProject,
		gatewayConfig.APIConfigPath,
		gatewayConfig.APIScrubbedPb,
	)
}

func endpointsConfigID() (string, error) {
	out, err := sh.Output(
		"gcloud",
		"endpoints",
		"configs",
		"list",
		"--service",
		gatewayConfig.ServiceName,
		"--project",
		gatewayConfig.GcpProject,
		"--limit",
		"1",
		"--format",
		"value(id)",
	)
	if err != nil {
		return "", err
	}
	return out, nil
}

func BuildImage() error {
	configID, err := endpointsConfigID()
	fmt.Printf(
		"[build-image] building image for %s in %s [%s] with config %s...\n",
		gatewayConfig.ServiceName,
		gatewayConfig.GcpProject,
		gatewayConfig.Region,
		configID,
	)
	if err != nil {
		return err
	}
	return sh.RunV(
		"scripts/gcloud-build-image.bash",
		"-s",
		gatewayConfig.ServiceName,
		"-p",
		gatewayConfig.GcpProject,
		"-c",
		configID,
		"-r",
		gatewayConfig.Region,
	)
}

func DeployCloudRun() error {
	fmt.Println("[deploy-cloud-run] deploying cloud run...")
	configID, err := endpointsConfigID()
	if err != nil {
		return err
	}
	return sh.RunV(
		"gcloud",
		"run",
		"deploy",
		"api-gateway",
		"--image",
		fmt.Sprintf(
			"%s-docker.pkg.dev/%s/docker/endpoints-runtime-serverless:%s-%s",
			gatewayConfig.Region,
			gatewayConfig.GcpProject,
			gatewayConfig.ServiceName,
			configID,
		),
		"--project",
		gatewayConfig.GcpProject,
		"--region",
		gatewayConfig.Region,
		"--platform",
		"managed",
		"--allow-unauthenticated",
	)
}
