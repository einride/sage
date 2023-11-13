package sgcommitlint

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	name               = "npm-artifact-registry-auth"
	packageJSONContent = `{
		"devDependencies": {
			"google-artifactregistry-auth: "3.1.2",
		}
	}`
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	toolsDir := sg.FromToolsDir(name)
	binary := filepath.Join(toolsDir, "node_modules", ".bin", name)
	packageJSON := filepath.Join(toolsDir, "package.json")
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
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
		toolsDir,
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

func Authenticate(ctx context.Context) error {
	return Command(ctx).Run()
}
