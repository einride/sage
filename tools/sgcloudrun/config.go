package sgcloudrun

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/tools/sggcloud"
	"go.einride.tech/sage/tools/sggit"
	"go.einride.tech/sage/tools/sgyq"
)

// Develop starts the Cloud Run service at the provided Go path with the provided service account and config.
//
// Deprecated: Develop uses a service account key which are inherently more risky as they are not often rotated,
// use LocalDevelop instead.
func Develop(ctx context.Context, path, keyFile, configFile string) error {
	cmd, err := DevelopCommand(ctx, path, keyFile, configFile)
	if err != nil {
		return err
	}
	return cmd.Run()
}

// DevelopCommand returns an *exec.Cmd pre-configured to start the Cloud Run service at the provided Go path
// with the provided service account and config.
//
// Deprecated: DevelopCommand uses a service account key which are inherently more risky as they are not often rotated,
// use LocalDevelopCommand instead.
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
//
// Deprecated: There was no reason to export this function and it may get removed in the future.
func LocalDevelopEnv(
	ctx context.Context,
	configFile string,
	projectID string,
	serviceAccountEmail string,
) ([]string, error) {
	creds, err := resolveCredentials()
	if err != nil {
		return nil, err
	}
	delegates := impersonationDelegates(creds.credentialType, serviceAccountEmail)

	accessToken, err := fetchImpersonatedAccessToken(ctx, creds, serviceAccountEmail, delegates)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch impersonated service account access token: %v", err)
	}

	env, err := resolveEnvFromConfigFile(ctx, configFile, projectID, accessToken)
	if err != nil {
		return nil, err
	}

	// The temporary credentials file is removed by CleanUpLocalDevelop, so the cleanup func is not needed here.
	credsPath, _, err := delegatedGoogleApplicationCredentials(creds, serviceAccountEmail, delegates)
	if err != nil {
		return nil, err
	}

	env = append(env, "K_REVISION=local"+sggit.SHA(ctx))
	env = append(env, "K_CONFIGURATION="+configFile)
	env = append(env, "GOOGLE_CLOUD_PROJECT="+projectID)
	env = append(env, googleApplicationCredentialsEnvVar+"="+credsPath)

	return env, nil
}

// CleanUpLocalDevelop is meant to be called after the Cloud Run service is shut down locally.
// It removes the temporary creds-xxxxx.json file.
//
// Deprecated: There was no reason to export this function and it may get removed in the future.
func CleanUpLocalDevelop(environ []string) error {
	for _, env := range environ {
		if !strings.Contains(env, googleApplicationCredentialsEnvVar) {
			continue
		}
		credsPath := strings.Split(env, "=")[1]
		return os.Remove(credsPath)
	}
	return fmt.Errorf("clean up local develop: no %s environment variable found", googleApplicationCredentialsEnvVar)
}

