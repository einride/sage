package sgcloudrun

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/tools/sggcloud"
	"go.einride.tech/sage/tools/sggit"
	"go.einride.tech/sage/tools/sgyq"
)

// Develop starts the Cloud Run service at the provided Go path with the provided service account and config.
// Deprecated: Develop uses a service account key which are inherently more risky as they are not often rotated.
// Use LocalDevelop instead.
func Develop(ctx context.Context, path, keyFile, configFile string) error {
	cmd, err := DevelopCommand(ctx, path, keyFile, configFile)
	if err != nil {
		return err
	}
	return cmd.Run()
}

// DevelopCommand returns an *exec.Cmd pre-configured to start the Cloud Run service at the provided Go path
// with the provided service account and config.
// Deprecated: DevelopCommand uses a service account key which are inherently more risky as they are not often rotated.
// Use LocalDevelopCommand instead.
func DevelopCommand(ctx context.Context, path, keyFile, configFile string) (*exec.Cmd, error) {
	var key struct {
		Type        string
		ProjectID   string `json:"project_id"`
		ClientEmail string `json:"client_email"`
	}
	keyData, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(keyData, &key); err != nil {
		return nil, err
	}
	if key.Type != "service_account" {
		return nil, fmt.Errorf("not a valid service account JSON key file: %s", keyFile)
	}
	accessToken, err := printServiceAccountAccessToken(ctx, key.ClientEmail, keyFile)
	if err != nil {
		return nil, err
	}
	env, err := resolveEnvFromConfigFile(ctx, configFile, key.ProjectID, accessToken)
	if err != nil {
		return nil, err
	}
	cmd := sg.Command(ctx, "go", "run", path)
	cmd.Env = append(cmd.Env, "K_REVISION=local"+sggit.SHA(ctx))
	cmd.Env = append(cmd.Env, "K_CONFIGURATION="+configFile)
	cmd.Env = append(cmd.Env, "GOOGLE_CLOUD_PROJECT="+key.ProjectID)
	cmd.Env = append(cmd.Env, "GOOGLE_APPLICATION_CREDENTIALS="+keyFile)
	cmd.Env = append(cmd.Env, env...)
	cmd.Env = append(cmd.Env, os.Environ()...) // allow environment overrides
	return cmd, nil
}

func LocalDevelop(ctx context.Context, path, configFile, projectID, serviceAccountEmail string) error {
	cmd, err := LocalDevelopCommand(ctx, path, configFile, projectID, serviceAccountEmail)
	if err != nil {
		return err
	}
	return cmd.Run()
}

// LocalDevelopEnv sets up the environment variables for running the Cloud Run service locally, with SA impersonation.
// The environment variables are returned on the format KEY=value and can easily be outputted to a .env file or similar.
// NOTE: this function creates a temporary creds-xxxxx.json file that is meant to be removed when the service is shut
// down. Make sure to call CleanUpLocalDevelop after shutting down the service.
func LocalDevelopEnv(
	ctx context.Context,
	configFile string,
	projectID string,
	serviceAccountEmail string,
) ([]string, error) {
	// Grab the local user token to impersonate the service account
	currentADC, err := applicationDefaultCredentials()
	if err != nil {
		return nil, err
	}

	// Store a local token wrapping the user token in metadata to make a delegated request.
	// This requires the impersonated service account to have implicitDelegation permissions on itself.
	// See https://cloud.google.com/iam/docs/create-short-lived-credentials-delegated#sa-credentials-delegated for details
	serviceAccountImpersonationURL := fmt.Sprintf(
		"https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/%s:generateAccessToken",
		serviceAccountEmail,
	)

	delegateCreds := struct {
		Delegates                      []string        `json:"delegates"`
		Type                           string          `json:"type"`
		ServiceAccountImpersonationURL string          `json:"service_account_impersonation_url"`
		SourceCredentials              json.RawMessage `json:"source_credentials"`
	}{
		Delegates:                      []string{"projects/-/serviceAccounts/" + serviceAccountEmail},
		Type:                           "impersonated_service_account",
		ServiceAccountImpersonationURL: serviceAccountImpersonationURL,
		SourceCredentials:              json.RawMessage(currentADC),
	}
	delegateCredsJSON, err := json.Marshal(delegateCreds)
	if err != nil {
		return nil, err
	}

	accessToken, err := fetchImpersonatedAccessToken(ctx, serviceAccountEmail)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch impersonated service account access token: %v", err)
	}

	env, err := resolveEnvFromConfigFile(ctx, configFile, projectID, accessToken)
	if err != nil {
		return nil, err
	}

	workDir := sg.FromBuildDir("gcloud")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return nil, fmt.Errorf("unable to create path to store gcloud credentials: %v", err)
	}
	credsPath := filepath.Join(workDir, fmt.Sprintf("creds-%s.json", randomLower(5)))
	if err := os.WriteFile(credsPath, delegateCredsJSON, 0o600); err != nil {
		return nil, err
	}

	env = append(env, "K_REVISION=local"+sggit.SHA(ctx))
	env = append(env, "K_CONFIGURATION="+configFile)
	env = append(env, "GOOGLE_CLOUD_PROJECT="+projectID)
	env = append(env, "GOOGLE_APPLICATION_CREDENTIALS="+credsPath)

	return env, nil
}

