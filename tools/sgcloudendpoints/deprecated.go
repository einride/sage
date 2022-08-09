package sgcloudendpoints

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/tools/sgdocker"
	"go.einride.tech/sage/tools/sggcloud"
)

// ConfigID returns the ID of the latest config for the provided Cloud Endpoints service.
// Deprecated: Use sgcloudendpoints.DeployConfig instead.
func ConfigID(ctx context.Context, serviceName, gcpProject string) string {
	sg.Logger(ctx).Printf(
		"retrieving endpoints configID from %s in %s...",
		serviceName,
		gcpProject,
	)
	return sg.Output(sggcloud.Command(
		ctx,
		"endpoints",
		"configs",
		"list",
		"--service",
		serviceName,
		"--project",
		gcpProject,
		"--limit",
		"1",
		"--format",
		"value(id)",
	))
}

// DockerImage builds a Cloud Endpoints Docker image.
// Deprecated: Use sgcloudendpoints.DeployConfig and sgcloudendpoints.BuildImage instead.
func DockerImage(ctx context.Context, serviceName, gcpProject, gcpRegion string) string {
	configID := ConfigID(ctx, serviceName, gcpProject)
	sg.Logger(ctx).Printf(
		"building image for %s with configID %s...",
		serviceName,
		configID,
	)
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf(
			"https://servicemanagement.googleapis.com/v1/services/%s/configs/%s?view=FULL",
			serviceName,
			configID,
		),
		nil,
	)
	if err != nil {
		panic(err)
	}
	token := sg.Output(sggcloud.Command(ctx, "auth", "print-access-token"))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	dir, err := os.MkdirTemp(os.TempDir(), "sgcloudendpoints")
	if err != nil {
		panic(err)
	}
	f, err := os.Create(filepath.Join(dir, "service.json"))
	if err != nil {
		panic(err)
	}
	if _, err := f.ReadFrom(resp.Body); err != nil {
		panic(err)
	}
	defer func() {
		_ = f.Close()
	}()
	tag := fmt.Sprintf(
		"%s-docker.pkg.dev/%s/docker/endpoints-runtime-serverless:%s-%s",
		gcpRegion,
		gcpProject,
		serviceName,
		configID,
	)
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), dockerfile, 0o600); err != nil {
		panic(err)
	}
	cmd := sgdocker.Command(ctx, "build", "-t", tag, ".")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	return tag
}