// LocalDevelopCommand returns an *exec.Cmd pre-configured to start the Cloud Run service at the provided Go path.
// Any environment variables configured in spec.template.spec.containers[0].env are exposed as environment variables.
// Secrets referred by through through spec.template.spec.containers[0].env.valueFrom.secretKeyRef are first read from
// secret manager..
// If serviceAccountEmail is not empty, an attempt to generate impersonated short-lived credentials for that
// service account is done. Otherwise the underlying credentials are used as-is.
//
// Impersonation is based on the active Application Default Credentials, which are resolved from
// GOOGLE_APPLICATION_CREDENTIALS, then CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE, then gcloud's well-known path.
// The environment variables take precedence so that CI setups pointing at a credential configuration file, such
// as google-github-actions/auth, are picked up. The resolved identity needs
// roles/iam.serviceAccountTokenCreator on serviceAccountEmail. A local user credential is additionally
// self-delegated through serviceAccountEmail, which requires that service account to hold the same role on
// itself; service account and external account credentials use plain impersonation and do not.
func LocalDevelopCommand(
	ctx context.Context,
	path string,
	configFile string,
	projectID string,
	serviceAccountEmail string,
) (*exec.Cmd, error) {
	var (
		accessToken string
		cleanup     = func() error { return nil }
		credsPath   string
	)
	if serviceAccountEmail != "" {
		creds, err := resolveCredentials()
		if err != nil {
			return nil, err
		}
		delegates := impersonationDelegates(creds.credentialType, serviceAccountEmail)
		accessToken, err = fetchImpersonatedAccessToken(ctx, creds, serviceAccountEmail, delegates)
		if err != nil {
			return nil, err
		}
		credsPath, cleanup, err = delegatedGoogleApplicationCredentials(creds, serviceAccountEmail, delegates)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		accessToken, err = fetchAccessToken(ctx, "")
		if err != nil {
			return nil, err
		}
	}
	env, err := resolveEnvFromConfigFile(ctx, configFile, projectID, accessToken)
	if err != nil {
		return nil, err
	}
	if credsPath != "" {
		env = append(env, googleApplicationCredentialsEnvVar+"="+credsPath)
	}

	cmd := sg.Command(ctx, "go", "run", path)
	cmd.Env = developEnviron(cmd.Env, env, credsPath)
	cmd.Cancel = func() error {
		if err := cmd.Process.Kill(); err != nil {
			return err
		}
		return cleanup()
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
	result = append(result, "K_REVISION=local"+sggit.SHA(ctx))
	result = append(result, "K_CONFIGURATION="+filename)
	result = append(result, "GOOGLE_CLOUD_PROJECT="+project)
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
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return "", fmt.Errorf("expected 200 response status, received %d", response.StatusCode)
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

const (
	// credentialTypeAuthorizedUser is the "type" of a credentials file holding a local user credential, as
	// written by 'gcloud auth application-default login'.
	credentialTypeAuthorizedUser = "authorized_user"
	// googleApplicationCredentialsEnvVar points the Google Cloud client libraries at a credentials file.
	googleApplicationCredentialsEnvVar = "GOOGLE_APPLICATION_CREDENTIALS" //nolint:gosec // a variable name
	// gcloudCredentialFileOverrideEnvVar points gcloud at a credentials file.
	gcloudCredentialFileOverrideEnvVar = "CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE" //nolint:gosec // a variable name
	// gcloudConfigEnvVar overrides the directory holding gcloud's configuration, including the well-known
	// application default credentials path.
	gcloudConfigEnvVar = "CLOUDSDK_CONFIG"
)

// resolvedCredentials is a credentials JSON document together with where it was found.
type resolvedCredentials struct {
	// path is the file the credentials were read from.
	path string
	// data is the raw credentials JSON document.
	data []byte
	// credentialType is the "type" field of the credentials document.
	credentialType string
	// fromEnv reports whether the credentials were resolved from an environment variable rather than from
	// gcloud's well-known path.
	fromEnv bool
}

// gcloudCredentialFileOverride returns the path to pass to gcloud as CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE, so
// that gcloud mints tokens for the same credentials we resolved. It is empty when the credentials came from
// gcloud's well-known path, in which case gcloud's own active account is left in charge.
func (c resolvedCredentials) gcloudCredentialFileOverride() string {
	if !c.fromEnv {
		return ""
	}
	return c.path
}

// resolveCredentials locates and reads the active Application Default Credentials.
//
// The environment variables take precedence over gcloud's well-known path, matching the Google Cloud client
// libraries. google-github-actions/auth writes its credential configuration to a temporary file and points
// these variables at it, so this is what makes impersonation work in CI.
//
// A path named by an environment variable must exist and hold valid JSON. Falling back to the well-known path
// when it does not would silently impersonate from whatever credentials happen to be there, which is the class
// of surprise this resolution order exists to avoid.
func resolveCredentials() (resolvedCredentials, error) {
	type candidate struct {
		path string
		// envVar is the environment variable the path came from, empty for gcloud's well-known path.
		envVar string
	}
	candidates := make([]candidate, 0, 3)
	for _, name := range []string{googleApplicationCredentialsEnvVar, gcloudCredentialFileOverrideEnvVar} {
		if path := os.Getenv(name); path != "" {
			candidates = append(candidates, candidate{path: path, envVar: name})
		}
	}
	configDir := os.Getenv(gcloudConfigEnvVar)
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return resolvedCredentials{}, err
		}
		configDir = filepath.Join(home, ".config", "gcloud")
	}
	candidates = append(candidates, candidate{path: filepath.Join(configDir, "application_default_credentials.json")})
	tried := make([]string, 0, len(candidates))
	for _, c := range candidates {
		tried = append(tried, c.path)
		source := c.path
		if c.envVar != "" {
			source = fmt.Sprintf("%s (from %s)", c.path, c.envVar)
		}
		data, err := os.ReadFile(c.path)
		if err != nil {
			// Only an absent well-known path falls through; gcloud's config dir holding no credentials just
			// means the developer has not run 'gcloud auth application-default login' yet.
			if os.IsNotExist(err) && c.envVar == "" {
				continue
			}
			return resolvedCredentials{}, fmt.Errorf(
				"unable to read application default credentials at %s - %v",
				source,
				err,
			)
		}
		var document struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(data, &document); err != nil {
			return resolvedCredentials{}, fmt.Errorf(
				"invalid application default credentials JSON at %s - %v",
				source,
				err,
			)
		}
		return resolvedCredentials{
			path:           c.path,
			data:           data,
			credentialType: document.Type,
			fromEnv:        c.envVar != "",
		}, nil
	}
	return resolvedCredentials{}, fmt.Errorf(
		"no application default credentials found, tried %s. Please authenticate using "+
			"'gcloud auth application-default login', or point GOOGLE_APPLICATION_CREDENTIALS at a credentials "+
			"file (google-github-actions/auth does this automatically)",
		strings.Join(tried, ", "),
	)
}

