package mggolangmigrate

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/tools"
)

const version = "v4.15.1"

var executable string

func GolangMigrate(ctx context.Context, source string, database string) error {
	logger := mglog.Logger("golang-migrate")
	ctx = logr.NewContext(ctx, logger)
	mg.CtxDeps(ctx, prepare)
	logger.Info("running...")
	return sh.RunV(executable, "-source", source, "-database", database, "up")
}

func prepare(ctx context.Context) error {
	exec, err := tools.GoInstall(
		ctx,
		"github.com/golang-migrate/migrate/v4/cmd/migrate",
		version,
	)
	if err != nil {
		return err
	}
	executable = exec
	return nil
}
