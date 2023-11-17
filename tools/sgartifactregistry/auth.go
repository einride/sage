package sgartifactregistry

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/tools/sggcloud"
)

// NpmAuthenticate authenticates to the Artifact Registry in the give registryURL.
// If yarn major version is 1 we add authentication information for the given registryURL
// in .npmrc in the user directory otherwise add to .yarnrc.yaml in the user directory.
// The npm and yarn commands are run from the give package.json directory in order to
// use the proper yarn version.
func NpmAuthenticate(ctx context.Context, packageJSONDir, registryURL string) error {
	// Grab Google Cloud Access Token
	var accessTokenOutput strings.Builder
	cmd := sggcloud.Command(ctx, "auth", "print-access-token")
	cmd.Stdout = &accessTokenOutput
	if err := cmd.Run(); err != nil {
		return err
	}
	registry := strings.TrimPrefix(registryURL, "https://")

	// If we have yarn installed, find its version
	yarnMajor := "1"
	_, err := exec.LookPath("yarn")
	if err != nil {
		if !errors.Is(err, exec.ErrNotFound) {
			return err
		}
	} else {
		// Find yarn version
		cmd = sg.Command(
			ctx,
			"yarn",
			"--version",
		)
		cmd.Dir = packageJSONDir
		version := sg.Output(cmd)
		yarnMajor = strings.Split(version, ".")[0]
	}

	switch {
	// If yarn v1 or npm we use npm config
	case yarnMajor == "1":
		cmd = sg.Command(
			ctx,
			"npm",
			"config",
			"set",
			"-L",
			"user",
			fmt.Sprintf("//%s/:_authToken", registry),
			strings.TrimSpace(accessTokenOutput.String()),
		)
		cmd.Dir = packageJSONDir
		if err := cmd.Run(); err != nil {
			return err
		}
	default:
		// If yarn v2 or higher, use yarn
		cmd = sg.Command(
			ctx,
			"yarn",
			"config",
			"set",
			"--home",
			fmt.Sprintf(`npmRegistries["//%s"].npmAuthToken`, registry),
			strings.TrimSpace(accessTokenOutput.String()),
		)
		cmd.Dir = packageJSONDir
		return cmd.Run()
	}

	return nil
}
