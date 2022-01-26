package mggolangmigrate

import (
	"context"
	"os/exec"

	"go.einride.tech/mage-tools/mg"
	"go.einride.tech/mage-tools/mgtool"
)

const version = "v4.15.1"

// nolint: gochecknoglobals
var commandPath string

func Command(ctx context.Context, args ...string) *exec.Cmd {
	mg.Deps(ctx, PrepareCommand)
	return mg.Command(ctx, commandPath, args...)
}

func PrepareCommand(ctx context.Context) error {
	binary, err := mgtool.GoInstall(ctx, "github.com/golang-migrate/migrate/v4/cmd/migrate", version)
	if err != nil {
		return err
	}
	commandPath = binary
	return nil
}
