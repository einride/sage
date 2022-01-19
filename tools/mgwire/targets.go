package mgwire

import (
	"os/exec"

	"go.einride.tech/mage-tools/mgtool"
)

func Command(args ...string) *exec.Cmd {
	args = append([]string{"run", "-mod=mod", "github.com/google/wire/cmd/wire"}, args...)
	return mgtool.Command("go", args...)
}
