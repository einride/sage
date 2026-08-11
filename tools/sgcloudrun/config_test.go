package sgcloudrun

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// clearCredentialEnv clears every environment variable resolveCredentials looks at, so that the developer's own
// environment cannot influence the result.
func clearCredentialEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		googleApplicationCredentialsEnvVar,
		gcloudCredentialFileOverrideEnvVar,
		gcloudConfigEnvVar,
	} {
		t.Setenv(name, "")
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
	}
	// Isolate the well-known gcloud path from the developer's real credentials.
	t.Setenv("HOME", t.TempDir())
}

func writeCredentials(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestResolveCredentials(t *testing.T) {
	t.Run("GOOGLE_APPLICATION_CREDENTIALS takes precedence", func(t *testing.T) {
		clearCredentialEnv(t)
		dir := t.TempDir()
		want := writeCredentials(t, dir, "gac.json", `{"type":"external_account"}`)
		t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", want)
		t.Setenv("CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE", writeCredentials(t, dir, "override.json", `{}`))
		t.Setenv("CLOUDSDK_CONFIG", dir)
		writeCredentials(t, dir, "application_default_credentials.json", `{}`)
		creds, err := resolveCredentials()
		if err != nil {
			t.Fatal(err)
		}
		if creds.path != want {
			t.Errorf("path = %q, want %q", creds.path, want)
		}
		if !creds.fromEnv {
			t.Error("fromEnv = false, want true")
		}
		if creds.credentialType != "external_account" {
			t.Errorf("credentialType = %q, want %q", creds.credentialType, "external_account")
		}
		if got := creds.gcloudCredentialFileOverride(); got != want {
			t.Errorf("gcloudCredentialFileOverride() = %q, want %q", got, want)
		}
	})

	t.Run("falls back to CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE", func(t *testing.T) {
		clearCredentialEnv(t)
		dir := t.TempDir()
		want := writeCredentials(t, dir, "override.json", `{"type":"service_account"}`)
		t.Setenv("CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE", want)
		creds, err := resolveCredentials()
		if err != nil {
			t.Fatal(err)
		}
		if creds.path != want {
			t.Errorf("path = %q, want %q", creds.path, want)
		}
		if !creds.fromEnv {
			t.Error("fromEnv = false, want true")
		}
	})

	// Falling through here would silently impersonate from whatever credentials the next candidate holds,
	// which is the identity surprise this resolution order exists to prevent.
	t.Run("a set but missing env path is an error", func(t *testing.T) {
		clearCredentialEnv(t)
		dir := t.TempDir()
		missing := filepath.Join(dir, "does-not-exist.json")
		t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", missing)
		t.Setenv("CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE", writeCredentials(t, dir, "override.json", `{}`))
		t.Setenv("CLOUDSDK_CONFIG", dir)
		writeCredentials(t, dir, "application_default_credentials.json", `{}`)
		_, err := resolveCredentials()
		if err == nil {
			t.Fatal("expected an error")
		}
		// The message must name both the path and the variable that supplied it.
		for _, want := range []string{missing, "GOOGLE_APPLICATION_CREDENTIALS"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q does not mention %q", err, want)
			}
		}
	})

	t.Run("invalid JSON is an error", func(t *testing.T) {
		clearCredentialEnv(t)
		dir := t.TempDir()
		path := writeCredentials(t, dir, "gac.json", "not json")
		t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", path)
		_, err := resolveCredentials()
		if err == nil {
			t.Fatal("expected an error")
		}
		if !strings.Contains(err.Error(), path) {
			t.Errorf("error %q does not mention %q", err, path)
		}
	})

	t.Run("CLOUDSDK_CONFIG is used for the well-known path", func(t *testing.T) {
		clearCredentialEnv(t)
		dir := t.TempDir()
		want := writeCredentials(t, dir, "application_default_credentials.json", `{"type":"authorized_user"}`)
		t.Setenv("CLOUDSDK_CONFIG", dir)
		creds, err := resolveCredentials()
		if err != nil {
			t.Fatal(err)
		}
		if creds.path != want {
			t.Errorf("path = %q, want %q", creds.path, want)
		}
		// The well-known path leaves gcloud's own active account in charge.
		if creds.fromEnv {
			t.Error("fromEnv = true, want false")
		}
		if got := creds.gcloudCredentialFileOverride(); got != "" {
			t.Errorf("gcloudCredentialFileOverride() = %q, want empty", got)
		}
	})

	t.Run("home directory well-known path", func(t *testing.T) {
		clearCredentialEnv(t)
		home := t.TempDir()
		t.Setenv("HOME", home)
		want := writeCredentials(
			t,
			filepath.Join(home, ".config", "gcloud"),
			"application_default_credentials.json",
			`{"type":"authorized_user"}`,
		)
		creds, err := resolveCredentials()
		if err != nil {
			t.Fatal(err)
		}
		if creds.path != want {
			t.Errorf("path = %q, want %q", creds.path, want)
		}
	})

	// CLOUDSDK_CONFIG names a directory rather than a credentials file, so an absent file inside it is the
	// ordinary not-authenticated case and must not become the hard read error added for env-named paths.
	t.Run("no credentials names both remedies", func(t *testing.T) {
		clearCredentialEnv(t)
		t.Setenv("CLOUDSDK_CONFIG", t.TempDir())
		_, err := resolveCredentials()
		if err == nil {
			t.Fatal("expected an error")
		}
		for _, want := range []string{"gcloud auth application-default login", "GOOGLE_APPLICATION_CREDENTIALS"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q does not mention %q", err, want)
			}
		}
		if strings.Contains(err.Error(), "unable to read") {
			t.Errorf("error %q should be the not-found error, not the read error", err)
		}
	})
}

