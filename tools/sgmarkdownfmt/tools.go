package sgmarkdownfmt

import (
	"context"
	"os/exec"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const version = "75134924a9fd3335f76a9709314c5f5cef4d6ddc"

//nolint: gochecknoglobals
var commandPath string

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, commandPath, args...)
}

func PrepareCommand(ctx context.Context) error {
	binary, err := sgtool.GoInstall(ctx, "github.com/shurcooL/markdownfmt", version)
	if err != nil {
		return err
	}
	commandPath = binary
	return nil
}
