package mggit

import (
	"bytes"
	"context"
	"fmt"

	"go.einride.tech/mage-tools/mgtool"
)

func VerifyNoDiff(ctx context.Context) error {
	cmd := mgtool.Command(ctx, "git", "status", "--porcelain")
	var status bytes.Buffer
	cmd.Stdout = &status
	if err := cmd.Run(); err != nil {
		return err
	}
	if status.String() != "" {
		if err := mgtool.Command(ctx, "git", "diff", "--patch").Run(); err != nil {
			return err
		}
		return fmt.Errorf("staging area is dirty, please add all files created by the build to .gitignore")
	}
	return nil
}

func Version(ctx context.Context) string {
	revision := mgtool.Output(
		mgtool.Command(ctx, "git", "rev-parse", "--verify", "HEAD"),
	)
	diff := mgtool.Output(
		mgtool.Command(ctx, "git", "status", "--porcelain"),
	)
	if diff != "" {
		revision += "-dirty"
	}
	return revision
}
