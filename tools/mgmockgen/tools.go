package mgmockgen

import (
	"os/exec"

	"go.einride.tech/mage-tools/mgtool"
)

func MockgenGenerate(packageName, destination, moduleName, mocks string) *exec.Cmd {
	return mgtool.Command(
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
