package version

import (
	"go.einride.tech/sage/tools/sgbuf"
	"go.einride.tech/sage/tools/sggolangcilintv2"
	"go.einride.tech/sage/tools/sggolicenses"
	"go.einride.tech/sage/tools/sgprotocgengogrpc"
)

// tools is the registry of versionable tools.
// Tools are added incrementally as they export their metadata.
//
//nolint:gochecknoglobals
var tools = []Tool{
	{
		Name:           sgbuf.Name,
		FilePath:       "tools/sgbuf/tools.go",
		CurrentVersion: sgbuf.Version,
		Verify:         sgbuf.PrepareCommand,
		SourceType:     SourceGitHub,
		Repo:           sgbuf.Repo,
		TagPattern:     sgbuf.TagPattern,
	},
	{
		Name:           sggolangcilintv2.Name,
		FilePath:       "tools/sggolangcilintv2/tools.go",
		CurrentVersion: sggolangcilintv2.Version,
		Verify:         sggolangcilintv2.PrepareCommandNoCfg,
		SourceType:     SourceGitHub,
		Repo:           sggolangcilintv2.Repo,
		TagPattern:     sggolangcilintv2.TagPattern,
	},
	{
		Name:           sgprotocgengogrpc.Name,
		FilePath:       "tools/sgprotocgengogrpc/tools.go",
		CurrentVersion: sgprotocgengogrpc.Version,
		Verify:         sgprotocgengogrpc.PrepareCommand,
		SourceType:     SourceGitHub,
		Repo:           sgprotocgengogrpc.Repo,
		TagPattern:     sgprotocgengogrpc.TagPattern,
	},
	{
		Name:           sggolicenses.Name,
		FilePath:       "tools/sggolicenses/tools.go",
		CurrentVersion: sggolicenses.Version,
		Verify:         sggolicenses.PrepareCommand,
		SourceType:     SourceGoProxy,
		Module:         sggolicenses.Module,
	},
}
