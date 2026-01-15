package sgrenovate

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
	"go.einride.tech/sage/tools/sgbun"
)

const (
	name = "renovate-config-validator"
)

// To update renovate version:
//  1. Update version in package.json
//  2. cd tools/sgrenovate && bun install && rm -rf node_modules
//  3. git add package.json bun.lock

//go:embed package.json
var packageJSONContent []byte

//go:embed bun.lock
var lockfileContent []byte

// ValidateConfig runs renovate-config-validator to validate renovate.json.
func ValidateConfig(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	// Parse version from embedded package.json
	var pkg struct {
		DevDependencies struct {
			Renovate string `json:"renovate"`
		} `json:"devDependencies"`
	}
	if err := json.Unmarshal(packageJSONContent, &pkg); err != nil {
		return fmt.Errorf("failed to parse package.json: %w", err)
	}
	version := pkg.DevDependencies.Renovate

	toolDir := sg.FromToolsDir("renovate", version)
	binary := filepath.Join(toolDir, "node_modules", ".bin", name)
	packageJSON := filepath.Join(toolDir, "package.json")
	lockfile := filepath.Join(toolDir, "bun.lock")

	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return err
	}

	// Write embedded package.json
	if err := os.WriteFile(packageJSON, packageJSONContent, 0o600); err != nil {
		return err
	}

	// Write embedded lockfile
	if err := os.WriteFile(lockfile, lockfileContent, 0o600); err != nil {
		return err
	}

	sg.Logger(ctx).Println("installing renovate package...")
	if err := sgbun.InstallFromLockfile(ctx, toolDir); err != nil {
		return err
	}

	if _, err := sgtool.CreateSymlink(binary); err != nil {
		return err
	}

	return nil
}
