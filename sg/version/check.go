package version

import (
	"context"
	"fmt"
)

// CheckAll checks all tools in the registry for updates.
func CheckAll(ctx context.Context) []CheckResult {
	results := make([]CheckResult, 0, len(tools))
	for _, tool := range tools {
		results = append(results, check(ctx, tool))
	}
	return results
}

// CheckByName checks a single tool by name.
func CheckByName(ctx context.Context, name string) (CheckResult, error) {
	for _, tool := range tools {
		if tool.Name == name {
			return check(ctx, tool), nil
		}
	}
	return CheckResult{}, fmt.Errorf("tool %q not found in registry", name)
}

// check checks a single tool for updates.
func check(ctx context.Context, tool Tool) CheckResult {
	result := CheckResult{Tool: tool}

	var latestVersion string
	var err error

	switch tool.SourceType {
	case SourceGitHub:
		latestVersion, err = getLatestGitHubVersion(ctx, tool.Repo, tool.TagPattern)
	case SourceGoProxy:
		latestVersion, err = getLatestGoModuleVersion(ctx, tool.Module)
	case SourceSkip:
		return result // No update check for skipped tools
	default:
		result.Error = fmt.Errorf("unknown source type: %d", tool.SourceType)
		return result
	}

	if err != nil {
		result.Error = err
		return result
	}

	result.LatestVersion = latestVersion
	result.NeedsUpdate = tool.CurrentVersion != latestVersion

	return result
}
