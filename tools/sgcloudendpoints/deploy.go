package sgcloudendpoints

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/tools/sgbuf"
	"go.einride.tech/sage/tools/sgdocker"
	"go.einride.tech/sage/tools/sggcloud"
	"go.einride.tech/sage/tools/sggooglecloudprotoscrubber"
)

type DeployOptions struct {
	// ProjectID is the ID of the GCP project to deploy to.
	ProjectID string
	// Region is the GCP region to deploy to.
	Region string
	// ArtifactRegistry is the name of the artifact registry to push the deployed image to.
	ArtifactRegistry string
	// BufModulePath is the path to the Buf module to deploy.
	BufModulePath string
	// EndpointsConfigPath is the path to the endpoints config to deploy.
	// Deprecated: Use EndpointsConfigPaths instead for multiple config files.
	EndpointsConfigPath string
	// EndpointsConfigPaths is the list of endpoints config files to deploy.
	// This can include gRPC service configs (google.api.Service YAML) and/or OpenAPI specs.
	// The first config file containing a "name:" field will be used to determine the service name.
	// If set, this takes precedence over EndpointsConfigPath.
	EndpointsConfigPaths []string
	// ServiceConfigPath is the path to the Knative YAML service config to deploy.
	//
	// The service config will be executed as a Go template, where the following variables are available:
	//
	//  - Image: The container image to deploy
	ServiceConfigPath string
}

func Deploy(ctx context.Context, opts DeployOptions) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("validate Cloud Endpoints deployment: %w", err)
		}
	}()
	rootDir := sg.FromBuildDir("sgcloudendpoints")
	if err := os.MkdirAll(rootDir, 0o700); err != nil {
		return err
	}
	tempDir, err := os.MkdirTemp(rootDir, "validate")
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			_ = os.RemoveAll(tempDir)
		}
	}()
	protoDescriptorPath := filepath.Join(tempDir, "descriptor.pb")
	cmd := sgbuf.Command(ctx, "build", "--exclude-source-info", "-o", protoDescriptorPath)
	cmd.Dir = opts.BufModulePath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build protobuf descriptor: %w", err)
	}
	if err := sggooglecloudprotoscrubber.Command(ctx, "-f", protoDescriptorPath).Run(); err != nil {
		return fmt.Errorf("scrub protobuf descriptor: %w", err)
	}
	// Support both single path (legacy) and multiple paths
	endpointsConfigs := opts.EndpointsConfigPaths
	if len(endpointsConfigs) == 0 && opts.EndpointsConfigPath != "" {
		endpointsConfigs = []string{opts.EndpointsConfigPath}
	}
	configFiles := append([]string{protoDescriptorPath}, endpointsConfigs...)
	configID, err := DeployConfig(ctx, opts.ProjectID, configFiles...)
	if err != nil {
		return err
	}
	serviceName, err := extractServiceName(endpointsConfigs)
	if err != nil {
		return err
	}
	image, err := BuildImage(ctx, opts.ProjectID, opts.Region, opts.ArtifactRegistry, serviceName, configID)
	if err != nil {
		return err
	}
	if err := sgdocker.Command(ctx, "push", image).Run(); err != nil {
		return fmt.Errorf("push container image: %w", err)
	}
	serviceConfigTemplateData, err := os.ReadFile(opts.ServiceConfigPath)
	if err != nil {
		return fmt.Errorf("read service config: %w", err)
	}
	serviceConfigTemplate, err := template.New("config.yaml").Parse(string(serviceConfigTemplateData))
	if err != nil {
		return fmt.Errorf("parse service config template: %w", err)
	}
	var serviceConfig bytes.Buffer
	if err := serviceConfigTemplate.Execute(&serviceConfig, struct{ Image string }{Image: image}); err != nil {
		return fmt.Errorf("execute config template: %w", err)
	}
	cmd = sggcloud.Command(
		ctx,
		"run",
		"services",
		"replace",
		"--project",
		opts.ProjectID,
		"--region",
		opts.Region,
		"--platform",
		"managed",
		"-",
	)
	cmd.Stdin = &serviceConfig
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("replace API gateway service: %w", err)
	}
	return nil
}

// extractServiceName extracts the service name from the first config file that contains a "name:" field.
func extractServiceName(configPaths []string) (string, error) {
	nameRegex := regexp.MustCompile(`name:\s*(.+)`)
	for _, configPath := range configPaths {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return "", fmt.Errorf("read config %s: %w", configPath, err)
		}
		matches := nameRegex.FindStringSubmatch(string(data))
		if len(matches) >= 2 {
			return strings.TrimSpace(matches[1]), nil
		}
	}
	return "", fmt.Errorf("service name not found in any config file: %v", configPaths)
}
