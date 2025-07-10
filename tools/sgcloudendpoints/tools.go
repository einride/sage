package sgcloudendpoints

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/tools/sgbuf"
	"go.einride.tech/sage/tools/sgdocker"
	"go.einride.tech/sage/tools/sggcloud"
	"go.einride.tech/sage/tools/sggooglecloudprotoscrubber"
)

// espVersion is the version of ESPv2 used for building images.
const espVersion = "2.53.0"

//go:embed Dockerfile
var dockerfile []byte

// BuildProtoDescriptor builds a Cloud Endpoints-compatible proto descriptor from the provided Buf module input dir.
func BuildProtoDescriptor(ctx context.Context, inputDir, outputFile string) error {
	sg.Logger(ctx).Printf("building descriptor...")
	if err := os.MkdirAll(filepath.Dir(outputFile), 0o700); err != nil {
		return err
	}
	cmd := sgbuf.Command(ctx, "build", "--exclude-source-info", "-o", outputFile)
	cmd.Dir = inputDir
	if err := cmd.Run(); err != nil {
		return err
	}
	return sggooglecloudprotoscrubber.Command(ctx, "-f", outputFile).Run()
}

// DeployConfig deploys the provided config files to Cloud Endpoints and returns the resulting config revision.
func DeployConfig(ctx context.Context, project string, configFiles ...string) (string, error) {
	sg.Logger(ctx).Printf("deploying config...")
	cmd := sggcloud.Command(
		ctx,
		append(
			[]string{
				"endpoints",
				"services",
				"deploy",
				"--format",
				"value(serviceConfig.id)",
				"--project",
				project,
			},
			configFiles...,
		)...,
	)
	var output strings.Builder
	cmd.Stdout = &output
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(output.String()), nil
}

// ValidateConfig validates the provided config files with Cloud Endpoints.
func ValidateConfig(ctx context.Context, project string, configFiles ...string) error {
	sg.Logger(ctx).Printf("validating config...")
	cmd := sggcloud.Command(
		ctx,
		append(
			[]string{
				"endpoints",
				"services",
				"deploy",
				"--project",
				project,
				"--validate-only",
			},
			configFiles...,
		)...,
	)
	cmd.Stdout, cmd.Stderr = nil, nil // suppress noise
	return cmd.Run()
}

// BuildImage builds a container image with a baked-in Cloud Endpoints service configuration.
func BuildImage(ctx context.Context, project, region, repo, service, configID string) (string, error) {
	sg.Logger(ctx).Printf("building image...")
	config, err := GetServiceConfig(ctx, service, configID)
	if err != nil {
		return "", err
	}
	buildRoot := sg.FromBuildDir("cloud-endpoints")
	if err := os.MkdirAll(buildRoot, 0o700); err != nil {
		return "", err
	}
	buildDir, err := os.MkdirTemp(buildRoot, "docker")
	if err != nil {
		return "", err
	}
	defer func() {
		_ = os.RemoveAll(buildDir)
	}()
	if err := os.WriteFile(filepath.Join(buildDir, "Dockerfile"), dockerfile, 0o600); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(buildDir, "service.json"), []byte(config), 0o600); err != nil {
		return "", err
	}
	image := fmt.Sprintf(
		"%s-docker.pkg.dev/%s/%s/endpoints-runtime-serverless:%s-%s-%s",
		region,
		project,
		repo,
		espVersion,
		service,
		configID,
	)
	if err := sgdocker.Command(
		ctx, "build", "-t", image, "--build-arg", "esp_version="+espVersion, buildDir,
	).Run(); err != nil {
		return "", err
	}
	return image, nil
}

// GetServiceConfig fetches a full service config from Cloud Endpoints.
func GetServiceConfig(ctx context.Context, service, configID string) (string, error) {
	var accessTokenOutput strings.Builder
	cmd := sggcloud.Command(ctx, "auth", "print-access-token")
	cmd.Stdout = &accessTokenOutput
	if err := cmd.Run(); err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf(
			"https://servicemanagement.googleapis.com/v1/services/%s/configs/%s?view=FULL",
			service,
			configID,
		),
		nil,
	)
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessTokenOutput.String()))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = response.Body.Close()
	}()
	result, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(result)), nil
}
