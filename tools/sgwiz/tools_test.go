package sgwiz

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestVerifyChecksum(t *testing.T) {
	// echo -n "wizcli" | shasum -a 256
	const (
		content  = "wizcli"
		checksum = "fcaedabad6acbd6c4278824799b267faae2c87ad7e96b1fdd3ce4af7fa042a95"
	)
	write := func(t *testing.T, binaryContent, checksumContent string) (string, string) {
		t.Helper()
		dir := t.TempDir()
		binary := filepath.Join(dir, name)
		checksumFile := binary + "-sha256"
		if err := os.WriteFile(binary, []byte(binaryContent), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(checksumFile, []byte(checksumContent), 0o600); err != nil {
			t.Fatal(err)
		}
		return binary, checksumFile
	}
	t.Run("mismatch is rejected", func(t *testing.T) {
		binary, checksumFile := write(t, "tampered with", checksum)
		if err := verifyChecksum(context.Background(), binary, checksumFile); err == nil {
			t.Error("verifyChecksum() = nil, expected a checksum mismatch")
		}
	})
	t.Run("truncated binary is rejected", func(t *testing.T) {
		binary, checksumFile := write(t, content[:3], checksum)
		if err := verifyChecksum(context.Background(), binary, checksumFile); err == nil {
			t.Error("verifyChecksum() = nil, expected a checksum mismatch")
		}
	})
	t.Run("malformed checksum is rejected", func(t *testing.T) {
		binary, checksumFile := write(t, content, "not-a-checksum")
		if err := verifyChecksum(context.Background(), binary, checksumFile); err == nil {
			t.Error("verifyChecksum() = nil, expected a malformed checksum error")
		}
	})
	t.Run("trailing whitespace is tolerated", func(t *testing.T) {
		binary, checksumFile := write(t, content, checksum+"\n")
		if err := verifyChecksum(context.Background(), binary, checksumFile); err != nil {
			t.Errorf("verifyChecksum() = %v, expected the checksum to match", err)
		}
	})
}

func TestIsCI(t *testing.T) {
	for _, tt := range []struct {
		name          string
		ci            string
		githubActions string
		expected      bool
	}{
		{name: "unset", expected: false},
		{name: "CI true", ci: "true", expected: true},
		{name: "CI 1", ci: "1", expected: true},
		{name: "CI false", ci: "false", expected: false},
		{name: "CI empty", ci: "", expected: false},
		{name: "GitHub Actions", githubActions: "true", expected: true},
		{name: "CI disabled in GitHub Actions", ci: "false", githubActions: "true", expected: true},
		{name: "non-boolean value", ci: "yes", expected: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CI", tt.ci)
			t.Setenv("GITHUB_ACTIONS", tt.githubActions)
			if actual := isCI(); actual != tt.expected {
				t.Errorf("isCI() = %v, expected %v", actual, tt.expected)
			}
		})
	}
}

func TestScanArgs(t *testing.T) {
	t.Run("locally", func(t *testing.T) {
		t.Setenv("CI", "")
		t.Setenv("GITHUB_ACTIONS", "")
		actual := scanArgs()
		for _, arg := range []string{"--use-device-code", "--no-publish"} {
			if !slices.Contains(actual, arg) {
				t.Errorf("scanArgs() = %v, expected it to contain %s", actual, arg)
			}
		}
		if slices.Contains(actual, "--no-color") {
			t.Errorf("scanArgs() = %v, expected it not to disable styled output locally", actual)
		}
	})
	t.Run("in a CI pipeline", func(t *testing.T) {
		t.Setenv("CI", "true")
		actual := scanArgs()
		for _, arg := range []string{"--no-publish", "--no-color", "--no-style"} {
			if !slices.Contains(actual, arg) {
				t.Errorf("scanArgs() = %v, expected it to contain %s", actual, arg)
			}
		}
		// In CI the service account credentials are used, not the developer's own.
		if slices.Contains(actual, "--use-device-code") {
			t.Errorf("scanArgs() = %v, expected it not to use the device code flow in CI", actual)
		}
	})
}
