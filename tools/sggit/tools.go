package sggit

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"go.einride.tech/sage/sg"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	return sg.Command(ctx, "git", args...)
}

func VerifyNoDiff(ctx context.Context) error {
	cmd := Command(ctx, "status", "--porcelain")
	var status bytes.Buffer
	cmd.Stdout = &status
	if err := cmd.Run(); err != nil {
		return err
	}
	if status.String() != "" {
		dirtyDetails := status.String()
		// attempt to give a nice patch output if the dirty files are tracked by git
		if patchOutput := sg.Output(Command(ctx, "diff", "--patch")); patchOutput != "" {
			dirtyDetails = patchOutput
		}
		return fmt.Errorf(
			"staging area is dirty, please add all files created by the build to .gitignore:\n%s",
			dirtyDetails,
		)
	}
	return nil
}

func SHA(ctx context.Context) string {
	revision := sg.Output(
		Command(ctx, "rev-parse", "--verify", "HEAD"),
	)
	if diff(ctx) != "" {
		revision += "-dirty"
	}
	return revision
}

func ShortSHA(ctx context.Context) string {
	revision := sg.Output(
		Command(ctx, "rev-parse", "--verify", "--short", "HEAD"),
	)
	if diff(ctx) != "" {
		revision += "-dirty"
	}
	return revision
}

func diff(ctx context.Context) string {
	return sg.Output(
		Command(ctx, "status", "--porcelain"),
	)
}
