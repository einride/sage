package golangmigrate

import (
	"fmt"

	"github.com/einride/mage-tools/tools"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var version string

func SetGolangMigrateVersion(v string) (string, error) {
	version = v
	return version, nil
}

func GolangMigrate(source string, database string) error {
	const defaultVersion = "4.15.1"
	const binaryName = "golang-migrate"

	if version == "" {
		version = defaultVersion
	} else {
		supportedVersions := []string{"0.9.3"}
		if err := tools.IsSupportedVersion(supportedVersions, version, binaryName); err != nil {
			return err
		}
	}

	mg.Deps(mg.F(
		tools.GoTool,
		binaryName,
		fmt.Sprintf("github.com/golang-migrate/migrate/v4/cmd/migrate@v%s", version),
	))
	return sh.RunV("migrate", "-source", source, "-database", database, "up")
}
