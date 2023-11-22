package sgmdformat

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
	"go.einride.tech/sage/tools/sggit"
	"go.einride.tech/sage/tools/sgpython"
)

const (
	name    = "mdformat"
	syntax  = "gfm"
	version = "0.3.5"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	args = setDefaultArgs(ctx, args)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

// setDefaultArgs to iterate numbers on ordered lists and wrap at 80 chars.
func setDefaultArgs(ctx context.Context, args []string) []string {
	if len(args) != 0 {
		return args
	}
	args = []string{
		"--number",
		"--wrap",
		"80",
	}
	args = append(args, listMarkdownFiles(ctx)...)
	return args
}

func PrepareCommand(ctx context.Context) error {
	toolDir := sg.FromToolsDir(name, version)
	mdformat := filepath.Join(toolDir, "bin", name)
	if _, err := os.Stat(mdformat); err == nil {
		if _, err := sgtool.CreateSymlink(mdformat); err != nil {
			return err
		}
		return nil
	}
	if err := sgpython.Command(ctx, "-m", "venv", toolDir).Run(); err != nil {
		return err
	}
	pip := filepath.Join(toolDir, "bin", "pip")
	if err := sg.Command(ctx, pip, "install", "-U", "pip").Run(); err != nil {
		return err
	}
	if err := sg.Command(ctx, pip, "install", name+"-"+syntax+"=="+version).Run(); err != nil {
		return err
	}
	if _, err := sgtool.CreateSymlink(mdformat); err != nil {
		return err
	}
	return nil
}

// list markdown files known by git + untracked ones.
func listMarkdownFiles(ctx context.Context) []string {
	output := strings.TrimSpace(
		sg.Output(sggit.Command(
			ctx,
			"ls-files",
			"*.md",               // only markdown files
			"--others",           // include untracked files
			"--exclude-standard", // exclude ignored files
			"--cached",           // include "normal" files
		)),
	)
	if len(output) == 0 {
		return nil
	}
	return strings.Split(output, "\n")
}
