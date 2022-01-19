package mggitverifynodiff

import (
	"bytes"
	"fmt"

	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgtool"
)

func GitVerifyNoDiff() error {
	mglog.Logger("git-verify-no-diff").Info("verifying that git has no diff...")
	cmd := mgtool.Command("git", "status", "--porcelain")
	var b bytes.Buffer
	cmd.Stdout = &b
	if err := cmd.Run(); err != nil {
		return err
	}
	if b.String() != "" {
		if err := mgtool.Command("git", "diff", "--patch").Run(); err != nil {
			return err
		}
		return fmt.Errorf("staging area is dirty, please add all files created by the build to .gitignore")
	}
	return nil
}