// CleanUpLocalDevelop is meant to be called after the Cloud Run service is shut down locally.
// It removes the temporary creds-xxxxx.json file.
func CleanUpLocalDevelop(environ []string) error {
	for _, env := range environ {
		if !strings.Contains(env, "GOOGLE_APPLICATION_CREDENTIALS") {
			continue
		}
		credsPath := strings.Split(env, "=")[1]
		return os.Remove(credsPath)
	}
	return fmt.Errorf("clean up local develop: no GOOGLE_APPLICATION_CREDENTIALS environment variable found")
}

func LocalDevelopCommand(
	ctx context.Context,
	path string,
	configFile string,
	projectID string,
	serviceAccountEmail string,
) (*exec.Cmd, error) {
	env, err := LocalDevelopEnv(ctx, configFile, projectID, serviceAccountEmail)
	if err != nil {
		return nil, err
	}

	cmd := sg.Command(ctx, "go", "run", path)
	cmd.Env = append(cmd.Env, env...)
	cmd.Env = append(cmd.Env, os.Environ()...) // allow environment overrides
	cmd.Cancel = func() error {
		if err := cmd.Process.Kill(); err != nil {
			return err
		}
		return CleanUpLocalDevelop(cmd.Env)
	}
	return cmd, nil
}

func resolveEnvFromConfigFile(ctx context.Context, filename, project, accessToken string) (_ []string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("resolve env from YAML service specification file %s: %w", filename, err)
		}
	}()
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	cmd := sgyq.Command(ctx, "-o", "json")
	cmd.Stdin = bytes.NewReader(data)
	var output bytes.Buffer
	cmd.Stdout = &output
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	var config struct {
		Metadata struct {
			Name string
		}
		Spec struct {
			Template struct {
				Spec struct {
					Containers []struct {
						Env []struct {
							Name      string
							Value     string
							ValueFrom struct {
								SecretKeyRef struct {
									Name string
									Key  string
								}
							}
						}
					}
				}
			}
		}
	}
	if err := json.NewDecoder(&output).Decode(&config); err != nil {
		return nil, err
	}
	if len(config.Spec.Template.Spec.Containers) != 1 {
		return nil, fmt.Errorf("unexpected number of containers: %d", len(config.Spec.Template.Spec.Containers))
	}
	result := make([]string, 0, 100)
	if config.Metadata.Name != "" {
		result = append(result, "K_SERVICE="+config.Metadata.Name)
	}
	for _, env := range config.Spec.Template.Spec.Containers[0].Env {
		if env.Value != "" {
			result = append(result, env.Name+"="+env.Value)
			continue
		}
		if env.ValueFrom.SecretKeyRef.Name != "" && env.ValueFrom.SecretKeyRef.Key != "" {
			secret, err := accessSecretVersion(
				ctx,
				accessToken,
				project,
				env.ValueFrom.SecretKeyRef.Name,
				env.ValueFrom.SecretKeyRef.Key,
			)
			if err != nil {
				return nil, err
			}
			result = append(result, env.Name+"="+secret)
		}
	}
	return result, nil
}

