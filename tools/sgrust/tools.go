package sgrust

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	version = "1.63"
)

const installTemplate = `curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | \
sh -s -- -y --no-modify-path --default-toolchain %s`

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	ctx = sg.ContextWithEnv(
		ctx,
		fmt.Sprintf("RUSTUP_HOME=%s", sg.FromToolsDir("rustup")),
		fmt.Sprintf("CARGO_HOME=%s", sg.FromToolsDir("cargo")),
	)
	return sg.Command(ctx, sg.FromBinDir("rustc"), args...)
}

func CargoCommand(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	ctx = sg.ContextWithEnv(
		ctx,
		fmt.Sprintf("RUSTUP_HOME=%s", sg.FromToolsDir("rustup")),
		fmt.Sprintf("CARGO_HOME=%s", sg.FromToolsDir("cargo")),
	)
	return sg.Command(ctx, sg.FromBinDir("cargo"), args...)
}

func RustupCommand(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	ctx = sg.ContextWithEnv(
		ctx,
		fmt.Sprintf("RUSTUP_HOME=%s", sg.FromToolsDir("rustup")),
		fmt.Sprintf("CARGO_HOME=%s", sg.FromToolsDir("cargo")),
	)
	return sg.Command(ctx, sg.FromBinDir("rustup"), args...)
}

func PrepareCommand(ctx context.Context) error {
	if _, err := os.Stat(sg.FromBinDir("rustup")); err == nil {
		return nil
	}
	sg.Logger(ctx).Println("installing Rust toolchain...")
	cmd := sg.Command(
		ctx,
		"bash",
		"-c",
		fmt.Sprintf(
			installTemplate,
			version,
		),
	)
	cmd.Env = append(
		cmd.Env,
		fmt.Sprintf("RUSTUP_HOME=%s", sg.FromToolsDir("rustup")),
		fmt.Sprintf("CARGO_HOME=%s", sg.FromToolsDir("cargo")),
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("install rustup and toolchain: %w", err)
	}
	if _, err := sgtool.CreateSymlink(filepath.Join(sg.FromToolsDir("cargo"), "bin", "rustc")); err != nil {
		return fmt.Errorf("create rustc symlink: %w", err)
	}
	if _, err := sgtool.CreateSymlink(filepath.Join(sg.FromToolsDir("cargo"), "bin", "cargo")); err != nil {
		return fmt.Errorf("create cargo symlink: %w", err)
	}
	if _, err := sgtool.CreateSymlink(filepath.Join(sg.FromToolsDir("cargo"), "bin", "rustup")); err != nil {
		return fmt.Errorf("create rustup symlink: %w", err)
	}
	return nil
}
