package mgmarkdownfmt

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgtool"
)

const version = "75134924a9fd3335f76a9709314c5f5cef4d6ddc"

// nolint: gochecknoglobals
var executable string

type Prepare mgtool.Prepare

func (Prepare) FormatMarkdown() {
	mg.Deps(prepare)
}

func FormatMarkdown(ctx context.Context) error {
	ctx = logr.NewContext(ctx, mglog.Logger("format-markdown"))
	logr.FromContextOrDiscard(ctx).Info("formatting...")
	mg.Deps(prepare)
	return sh.RunV(executable, "-w", ".")
}

func prepare(ctx context.Context) error {
	exec, err := mgtool.GoInstall(ctx, "github.com/shurcooL/markdownfmt", version)
	if err != nil {
		return err
	}
	executable = exec
	return nil
}