func printServiceAccountAccessToken(ctx context.Context, serviceAccount, keyFile string) (_ string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("print service account access token for %s: %w", serviceAccount, err)
		}
	}()
	sg.Logger(ctx).Printf("generating access token for %s...", serviceAccount)
	var getAccountOutput strings.Builder
	cmd := sggcloud.Command(ctx, "config", "get", "account")
	cmd.Stdout = &getAccountOutput
	if err := cmd.Run(); err != nil {
		return "", err
	}
	prevAccount := strings.TrimSpace(getAccountOutput.String())
	if prevAccount == "" {
		return "", fmt.Errorf("no active Google Cloud Account, did you remember to `gcloud auth login`")
	}
	cmd = sggcloud.Command(ctx, "auth", "activate-service-account", "--key-file", keyFile)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return "", err
	}
	var accessTokenOutput strings.Builder
	cmd = sggcloud.Command(ctx, "auth", "print-access-token")
	cmd.Stdout = &accessTokenOutput
	if err := cmd.Run(); err != nil {
		return "", err
	}
	accessToken := strings.TrimSpace(accessTokenOutput.String())
	if accessToken == "" {
		return "", fmt.Errorf("got empty access token")
	}
	cmd = sggcloud.Command(ctx, "auth", "revoke", serviceAccount)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return "", err
	}
	cmd = sggcloud.Command(ctx, "config", "set", "account", prevAccount)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return accessToken, nil
}

func accessSecretVersion(ctx context.Context, accessToken, project, secret, version string) (string, error) {
	secretName := fmt.Sprintf("projects/%s/secrets/%s/versions/%s", project, secret, version)
	sg.Logger(ctx).Printf("accessing secret %s...", secretName)
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("https://secretmanager.googleapis.com/v1/%s:access", secretName),
		nil,
	)
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = response.Body.Close()
	}()
	var responseBody struct {
		Payload struct {
			Data string
		}
	}
	if err := json.NewDecoder(response.Body).Decode(&responseBody); err != nil {
		return "", err
	}
	if responseBody.Payload.Data == "" {
		return "", fmt.Errorf("no value for secret %s", secretName)
	}
	decodedData, err := base64.URLEncoding.DecodeString(responseBody.Payload.Data)
	if err != nil {
		return "", err
	}
	return string(decodedData), nil
}

func applicationDefaultCredentials() (string, error) {
	home := os.Getenv("HOME")
	path := filepath.Join(home, ".config/gcloud/application_default_credentials.json")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.New(
				"no application default credentials found. Please authenticate using 'gcloud auth application-default login'",
			)
		}
		return "", fmt.Errorf("unable to read application default credentials at %s - %v", path, err)
	}
	return string(b), nil
}

func fetchImpersonatedAccessToken(ctx context.Context, serviceAccountEmail string) (string, error) {
	// Fetch user access token
	var accessTokenOutput strings.Builder
	cmd := sggcloud.Command(ctx, "auth", "print-access-token")
	cmd.Stdout = &accessTokenOutput
	if err := cmd.Run(); err != nil {
		return "", err
	}
	accessToken := strings.TrimSpace(accessTokenOutput.String())
	if accessToken == "" {
		return "", fmt.Errorf("got empty access token")
	}

	// Generate access token for delegated service account
	body := struct {
		Delegates []string `json:"delegates"`
		Scope     []string `json:"scope"`
	}{
		Delegates: []string{"projects/-/serviceAccounts/" + serviceAccountEmail},
		Scope:     []string{"https://www.googleapis.com/auth/cloud-platform"},
	}
	reqBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("unable to json marshal access token request body: %v", err)
	}

	serviceAccountImpersonationURL := fmt.Sprintf(
		"https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/%s:generateAccessToken",
		serviceAccountEmail,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, serviceAccountImpersonationURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to generate access token for service account: %s", string(b))
	}

	var tokens struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return "", err
	}

	return tokens.AccessToken, nil
}

func randomLower(n uint32) string {
	b := make([]rune, n)
	for i := range b {
		// NOTE: code of 'a' is 97, code of 'z' is 122.
		//       so rand.IntN(26) + 97 is code of a "random" lowercase rune.
		//nolint:gosec // we don't need a secure randomizer for this.
		b[i] = rune(rand.IntN(26) + 97)
	}
	return string(b)
}
