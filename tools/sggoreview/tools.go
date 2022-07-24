package sggoreview

import (
	"context"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"runtime"
	"unicode"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	name    = "goreview"
	version = "0.22.1"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

// Run goreview in every Go module from the root of the current git repo.
func Run(ctx context.Context, args ...string) error {
	var commands []*exec.Cmd
	if err := filepath.WalkDir(sg.FromGitRoot(), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "go.mod" {
			return nil
		}
		cmd := Command(ctx, append([]string{"-c", "1", "./..."}, args...)...)
		cmd.Dir = filepath.Dir(path)
		commands = append(commands, cmd)
		return cmd.Start()
	}); err != nil {
		return err
	}
	for _, cmd := range commands {
		if err := cmd.Wait(); err != nil {
			return err
		}
	}
	return nil
}

func PrepareCommand(ctx context.Context) error {
	toolDir := sg.FromToolsDir(name)
	binDir := filepath.Join(toolDir, version, "bin")
	binary := filepath.Join(binDir, name)
	goos := []rune(runtime.GOOS)
	hostOS := string(append([]rune{unicode.ToUpper(goos[0])}, goos[1:]...)) // capitalizes the first letter.
	hostArch := runtime.GOARCH
	fileName := fmt.Sprintf("goreview_%s_%s_%s", version, hostOS, hostArch)
	binURL := fmt.Sprintf(
		"https://github.com/einride/goreview/releases/download/v%s/%s.tar.gz",
		version,
		fileName,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(binDir),
		sgtool.WithUntarGz(),
		sgtool.WithRenameFile(fmt.Sprintf("%s/goreview", fileName), name),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", name, err)
	}
	return nil
}
