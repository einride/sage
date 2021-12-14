package mggitverifynodiff

import (
	"fmt"

	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mglog"
)

func GitVerifyNoDiff() error {
	mglog.Logger("git-verify-no-diff").Info("verifying that git has no diff...")
	diff, _ := sh.Output("git", "status", "--porcelain")
	if diff != "" {
		_ = sh.RunV("git", "diff", "--patch")
		return fmt.Errorf("staging area is dirty, please add all files created by the build to .gitignore")
	}
	return nil
}
