package golangmigrate

import (
	"github.com/einride/mage-tools/tools"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

func GolangMigrate(source string, database string) error {
	mg.Deps(mg.F(tools.GoTool, "golang-migrate", "github.com/golang-migrate/migrate/v4/cmd/migrate@v4.15.1"))
	err := sh.RunV("migrate", "-source", source, "-database", database, "up")
	if err != nil {
		return err
	}
	return nil
}
