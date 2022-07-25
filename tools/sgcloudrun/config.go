package sgcloudrun

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/tools/sggcloud"
	"go.einride.tech/sage/tools/sggit"
	"go.einride.tech/sage/tools/sgyq"
)

// Develop starts the Cloud Run service at the provided Go path with the provided service account and config.
func Develop(ctx context.Context, path, keyFile, configFile string) error {
	cmd, err := DevelopCommand(ctx, path, keyFile, configFile)
	if err != nil {
		return err
	}
	return cmd.Run()
}

// DevelopCommand returns an *exec.Cmd pre-configured to start the Cloud Run service at the provided Go path
// with the provided service account and config.
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
