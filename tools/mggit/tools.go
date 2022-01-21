package mggit

import (
	"bytes"
	"fmt"
	"strings"

	"go.einride.tech/mage-tools/mgtool"
)

func VerifyNoDiff() error {
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

func Version() string {
	cmd := mgtool.Command("git", "rev-parse", "--verify", "HEAD")
	var b bytes.Buffer
	cmd.Stdout = &b
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	revision := strings.TrimSpace(b.String())
	cmd = mgtool.Command("git", "status", "--porcelain")
	var diff bytes.Buffer
	cmd.Stdout = &diff
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	if diff.String() != "" {
		revision += "-dirty"
	}
	return revision
}
