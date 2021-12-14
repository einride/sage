package mgwire

import (
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mglog"
)

func WireGenerate(path string) error {
	mglog.Logger("wire-generate").Info("generating initializers...")
	return sh.RunV("go", "run", "-mod=mod", "github.com/google/wire/cmd/wire", "gen", path)
}
