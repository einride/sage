package sgbuf

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"unicode"
	"unicode/utf8"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	// Version is the buf version.
	Version = "1.59.0"
	// Name is the buf binary name.
	Name = "buf"
	// Repo is the GitHub repository.
	Repo = "bufbuild/buf"
	// TagPattern matches version tags.
	TagPattern = `^v(\d+\.\d+\.\d+)$`
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(Name), args...)
}

func PrepareCommand(ctx context.Context) error {
	toolDir := sg.FromToolsDir(Name, Version)
	binary := filepath.Join(toolDir, Name, "bin", Name)
	arch := runtime.GOARCH
	if arch == sgtool.AMD64 {
		arch = sgtool.X8664
	}
	binURL := fmt.Sprintf(
		"https://github.com/bufbuild/buf/releases/download/v%s/buf-%s-%s.tar.gz",
		Version,
		toInitialCamel(runtime.GOOS),
		arch,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(toolDir),
		sgtool.WithUntarGz(),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", Name, err)
	}
	return nil
}

func toInitialCamel(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return s
	}
	return string(unicode.ToUpper(r)) + s[n:]
}
