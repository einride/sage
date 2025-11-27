package version

import "go.einride.tech/sage/tools/sgbuf"

// Tools is the registry of versionable tools.
// Tools are added incrementally as they export their metadata.
//
//nolint:gochecknoglobals
var Tools = []Tool{
	{
		Name:           sgbuf.Name,
		FilePath:       "tools/sgbuf/tools.go",
		CurrentVersion: sgbuf.Version,
		SourceType:     SourceGitHub,
		Repo:           sgbuf.Repo,
		TagPattern:     sgbuf.TagPattern,
	},
}
