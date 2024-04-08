// Deprecated: markdownfmt is deprecated and has been replaced by mdformat.
//
// See sgmdformat package for a replacement.
package sgmarkdownfmt

import (
	"context"
	"os/exec"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const version = "75134924a9fd3335f76a9709314c5f5cef4d6ddc"

//nolint:gochecknoglobals
var commandPath string

// Command returns an [exec.Cmd] pointing to the markdownfmt binary.
//
// Deprecated: Use sgmdformat.Command instead.
func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, commandPath, args...)
}

// PrepareCommand downloads the markdownfmt binary and adds it to the PATH.
//
// Deprecated: Use sgmdformat.PrepareCommand instead.
func PrepareCommand(ctx context.Context) error {
	binary, err := sgtool.GoInstall(ctx, "github.com/shurcooL/markdownfmt", version)
	if err != nil {
		return err
	}
	commandPath = binary
	return nil
}
