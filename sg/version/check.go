package version

import (
	"context"
	"fmt"
)

// CheckAll checks all tools in the registry for updates.
func CheckAll(ctx context.Context) []CheckResult {
	results := make([]CheckResult, 0, len(Tools))
	for _, tool := range Tools {
		results = append(results, Check(ctx, tool))
	}
	return results
}

// CheckByName checks a single tool by name.
func CheckByName(ctx context.Context, name string) (CheckResult, error) {
	for _, tool := range Tools {
		if tool.Name == name {
			return Check(ctx, tool), nil
		}
	}
	return CheckResult{}, fmt.Errorf("tool %q not found in registry", name)
}

// Check checks a single tool for updates.
func Check(ctx context.Context, tool Tool) CheckResult {
	result := CheckResult{Tool: tool}

	var latestVersion string
	var err error

	switch tool.SourceType {
	case SourceGitHub:
		latestVersion, err = GetLatestGitHubVersion(ctx, tool.Repo, tool.TagPattern)
	case SourceGoProxy:
		latestVersion, err = GetLatestGoModuleVersion(ctx, tool.Module)
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
