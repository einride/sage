package version

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/tools/sggh"
)

// GetLatestGitHubVersion fetches the latest version from GitHub tags.
// The tagPattern must be a regex with a capture group for the version.
// For example: `^v(\d+\.\d+\.\d+)$` matches "v1.2.3" and extracts "1.2.3".
func GetLatestGitHubVersion(ctx context.Context, repo, tagPattern string) (string, error) {
	sg.Deps(ctx, sggh.PrepareCommand)

	cmd := sggh.Command(ctx, "api",
		fmt.Sprintf("repos/%s/tags", repo),
		"--jq", ".[].name",
		"--paginate",
	)
	cmd.Stdout = nil // Clear stdout set by sg.Command so Output() works
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to fetch tags for %s: %w", repo, err)
	}

	re, err := regexp.Compile(tagPattern)
	if err != nil {
		return "", fmt.Errorf("invalid tag pattern %q: %w", tagPattern, err)
	}

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if matches := re.FindStringSubmatch(line); len(matches) > 1 {
			return matches[1], nil // Return first match (latest)
		}
	}

	return "", fmt.Errorf("no matching tag found for pattern %s in %s", tagPattern, repo)
}
