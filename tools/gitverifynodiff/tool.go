package gitverifynodiff

import (
	"fmt"

	"github.com/magefile/mage/sh"
)

func GitVerifyNoDiff() error {
	fmt.Println("[git-verify-no-diff] verifying that git has no diff...")
	diff, _ := sh.Output("git", "status", "--porcelain")
	if diff != "" {
		_ = sh.RunV("git", "diff", "--patch")
		return fmt.Errorf("staging area is dirty, please add all files created by the build to .gitignore")
	}
	return nil
}
