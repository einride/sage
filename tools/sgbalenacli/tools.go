package sgbalenacli

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	toolName   = "balena-cli"
	binaryName = "balena"
	version    = "v15.0.3"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(binaryName), args...)
}

func Whoami(ctx context.Context) (WhoamiInfo, error) {
	cmd := Command(ctx, "whoami")
	cmd.Stdout = nil
	output, err := cmd.Output()
	if err != nil {
		return WhoamiInfo{}, fmt.Errorf("balena whoami failed: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")[1:]
	if len(lines) != 3 {
		return WhoamiInfo{}, fmt.Errorf("unexpected output from Balena: %q", output)
	}

	// Example output we need to trim.
	// == ACCOUNT INFORMATION
	// USERNAME: <username>
	// EMAIL:    <email>
	// URL:      balena-cloud.com
	trim := func(in string) string {
		// trim everything before first :
		i := strings.IndexByte(in, ':') + 1
		return strings.TrimSpace(in[i:])
	}
	w := WhoamiInfo{
		Username: trim(lines[0]),
		Email:    trim(lines[1]),
		URL:      trim(lines[2]),
	}
	return w, nil
}

func PrepareCommand(ctx context.Context) error {
	// Special case: use local balena CLI if available.
	if binary, err := exec.LookPath("balena"); err == nil {
		if _, err := sgtool.CreateSymlink(binary); err != nil {
			return err
		}
		return nil
	}
	binDir := sg.FromToolsDir(toolName, version)
	binary := filepath.Join(binDir, toolName, binaryName)
	hostOS := runtime.GOOS
	if hostOS == sgtool.Darwin {
		hostOS = "macOS"
	}
	balena := fmt.Sprintf("balena-cli-%s-%s-x64-standalone", version, hostOS) // only x64 supported.
	binURL := fmt.Sprintf(
		"https://github.com/balena-io/balena-cli/releases/download/%s/%s.zip",
		version,
		balena,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(binDir),
		sgtool.WithUnzip(),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", toolName, err)
	}
	return nil
}

type WhoamiInfo struct {
	Username string
	Email    string
	URL      string
}
