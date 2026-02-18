package sgartifactregistry

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
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
	// Trailing slashes at the end of the URL have been known to cause issues with some setups
	registry = strings.TrimSuffix(registry, "/")

	expired, err := isGoogleAuthExpired(ctx)
	if err != nil {
		return fmt.Errorf("unable to verify Google authentication token expiration: %v", err)
	}
	if expired {
		return fmt.Errorf("google authentication token is expired. Please authenticate using 'gcloud auth login'")
	}

	// If we have yarn installed, find its version
	yarnMajor := "1"
	_, err = exec.LookPath("yarn")
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

	switch yarnMajor {
	// If yarn v1 or npm we use npm config
	case "1":
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

// isGoogleAuthExpired uses gcloud to determine whether the user has an expired token or not.
// false (along with a warning log) is returned if the gcloud auth describe command does not return data
// in the format we are expecting.
func isGoogleAuthExpired(ctx context.Context) (bool, error) {
	var strExpiry string
	account := sg.Output(sg.Command(ctx, "gcloud", "config", "get-value", "account"))
	// gcloud auth describe is an undocumented gcloud API which allows us to get back
	// information about the currently authenticated user including the token expiration time.
	authInfo := sg.Output(sg.Command(ctx, "gcloud", "auth", "describe", account))
	lines := strings.SplitSeq(authInfo, "\n")
	for line := range lines {
		if !strings.HasPrefix(line, "expired: ") {
			continue
		}
		strExpiry = strings.TrimPrefix(line, "expired: ")
	}
	// Because the atuth describe command is not documented and could potentially changes,
	// if we are unable to parse its output we will log a warning and return false.
	if strExpiry == "" {
		sg.Logger(ctx).Println("WARNING: unable to determine expiration time of Google authentication token")
		return false, nil
	}
	authExpired, err := strconv.ParseBool(strExpiry)
	if err != nil {
		return false, fmt.Errorf("unable to parse expiration time of Google authentication token: %v", err)
	}
	return authExpired, nil
}