// impersonationDelegates returns the delegation chain to use when impersonating serviceAccountEmail.
//
// A local user credential is self-delegated through the impersonated service account, which is how developers
// reach it without a direct getAccessToken binding. See
// https://cloud.google.com/iam/docs/create-short-lived-credentials-delegated#sa-credentials-delegated
// Service account and external account source credentials use plain impersonation instead: they only need
// roles/iam.serviceAccountTokenCreator on the target, whereas self-delegation would additionally require the
// target service account to hold that role on itself.
func impersonationDelegates(credentialType, serviceAccountEmail string) []string {
	if credentialType != credentialTypeAuthorizedUser {
		return nil
	}
	return []string{"projects/-/serviceAccounts/" + serviceAccountEmail}
}

// developEnviron builds the environment for a locally running Cloud Run service: the base environment, then the
// values resolved from the service specification, then the process environment again to allow environment
// overrides. os/exec keeps the last occurrence of a duplicated key, so the trailing copy normally wins.
//
// When credsPath is set, the credential variables are dropped from both the base environment and the overrides,
// leaving the resolved values in charge. google-github-actions/auth exports GOOGLE_APPLICATION_CREDENTIALS and
// CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE, which would otherwise take precedence and silently run the service as
// the CI identity rather than as the impersonated service account.
func developEnviron(base, resolved []string, credsPath string) []string {
	overrides := os.Environ()
	if credsPath != "" {
		base = withoutEnvVars(base, googleApplicationCredentialsEnvVar, gcloudCredentialFileOverrideEnvVar)
		overrides = withoutEnvVars(overrides, googleApplicationCredentialsEnvVar, gcloudCredentialFileOverrideEnvVar)
	}
	environ := make([]string, 0, len(base)+len(resolved)+len(overrides))
	environ = append(environ, base...)
	environ = append(environ, resolved...)
	environ = append(environ, overrides...)
	return environ
}

// withoutEnvVars returns environ without the entries assigning any of the provided variable names.
func withoutEnvVars(environ []string, names ...string) []string {
	result := make([]string, 0, len(environ))
	for _, entry := range environ {
		if slices.ContainsFunc(names, func(name string) bool { return strings.HasPrefix(entry, name+"=") }) {
			continue
		}
		result = append(result, entry)
	}
	return result
}

// fetchAccessToken prints an access token for the active gcloud credential. When credentialFileOverride is not
// empty, gcloud is pointed at that credentials file so that the token identity matches the credentials sage
// resolved rather than whichever account gcloud happens to have active.
func fetchAccessToken(ctx context.Context, credentialFileOverride string) (string, error) {
	var accessTokenOutput strings.Builder
	cmd := sggcloud.Command(ctx, "auth", "print-access-token")
	if credentialFileOverride != "" {
		cmd.Env = append(cmd.Env, gcloudCredentialFileOverrideEnvVar+"="+credentialFileOverride)
	}
	cmd.Stdout = &accessTokenOutput
	if err := cmd.Run(); err != nil {
		return "", err
	}
	accessToken := strings.TrimSpace(accessTokenOutput.String())
	if accessToken == "" {
		return "", fmt.Errorf("got empty access token")
	}

	return accessToken, nil
}

func fetchImpersonatedAccessToken(
	ctx context.Context,
	creds resolvedCredentials,
	serviceAccountEmail string,
	delegates []string,
) (string, error) {
	// Grab the source token to impersonate the service account
	accessToken, err := fetchAccessToken(ctx, creds.gcloudCredentialFileOverride())
	if err != nil {
		return "", err
	}

	// Generate access token for the impersonated service account
	body := struct {
		Delegates []string `json:"delegates,omitempty"`
		Scope     []string `json:"scope"`
	}{
		Delegates: delegates,
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
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		serviceAccountImpersonationURL,
		bytes.NewReader(reqBody),
	)
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

// delegatedGoogleApplicationCredentials writes a credentials file that impersonates serviceAccountEmail on top
// of the provided source credentials, and returns its path along with a func removing it again.
func delegatedGoogleApplicationCredentials(
	creds resolvedCredentials,
	serviceAccountEmail string,
	delegates []string,
) (string, func() error, error) {
	serviceAccountImpersonationURL := fmt.Sprintf(
		"https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/%s:generateAccessToken",
		serviceAccountEmail,
	)

	delegateCreds := struct {
		Delegates                      []string        `json:"delegates,omitempty"`
		Type                           string          `json:"type"`
		ServiceAccountImpersonationURL string          `json:"service_account_impersonation_url"`
		SourceCredentials              json.RawMessage `json:"source_credentials"`
	}{
		Delegates:                      delegates,
		Type:                           "impersonated_service_account",
		ServiceAccountImpersonationURL: serviceAccountImpersonationURL,
		SourceCredentials:              json.RawMessage(creds.data),
	}
	delegateCredsJSON, err := json.Marshal(delegateCreds)
	if err != nil {
		return "", func() error { return nil }, err
	}

	workDir := sg.FromBuildDir("gcloud")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return "", func() error { return nil }, fmt.Errorf("unable to create path to store gcloud credentials: %v", err)
	}
	credsPath := filepath.Join(workDir, fmt.Sprintf("creds-%s.json", randomLower(5)))
	if err := os.WriteFile(credsPath, delegateCredsJSON, 0o600); err != nil {
		return "", func() error { return nil }, err
	}

	return credsPath, func() error { return os.Remove(credsPath) }, nil
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
