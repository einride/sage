package sggit

import (
	"bytes"
	"context"
	"fmt"

	"go.einride.tech/sage/sg"
)

func VerifyNoDiff(ctx context.Context) error {
	cmd := sg.Command(ctx, "git", "status", "--porcelain")
	var status bytes.Buffer
	cmd.Stdout = &status
	if err := cmd.Run(); err != nil {
		return err
	}
	if status.String() != "" {
		if err := sg.Command(ctx, "git", "diff", "--patch").Run(); err != nil {
			return err
		}
		return fmt.Errorf("staging area is dirty, please add all files created by the build to .gitignore")
	}
	return nil
}

func Version(ctx context.Context) string {
	revision := sg.Output(
		sg.Command(ctx, "git", "rev-parse", "--verify", "HEAD"),
	)
	diff := sg.Output(
		sg.Command(ctx, "git", "status", "--porcelain"),
	)
	if diff != "" {
		revision += "-dirty"
	}
	return revision
}
