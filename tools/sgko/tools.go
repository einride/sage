package sgko

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
	name    = "ko"
	version = "0.15.2"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	binDir := sg.FromToolsDir(name, version, "bin")
	binary := filepath.Join(binDir, name)
	arch := runtime.GOARCH
	if arch == sgtool.AMD64 {
		arch = sgtool.X8664
	}
	binURL := fmt.Sprintf(
		"https://github.com/google/ko/releases/download/v%s/ko_%s_%s_%s.tar.gz",
		version,
		version,
		toInitialCamel(runtime.GOOS),
		arch,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(binDir),
		sgtool.WithUntarGz(),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", name, err)
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
