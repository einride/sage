package sggit

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"go.einride.tech/sage/sg"
)

// Command returns an [exec.Cmd] for invoking git.
func Command(ctx context.Context, args ...string) *exec.Cmd {
	return sg.Command(ctx, "git", args...)
}

// VerifyNoDiff returns an error if the current working tree has a diff.
// Optional [git pathspecs] can be provided to scope the check to specific files
// (e.g. "*.md", ":(exclude)vendor"). When no pathspecs are given the entire
// working tree is checked.
//
// [git pathspecs]: https://git-scm.com/docs/gitglossary#Documentation/gitglossary.txt-aiddefpathaborpathspec
func VerifyNoDiff(ctx context.Context, pathspecs ...string) error {
	statusArgs := []string{"status", "--porcelain"}
	if len(pathspecs) > 0 {
		statusArgs = append(statusArgs, "--")
		statusArgs = append(statusArgs, pathspecs...)
	}
	cmd := Command(ctx, statusArgs...)
	var status bytes.Buffer
	cmd.Stdout = &status
	if err := cmd.Run(); err != nil {
		return err
	}
	if status.String() != "" {
		dirtyDetails := status.String()
		// attempt to give a nice patch output if the dirty files are tracked by git
		diffArgs := []string{"diff", "--patch"}
		if len(pathspecs) > 0 {
			diffArgs = append(diffArgs, "--")
			diffArgs = append(diffArgs, pathspecs...)
		}
		if patchOutput := sg.Output(Command(ctx, diffArgs...)); patchOutput != "" {
			dirtyDetails = patchOutput
		}
		return fmt.Errorf(
			"staging area is dirty, please add all files created by the build to .gitignore:\n%s",
			dirtyDetails,
		)
	}
	return nil
}

// Tags returns the tags of the current HEAD.
func Tags(ctx context.Context) []string {
	result := strings.Split(sg.Output(Command(ctx, "tag", "--points-at", "HEAD")), "\n")
	if len(result) == 1 && result[0] == "" {
		return nil
	}
	return result
}

// SHA returns the full SHA of the current HEAD.
func SHA(ctx context.Context) string {
	revision := sg.Output(
		Command(ctx, "rev-parse", "--verify", "HEAD"),
	)
	if diff(ctx) != "" {
		revision += "-dirty"
	}
	return revision
}

// ShortSHA returns the short SHA of the current HEAD.
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
