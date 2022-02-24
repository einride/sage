package sggcloud

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.einride.tech/sage/sg"
)

// GHACredentials generates short-lived credentials for GCP with workload identity federation.
func GHACredentials(ctx context.Context, wifPoolName, serviceAccount string) string {
	outputFile := filepath.Join(os.TempDir(), fmt.Sprintf("gha-creds-%s.json", strings.Split(wifPoolName, "/")[1]))
	if _, err := os.Stat(outputFile); err == nil {
		return outputFile
	}
	audienceURL := fmt.Sprintf("https://iam.googleapis.com/%s", wifPoolName)
	if err := sg.Command(ctx, "gcloud", "iam", "workload-identity-pools", "create-cred-config",
		wifPoolName,
		"--service-account", serviceAccount,
		"--output-file", outputFile,
		"--credential-source-url", fmt.Sprintf("%s&audience=%s", os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL"), audienceURL),
		"--credential-source-headers", fmt.Sprintf("Authorization=Bearer %s", os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")),
		"--credential-source-type", "json",
		"--credential-source-field-name", "value",
	).Run(); err != nil {
		panic(err)
	}
	return outputFile
}