func TestImpersonationDelegates(t *testing.T) {
	const serviceAccountEmail = "target@project.iam.gserviceaccount.com"
	for _, tt := range []struct {
		credentialType string
		want           []string
	}{
		{
			credentialType: "authorized_user",
			want:           []string{"projects/-/serviceAccounts/" + serviceAccountEmail},
		},
		{credentialType: "service_account"},
		{credentialType: "external_account"},
		{credentialType: "impersonated_service_account"},
		{credentialType: ""},
	} {
		t.Run(tt.credentialType, func(t *testing.T) {
			got := impersonationDelegates(tt.credentialType, serviceAccountEmail)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestWithoutEnvVars(t *testing.T) {
	got := withoutEnvVars(
		[]string{
			"GOOGLE_APPLICATION_CREDENTIALS=/ambient",
			"GOOGLE_APPLICATION_CREDENTIALS_EXTRA=/keep",
			"PREFIXED_GOOGLE_APPLICATION_CREDENTIALS=/keep",
			"CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE=/ambient",
			"GOOGLE_CLOUD_PROJECT=keep",
		},
		"GOOGLE_APPLICATION_CREDENTIALS",
		"CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE",
	)
	want := []string{
		"GOOGLE_APPLICATION_CREDENTIALS_EXTRA=/keep",
		"PREFIXED_GOOGLE_APPLICATION_CREDENTIALS=/keep",
		"GOOGLE_CLOUD_PROJECT=keep",
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// lastEnvValue returns the value of the last assignment of name, which is the one os/exec applies.
func lastEnvValue(environ []string, name string) (string, bool) {
	value, found := "", false
	for _, entry := range environ {
		if after, ok := strings.CutPrefix(entry, name+"="); ok {
			value, found = after, true
		}
	}
	return value, found
}

func TestDevelopEnviron(t *testing.T) {
	// google-github-actions/auth exports these into the job environment.
	const ambient = "/ambient/gha-creds.json"
	resolved := []string{"GOOGLE_APPLICATION_CREDENTIALS=/sage/creds.json", "SOME_SERVICE_CONFIG=from-spec"}
	// sg.Command seeds cmd.Env with the process environment plus entries of its own, so the base already carries
	// the ambient credentials and must be filtered too.
	base := func() []string {
		return append(os.Environ(), "SAGE_BASE_ONLY=from-base")
	}

	t.Run("sage credentials win over the ambient ones", func(t *testing.T) {
		t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", ambient)
		t.Setenv("CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE", ambient)
		t.Setenv("SOME_SERVICE_CONFIG", "from-environment")
		environ := developEnviron(base(), resolved, "/sage/creds.json")
		if got, _ := lastEnvValue(environ, "GOOGLE_APPLICATION_CREDENTIALS"); got != "/sage/creds.json" {
			t.Errorf("GOOGLE_APPLICATION_CREDENTIALS = %q, want %q", got, "/sage/creds.json")
		}
		// gcloud must not be pointed at a different identity than the service.
		if got, found := lastEnvValue(environ, "CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE"); found {
			t.Errorf("CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE = %q, want it to be removed", got)
		}
		// The base environment is otherwise preserved.
		if got, _ := lastEnvValue(environ, "SAGE_BASE_ONLY"); got != "from-base" {
			t.Errorf("SAGE_BASE_ONLY = %q, want %q", got, "from-base")
		}
		// Every other variable keeps its documented override behavior.
		if got, _ := lastEnvValue(environ, "SOME_SERVICE_CONFIG"); got != "from-environment" {
			t.Errorf("SOME_SERVICE_CONFIG = %q, want %q", got, "from-environment")
		}
	})

	t.Run("without impersonation the ambient credentials are kept", func(t *testing.T) {
		t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", ambient)
		t.Setenv("CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE", ambient)
		environ := developEnviron(base(), []string{"SOME_SERVICE_CONFIG=from-spec"}, "")
		if got, _ := lastEnvValue(environ, "GOOGLE_APPLICATION_CREDENTIALS"); got != ambient {
			t.Errorf("GOOGLE_APPLICATION_CREDENTIALS = %q, want %q", got, ambient)
		}
		if got, _ := lastEnvValue(environ, "CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE"); got != ambient {
			t.Errorf("CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE = %q, want %q", got, ambient)
		}
		// Values from the service specification still apply when nothing overrides them.
		if got, _ := lastEnvValue(environ, "SOME_SERVICE_CONFIG"); got != "from-spec" {
			t.Errorf("SOME_SERVICE_CONFIG = %q, want %q", got, "from-spec")
		}
	})
}
