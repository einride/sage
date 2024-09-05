package sgcspevaluatorcli

import (
	"context"
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const name = "csp"

//go:embed package.json
var packageJSONContent []byte

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	sg.Logger(ctx).Println("installing packages...")
	toolDir := sg.FromToolsDir(name)
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return err
	}
	packageJSONPath := filepath.Join(toolDir, "package.json")
	if err := os.WriteFile(packageJSONPath, packageJSONContent, 0o600); err != nil {
		return err
	}
	cmd := sg.Command(
		ctx,
		"npm",
		"--silent",
		"install",
		"--prefix",
		toolDir,
		"--no-save",
		"--no-audit",
		"--ignore-script",
	)
	if err := cmd.Run(); err != nil {
		return err
	}
	binary := filepath.Join(toolDir, "node_modules", ".bin", name)
	_, err := sgtool.CreateSymlink(binary)
	return err
}
