package mgmarkdownfmt

import (
	"context"
	"os/exec"

	"go.einride.tech/mage-tools/mg"
	"go.einride.tech/mage-tools/mgtool"
)

const version = "75134924a9fd3335f76a9709314c5f5cef4d6ddc"

// nolint: gochecknoglobals
var commandPath string

func Command(ctx context.Context, args ...string) *exec.Cmd {
	mg.Deps(ctx, PrepareCommand)
	return mg.Command(ctx, commandPath, args...)
}

func PrepareCommand(ctx context.Context) error {
	binary, err := mgtool.GoInstall(ctx, "github.com/shurcooL/markdownfmt", version)
	if err != nil {
		return err
	}
	commandPath = binary
	return nil
}
