package sgcloudendpoints

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/tools/sgbuf"
	"go.einride.tech/sage/tools/sggcloud"
	"go.einride.tech/sage/tools/sggooglecloudprotoscrubber"
)

type ValidateOptions struct {
	// ProjectID is the ID of the GCP project to validate in.
	ProjectID string
	// BufModulePath is the path to the Buf module to validate.
	BufModulePath string
	// EndpointsConfigPath is the path to the endpoints config to validate.
	// Deprecated: Use EndpointsConfigPaths instead for multiple config files.
	EndpointsConfigPath string
	// EndpointsConfigPaths is the list of endpoints config files to validate.
	// This can include gRPC service configs (google.api.Service YAML) and/or OpenAPI specs.
	// If set, this takes precedence over EndpointsConfigPath.
	EndpointsConfigPaths []string
}

// Validate a Cloud Endpoints deployment from a Buf proto module and an endpoints config.
func Validate(ctx context.Context, opts ValidateOptions) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("validate Cloud Endpoints API gateway: %w", err)
		}
	}()
	rootDir := sg.FromBuildDir("sgcloudendpoints")
	if err := os.MkdirAll(rootDir, 0o700); err != nil {
		return err
	}
	tempDir, err := os.MkdirTemp(rootDir, "deploy")
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
	cmd = sggcloud.Command(
		ctx,
		append(
			[]string{
				"endpoints",
				"services",
				"deploy",
				"--project",
				opts.ProjectID,
				"--validate-only",
			},
			configFiles...,
		)...,
	)
	var stderr strings.Builder
	cmd.Stdout, cmd.Stderr = nil, &stderr // suppress noise
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", stderr.String(), err)
	}
	return nil
}
