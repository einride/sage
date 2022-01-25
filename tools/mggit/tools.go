package mggit

import (
	"bytes"
	"context"
	"fmt"
	"go.einride.tech/mage-tools/mg"
)

func VerifyNoDiff(ctx context.Context) error {
	cmd := mg.Command(ctx, "git", "status", "--porcelain")
	var status bytes.Buffer
	cmd.Stdout = &status
	if err := cmd.Run(); err != nil {
		return err
	}
	if status.String() != "" {
		if err := mg.Command(ctx, "git", "diff", "--patch").Run(); err != nil {
			return err
		}
		return fmt.Errorf("staging area is dirty, please add all files created by the build to .gitignore")
	}
	return nil
}

func Version(ctx context.Context) string {
	revision := mg.Output(
		mg.Command(ctx, "git", "rev-parse", "--verify", "HEAD"),
	)
	diff := mg.Output(
		mg.Command(ctx, "git", "status", "--porcelain"),
	)
	if diff != "" {
		revision += "-dirty"
	}
	return revision
}
