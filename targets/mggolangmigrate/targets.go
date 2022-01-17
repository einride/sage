package mggolangmigrate

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgtool"
)

const version = "v4.15.1"

// nolint: gochecknoglobals
var executable string

type Prepare mgtool.Prepare

func (Prepare) GolangMigrate(ctx context.Context) error {
	return prepare(ctx)
}

func GolangMigrate(ctx context.Context, source, database string) error {
	logger := mglog.Logger("golang-migrate")
	ctx = logr.NewContext(ctx, logger)
	mg.CtxDeps(ctx, prepare)
	logger.Info("running...")
	return sh.RunV(executable, "-source", source, "-database", database, "up")
}

func prepare(ctx context.Context) error {
	exec, err := mgtool.GoInstall(
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
