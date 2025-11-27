package version

import (
	"go.einride.tech/sage/tools/sgbuf"
	"go.einride.tech/sage/tools/sggolangcilintv2"
	"go.einride.tech/sage/tools/sgprotocgengogrpc"
)

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
	{
		Name:           sggolangcilintv2.Name,
		FilePath:       "tools/sggolangcilintv2/tools.go",
		CurrentVersion: sggolangcilintv2.Version,
		SourceType:     SourceGitHub,
		Repo:           sggolangcilintv2.Repo,
		TagPattern:     sggolangcilintv2.TagPattern,
	},
	{
		Name:           sgprotocgengogrpc.Name,
		FilePath:       "tools/sgprotocgengogrpc/tools.go",
		CurrentVersion: sgprotocgengogrpc.Version,
		SourceType:     SourceGitHub,
		Repo:           sgprotocgengogrpc.Repo,
		TagPattern:     sgprotocgengogrpc.TagPattern,
	},
}
