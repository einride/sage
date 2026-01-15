package sgbun

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	version    = "1.3.6"
	binaryName = "bun"
)

// Command creates a bun command with the given arguments.
func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(binaryName), args...)
}

func PrepareCommand(ctx context.Context) error {
	binDir := sg.FromToolsDir(binaryName, version)
	binary := filepath.Join(binDir, binaryName+"-"+platformArch(), binaryName)
	downloadURL := fmt.Sprintf(
		"https://github.com/oven-sh/bun/releases/download/bun-v%s/bun-%s.zip",
		version,
		platformArch(),
	)

	if err := sgtool.FromRemote(
		ctx,
		downloadURL,
		sgtool.WithDestinationDir(binDir),
		sgtool.WithUnzip(),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}

	if err := os.Chmod(binary, 0o755); err != nil {
		return fmt.Errorf("unable to make %s executable: %w", binaryName, err)
	}

	// bunx is the same binary, just create another symlink
	bunx := filepath.Join(binDir, binaryName+"-"+platformArch(), binaryName+"x")
	if err := os.Symlink(binary, bunx); err != nil && !os.IsExist(err) {
		return fmt.Errorf("unable to create bunx symlink: %w", err)
	}
	if err := os.Chmod(bunx, 0o755); err != nil {
		return fmt.Errorf("unable to make bunx executable: %w", err)
	}
	if _, err := sgtool.CreateSymlink(bunx); err != nil {
		return err
	}

	return nil
}

// Install installs packages into the specified directory using bun install.
// The packages will be installed in dir/node_modules (e.g., .sage/tools/tool-name/version/node_modules).
func Install(ctx context.Context, dir string, packages ...string) error {
	sg.Deps(ctx, PrepareCommand)
	args := append([]string{"install", "--cwd", dir, "--no-save", "--ignore-scripts"}, packages...)
	return Command(ctx, args...).Run()
}

// InstallFromLockfile installs dependencies from package.json and bun.lock.
// Requires both files in dir for reproducible builds with exact dependency versions.
// The --frozen-lockfile flag prevents modifications and enforces that the files are in sync.
func InstallFromLockfile(ctx context.Context, dir string) error {
	sg.Deps(ctx, PrepareCommand)

	// Verify both required files exist
	packageJSON := filepath.Join(dir, "package.json")
	lockfile := filepath.Join(dir, "bun.lock")

	if _, err := os.Stat(packageJSON); err != nil {
		return fmt.Errorf("package.json not found in %s: %w", dir, err)
	}
	if _, err := os.Stat(lockfile); err != nil {
		return fmt.Errorf("bun.lock not found in %s: %w", dir, err)
	}

	return Command(ctx, "install", "--cwd", dir, "--frozen-lockfile", "--no-save", "--ignore-scripts").Run()
}

func platformArch() string {
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH

	switch hostOS {
	case "darwin":
		if hostArch == "arm64" {
			return "darwin-aarch64"
		}
		return "darwin-x64"
	case "linux":
		if hostArch == "arm64" {
			return "linux-aarch64"
		}
		return "linux-x64"
	case "windows":
		return "windows-x64"
	default:
		return fmt.Sprintf("%s-%s", hostOS, hostArch)
	}
}
