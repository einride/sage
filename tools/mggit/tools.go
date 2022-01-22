package mggit

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"go.einride.tech/mage-tools/mgtool"
)

// DefaultBranch returns the default git branch name.
func DefaultBranch(ctx context.Context) string {
	// git branch -r --points-at refs/remotes/origin/HEAD --format '%(refname)'
	cmd := mgtool.Command(
		ctx,
		"git",
		"branch",
		"-r",
		"--points-at",
		"refs/remotes/origin/HEAD",
		"--format",
		"%(refname)",
	)
	var stdout strings.Builder
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "refs/remotes/origin/HEAD" {
			continue
		}
		return strings.TrimPrefix(line, "refs/remotes/origin/")
	}
	panic(fmt.Errorf("failed to determine default git branch"))
}

func VerifyNoDiff(ctx context.Context) error {
	cmd := mgtool.Command(ctx, "git", "status", "--porcelain")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return err
	}
	if stdout.Len() != 0 {
		if err := mgtool.Command(ctx, "git", "diff", "--patch").Run(); err != nil {
			return err
		}
		return fmt.Errorf("staging area is dirty, please add all files created by the build to .gitignore")
	}
	return nil
}

func Version(ctx context.Context) string {
	cmd := mgtool.Command(ctx, "git", "rev-parse", "--verify", "HEAD")
	var b bytes.Buffer
	cmd.Stdout = &b
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	revision := strings.TrimSpace(b.String())
	cmd = mgtool.Command(ctx, "git", "status", "--porcelain")
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
