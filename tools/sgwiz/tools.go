// Package sgwiz provides commands for the Wiz CLI.
//
// Authentication differs between environments: in a CI pipeline the CLI authenticates with a
// service account, using the WIZ_CLIENT_ID and WIZ_CLIENT_SECRET environment variables, while
// locally it authenticates the developer through the device code flow. The scan commands in this
// package pick the right one, so no separate authentication step is needed. See [Authenticate].
package sgwiz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	// The Wiz CLI is distributed from downloads.wiz.io, which has no Renovate datasource,
	// so this version must be bumped manually.

	version = "1.70.0"
	name    = "wizcli"

	downloadURL = "https://downloads.wiz.io/v1/wizcli"
)

// Authenticate authenticates the Wiz CLI, caching a token for subsequent scans.
//
// The scan commands in this package authenticate on their own, so this is only needed to
// authenticate up front – locally, to get the browser-based device code flow out of the way
// before scanning.
//
// In a CI pipeline, it authenticates with the service account credentials in the WIZ_CLIENT_ID
// and WIZ_CLIENT_SECRET environment variables.
func Authenticate(ctx context.Context) error {
	if isCI() {
		// The Wiz CLI reads the client ID and secret from the environment.
		for _, key := range []string{"WIZ_CLIENT_ID", "WIZ_CLIENT_SECRET"} {
			if os.Getenv(key) == "" {
				return fmt.Errorf("%s must be set to authenticate %s in a CI pipeline", key, name)
			}
		}
	}
	args := append([]string{"auth"}, commonArgs()...)
	return Command(ctx, args...).Run()
}

// ScanDirCommand returns a command that scans dir.
//
// A directory scan covers vulnerabilities, secrets, sensitive data, infrastructure as code
// misconfigurations, software supply chain risks, AI models, SAST weaknesses and malware. Pass
// --disabled-scanners to opt out of any of them.
func ScanDirCommand(ctx context.Context, dir string) *exec.Cmd {
	args := append([]string{"scan", "dir", dir}, scanArgs()...)
	return Command(ctx, args...)
}

// ScanContainerImageCommand returns a command that scans a container image, given by name
// including its tag or digest, or by path to a tarball.
func ScanContainerImageCommand(ctx context.Context, image string) *exec.Cmd {
	args := append([]string{"scan", "container-image", image}, scanArgs()...)
	return Command(ctx, args...)
}

// Command returns a command for the Wiz CLI with the given arguments.
//
// Unlike the scan commands in this package, it applies no default arguments and does not
// authenticate. Use it for subcommands this package doesn't cover, or to opt out of the defaults.
func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

// PrepareCommand downloads and installs the Wiz CLI, verifying the download against the SHA256
// checksum published alongside it. See [verifyChecksum].
//
// A successful verification is recorded next to the binary, so an already installed Wiz CLI is
// neither re-downloaded nor re-verified. The install directory is version scoped, so bumping
// [version] always installs and verifies afresh.
func PrepareCommand(ctx context.Context) error {
	switch runtime.GOOS {
	case "linux", sgtool.Darwin:
	default:
		return fmt.Errorf("unsupported OS in sgwiz package: %s", runtime.GOOS)
	}
	switch runtime.GOARCH {
	case sgtool.AMD64, sgtool.ARM64:
	default:
		return fmt.Errorf("unsupported ARCH in sgwiz package: %s", runtime.GOARCH)
	}
	toolDir := sg.FromToolsDir(name, version)
	binary := filepath.Join(toolDir, name)
	verified := binary + ".verified"
	if _, err := os.Stat(verified); err == nil {
		_, err := sgtool.CreateSymlink(binary)
		return err
	}
	// A gzipped binary is also published, at a third of the size, but only for the latest
	// version and not for the pinned versions this package downloads.
	binURL := fmt.Sprintf("%s/%s/%s-%s-%s", downloadURL, version, name, runtime.GOOS, runtime.GOARCH)
	// The binary is deliberately not symlinked until it has been verified.
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(toolDir),
		sgtool.WithRenameFile("", name),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", name, err)
	}
	if err := os.Chmod(binary, 0o755); err != nil {
		return fmt.Errorf("unable to make %s command: %w", name, err)
	}
	checksumFile := binary + "-sha256"
	if err := sgtool.FromRemote(
		ctx,
		binURL+"-sha256",
		sgtool.WithDestinationDir(toolDir),
		sgtool.WithRenameFile("", filepath.Base(checksumFile)),
	); err != nil {
		return fmt.Errorf("unable to download the %s checksum: %w", name, err)
	}
	if err := verifyChecksum(ctx, binary, checksumFile); err != nil {
		// Remove the binary, so that a tampered with or truncated download can neither be
		// run nor be treated as installed by a later invocation.
		if removeErr := os.Remove(binary); removeErr != nil {
			return fmt.Errorf("%w (unable to remove %s: %v)", err, binary, removeErr)
		}
		return err
	}
	if err := os.WriteFile(verified, []byte(version), 0o600); err != nil {
		return err
	}
	_, err := sgtool.CreateSymlink(binary)
	return err
}

// verifyChecksum verifies the binary against the SHA256 checksum published alongside it.
//
// This detects a corrupted or truncated download. Note that it does not establish that Wiz
// published the binary, as the checksum is served from the same host, over the same channel, as
// the binary itself. Wiz also publishes an OpenPGP signature of the checksum, which would
// establish that, but verifying it requires gpg and is deliberately not done here.
func verifyChecksum(ctx context.Context, binary, checksumFile string) error {
	content, err := os.ReadFile(checksumFile)
	if err != nil {
		return err
	}
	// The published checksum file holds a bare hex digest, with no trailing newline or filename.
	expected := strings.ToLower(strings.TrimSpace(string(content)))
	if len(expected) != hex.EncodedLen(sha256.Size) {
		return fmt.Errorf("unexpected %s checksum %q", name, expected)
	}
	file, err := os.Open(binary)
	if err != nil {
		return err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return err
	}
	if actual := hex.EncodeToString(digest.Sum(nil)); actual != expected {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", binary, expected, actual)
	}
	sg.Logger(ctx).Printf("verified %s checksum", name)
	return nil
}

// scanArgs returns the arguments applied to all scan commands in this package.
func scanArgs() []string {
	var args []string
	if !isCI() {
		// Authenticate the developer rather than a service account. In a CI pipeline the client
		// ID and secret are picked up from the environment instead.
		args = append(args, "--use-device-code")
	}
	// Keep scan results local instead of publishing them to the Wiz portal.
	args = append(args, "--no-publish")
	return append(args, commonArgs()...)
}

// commonArgs returns arguments applied to every Wiz CLI command in this package.
func commonArgs() []string {
	if !isCI() {
		return nil
	}
	// Styled output is unreadable in CI logs, which have no terminal.
	return []string{"--no-color", "--no-style"}
}

// isCI reports whether we are running in a CI pipeline.
func isCI() bool {
	for _, key := range []string{"CI", "GITHUB_ACTIONS"} {
		// CI systems commonly set these to "true" or "1", but treat any other non-empty
		// value as an indication that we are running in a pipeline.
		switch os.Getenv(key) {
		case "", "false", "0":
		default:
			return true
		}
	}
	return false
}
