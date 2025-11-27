package version

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/tools/sggh"
)

// UpdateVersion updates the Version constant in a tool's file.
func UpdateVersion(filePath, oldVersion, newVersion string) error {
	absPath := filepath.Join(sg.FromGitRoot(), filePath)
	content, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	old := fmt.Sprintf(`Version = "%s"`, oldVersion)
	newStr := fmt.Sprintf(`Version = "%s"`, newVersion)
	updated := strings.Replace(string(content), old, newStr, 1)

	if updated == string(content) {
		return fmt.Errorf("version string %q not found in %s", old, filePath)
	}

	//nolint:gosec // Source files should be world-readable
	if err := os.WriteFile(absPath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", filePath, err)
	}

	return nil
}

// CloseExistingPRs closes any existing open PRs for the given tool.
func CloseExistingPRs(ctx context.Context, toolName string) error {
	sg.Deps(ctx, sggh.PrepareCommand)

	branch := fmt.Sprintf("sage/bump-%s", toolName)

	// Find open PRs from this branch
	cmd := sggh.Command(ctx, "pr", "list",
		"--head", branch,
		"--state", "open",
		"--json", "number",
		"--jq", ".[].number",
	)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list PRs: %w", err)
	}

	// Close each open PR
	for _, numStr := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if numStr == "" {
			continue
		}
		sg.Logger(ctx).Printf("Closing existing PR #%s for %s\n", numStr, toolName)
		closeCmd := sggh.Command(ctx, "pr", "close", numStr,
			"--comment", "Superseded by newer version update",
			"--delete-branch",
		)
		if err := closeCmd.Run(); err != nil {
			return fmt.Errorf("failed to close PR #%s: %w", numStr, err)
		}
	}

	return nil
}

// CreatePR creates a branch and PR for a tool version update.
func CreatePR(ctx context.Context, tool Tool, newVersion string) error {
	sg.Deps(ctx, sggh.PrepareCommand)

	// Close any existing PRs for this tool
	if err := CloseExistingPRs(ctx, tool.Name); err != nil {
		return err
	}

	branch := fmt.Sprintf("sage/bump-%s", tool.Name)
	pkgName := filepath.Base(filepath.Dir(tool.FilePath))

	// Create and checkout branch (force to overwrite if exists)
	if err := sg.Command(ctx, "git", "checkout", "-B", branch).Run(); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Stage the changed file
	if err := sg.Command(ctx, "git", "add", tool.FilePath).Run(); err != nil {
		return fmt.Errorf("failed to stage file: %w", err)
	}

	// Commit
	commitMsg := fmt.Sprintf("feat(%s): bump to v%s", pkgName, strings.TrimPrefix(newVersion, "v"))
	if err := sg.Command(ctx, "git", "commit", "-m", commitMsg).Run(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	// Force push to remote
	if err := sg.Command(ctx, "git", "push", "-f", "origin", branch).Run(); err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	// Create PR
	prTitle := fmt.Sprintf("feat(%s): bump to v%s", pkgName, strings.TrimPrefix(newVersion, "v"))
	prBody := fmt.Sprintf("Bumps %s from %s to %s.\n\nRepository: https://github.com/%s",
		tool.Name,
		tool.CurrentVersion,
		newVersion,
		tool.Repo,
	)

	prCmd := sggh.Command(ctx, "pr", "create",
		"--head", branch,
		"--title", prTitle,
		"--body", prBody,
	)
	if err := prCmd.Run(); err != nil {
		return fmt.Errorf("failed to create PR: %w", err)
	}

	// Switch back to original branch
	if err := sg.Command(ctx, "git", "checkout", "-").Run(); err != nil {
		return fmt.Errorf("failed to switch back to original branch: %w", err)
	}

	return nil
}
