package sgnpmlicense

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

// @TODO consider replacing this with https://www.npmjs.com/package/license-checker-rseidelsohn
const (
	//nolint: lll
	banList            = "UNLICENCED;GPL-1.0-or-later;LGPL-2.0-or-later;AGPL-3.0;MS-PL;SPL-1.0;CC-BY-NC-1.0;CC-BY-NC-2.0;CC-BY-NC-2.5;CC-BY-NC-4.0;CC-BY-NC-ND-1.0;CC-BY-NC-ND-2.0;CC-BY-NC-ND-2.5;CC-BY-NC-ND-4.0;CC-BY-NC-SA-1.0;CC-BY-NC-SA-2.0;CC-BY-NC-SA-2.5;CC-BY-NC-SA-4.0;EUPL-1.0;EUPL-1.1;EUPL-1.2"
	name               = "license-checker"
	packageJSONContent = `{
  "devDependencies": {
    "license-checker": "25.0.1"
  }
}`
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func LicenseCheckerCommand(ctx context.Context) *exec.Cmd {
	args := []string{
		"--summary",
		"--excludePrivatePackages",
		"--failOn",
		banList,
	}
	return Command(ctx, args...)
}

func PrepareCommand(ctx context.Context) error {
	toolDir := sg.FromToolsDir(name)
	binary := filepath.Join(toolDir, "node_modules", ".bin", name)
	packageJSON := filepath.Join(toolDir, "package.json")
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(packageJSON, []byte(packageJSONContent), 0o600); err != nil {
		return err
	}
	sg.Logger(ctx).Println("installing packages...")
	if err := sg.Command(
		ctx,
		"npm",
		"--silent",
		"install",
		"--prefix",
		toolDir,
		"--no-save",
		"--no-audit",
		"--ignore-script",
	).Run(); err != nil {
		return err
	}
	if _, err := sgtool.CreateSymlink(binary); err != nil {
		return err
	}
	return nil
}
