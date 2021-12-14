package mgmockgen

import (
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mglog"
)

func MockgenGenerate(packageName, destination, moduleName, mocks string) error {
	mglog.Logger("mockgen").Info("generating...", "package", packageName)
	return sh.RunV(
		"go",
		"run",
		"-mod=mod",
		"github.com/golang/mock/mockgen",
		"-package",
		packageName,
		"-destination",
		destination,
		moduleName,
		mocks,
	)
}
