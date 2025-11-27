# Tool Version Checker

## Problem

Sage tools have hardcoded versions that cannot be updated via Dependabot. We
need a mechanism to detect outdated versions and create PRs for updates.

## Implementation Guidelines

**Keep it simple.** Prioritize clarity over cleverness:

- No over-engineering - solve the problem directly
- No unnecessary abstractions or interfaces
- Simple string matching over complex parsing
- Straightforward control flow
- Minimal dependencies - use stdlib where possible
- Error messages should be clear and actionable

## Design Principles

**Tools own their metadata.** Each tool package exports everything needed for
version checking:

- `Version` - current version string
- `Name` - display name (e.g., "buf")
- `Repo` - GitHub repo for GitHub-sourced tools (e.g., "bufbuild/buf")
- `TagPattern` - regex pattern for matching version tags
- `Module` - Go module path for Go proxy-sourced tools

The registry becomes a simple list of tool references, not a place to duplicate
metadata.

______________________________________________________________________

## Architecture

### Core Types (`sg/version/version.go`)

```go
package version

// SourceType indicates where to fetch the latest version from
type SourceType int

const (
    SourceGitHub SourceType = iota
    SourceGoProxy
    SourceSkip // For tools like sgmarkdownfmt with git hashes
)

// Tool represents a versionable tool with metadata for version checking.
// All fields are populated from the tool package's exported constants.
type Tool struct {
    Name           string
    FilePath       string // Relative path to tool file (e.g., "tools/sgbuf/tools.go")
    CurrentVersion string
    SourceType     SourceType

    // For GitHub-based tools
    Repo       string // e.g., "bufbuild/buf"
    TagPattern string // Regex with capture group for version

    // For Go module tools
    Module string // e.g., "github.com/google/go-licenses/v2"
}

type CheckResult struct {
    Tool          Tool
    LatestVersion string
    NeedsUpdate   bool
    Error         error
}
```

### Tool Package Exports

Each tool package exports its metadata. Example for `sgbuf`:

```go
// tools/sgbuf/tools.go
package sgbuf

const (
    Name       = "buf"
    Version    = "1.47.2"
    Repo       = "bufbuild/buf"
    TagPattern = `^v(\d+\.\d+\.\d+)$`
)
```

For Go proxy tools like `sggolicenses`:

```go
// tools/sggolicenses/tools.go
package sggolicenses

const (
    Name    = "go-licenses"
    Version = "2.0.0-alpha.1"
    Module  = "github.com/google/go-licenses/v2"
)
```

### Registry (`sg/version/registry.go`)

The registry simply references the tool exports:

```go
package version

import (
    "go.einride.tech/sage/tools/sgbuf"
    "go.einride.tech/sage/tools/sggolangcilintv2"
    "go.einride.tech/sage/tools/sggolicenses"
    "go.einride.tech/sage/tools/sgprotocgengogrpc"
)

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
    {
        Name:           sggolicenses.Name,
        FilePath:       "tools/sggolicenses/tools.go",
        CurrentVersion: sggolicenses.Version,
        SourceType:     SourceGoProxy,
        Module:         sggolicenses.Module,
    },
}
```

**Note**: `FilePath` and `SourceType` remain in the registry since they're not
naturally part of the tool's own metadata.

### GitHub Version Check (`sg/version/github.go`)

Use `gh api` for authenticated requests:

```go
func GetLatestGitHubVersion(ctx context.Context, repo, tagPattern string) (string, error) {
    sg.Deps(ctx, sggh.PrepareCommand)

    cmd := sggh.Command(ctx, "api",
        fmt.Sprintf("repos/%s/tags", repo),
        "--jq", ".[].name",
        "--paginate",
    )
    output, err := cmd.Output()
    if err != nil {
        return "", err
    }

    re := regexp.MustCompile(tagPattern)
    for _, line := range strings.Split(string(output), "\n") {
        if matches := re.FindStringSubmatch(line); matches != nil {
            return matches[1], nil // Return first match (latest)
        }
    }
    return "", fmt.Errorf("no matching tag found for pattern %s", tagPattern)
}
```

### Go Proxy Version Check (`sg/version/goproxy.go`)

```go
func GetLatestGoModuleVersion(ctx context.Context, module string) (string, error) {
    url := fmt.Sprintf("https://proxy.golang.org/%s/@latest",
        strings.ReplaceAll(module, "/", "%2F"))

    resp, err := http.Get(url)
    // ... parse JSON response for Version field
}
```

### Update Logic (`sg/version/update.go`)

Update the `Version` constant in tool files:

```go
func UpdateVersion(filePath, oldVersion, newVersion string) error {
    content, err := os.ReadFile(filePath)
    if err != nil {
        return err
    }

    old := fmt.Sprintf(`Version = "%s"`, oldVersion)
    new := fmt.Sprintf(`Version = "%s"`, newVersion)
    updated := strings.Replace(string(content), old, new, 1)

    return os.WriteFile(filePath, []byte(updated), 0o644)
}
```

### Stale PR Handling

Before creating a new PR for a tool, close any existing open PRs:

```go
func CloseExistingPRs(ctx context.Context, toolName string) error {
    branch := fmt.Sprintf("sage/bump-%s", toolName)

    cmd := sggh.Command(ctx, "pr", "list",
        "--head", branch,
        "--state", "open",
        "--json", "number",
        "--jq", ".[].number",
    )
    output, err := cmd.Output()
    if err != nil {
        return err
    }

    for _, numStr := range strings.Split(strings.TrimSpace(string(output)), "\n") {
        if numStr == "" {
            continue
        }
        closeCmd := sggh.Command(ctx, "pr", "close", numStr,
            "--comment", "Superseded by newer version update",
            "--delete-branch",
        )
        if err := closeCmd.Run(); err != nil {
            return err
        }
    }
    return nil
}
```

**Key points:**

- Branch name is `sage/bump-{tool}` (no version) - reused each time
- Force push to same branch overwrites old changes
- Close existing open PRs before creating new one

______________________________________________________________________

## Sage Target

In `.sage/main.go`:

```go
func CheckToolVersions(ctx context.Context, tool string, apply string, pr string) error {
    applyBool := apply == "true"
    prBool := pr == "true"

    // 1. For each tool in registry (or single tool if tool != "all")
    // 2. Fetch latest version from source
    // 3. Compare with CurrentVersion
    // 4. Print report: "buf: 1.59.0 -> 1.60.0 [outdated]"
    // 5. If apply=true: update Version constant in file
    // 6. If pr=true: create branch, commit, close stale PRs, create new PR
}
```

**Usage:**

```bash
make check-tool-versions tool=all apply=false pr=false     # check all (dry-run)
make check-tool-versions tool=buf apply=false pr=false     # check single tool
make check-tool-versions tool=buf apply=true pr=false      # update version in file
make check-tool-versions tool=buf apply=true pr=true       # update + create PR
```

**Verification:** After `apply=true`, run `make` to rebuild with the new version
and verify the tool works through normal build/test targets. The PR's CI also
runs `make`, so if the new version has issues, the PR build fails.

**Note on parameter handling:** The current approach uses string parameters with
explicit "true"/"false" values because sage's Makefile generation requires all
parameters to be set (uses `ifndef` checks). Alternative approaches for the
future:

- Add a dedicated CLI command separate from the Makefile target
- Update `sg/makefile.go` to support default values for bool parameters

______________________________________________________________________

## GitHub Action (`.github/workflows/tool-updates.yml`)

```yaml
name: Check Tool Versions
on:
  schedule:
    - cron: "0 9 1 * *" # Monthly on 1st at 9am UTC
  workflow_dispatch:
    inputs:
      tool:
        description: "Specific tool to check (default: all)"
        required: false
        default: "all"

jobs:
  check:
    runs-on: ubuntu-latest
    outputs:
      outdated: ${{ steps.check.outputs.outdated }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - name: Check versions
        id: check
        run: |
          tool="${{ github.event.inputs.tool || 'all' }}"
          make check-tool-versions tool="$tool" apply=false pr=false 2>&1 | tee output.txt
          outdated=$(grep '\[outdated\]' output.txt | awk -F: '{print $1}' | jq -R -s -c 'split("\n") | map(select(. != ""))')
          echo "outdated=$outdated" >> $GITHUB_OUTPUT
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}

  update:
    needs: check
    if: needs.check.outputs.outdated != '[]' && needs.check.outputs.outdated != ''
    strategy:
      fail-fast: false
      matrix:
        tool: ${{ fromJson(needs.check.outputs.outdated) }}
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - name: Update and create PR
        run: make check-tool-versions tool=${{ matrix.tool }} apply=true pr=true
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

The PR's CI will run `make` which verifies the new version works. If the build
fails, the PR won't be merged.

______________________________________________________________________

## Implementation Checklist

### Commit 1: Core CLI infrastructure

**Message:** `feat(sg/version): add version checker CLI infrastructure`

- [ ] Create `sg/version/version.go` - types
- [ ] Create `sg/version/registry.go` - empty `var Tools = []Tool{}`
- [ ] Create `sg/version/github.go` - `GetLatestGitHubVersion()`
- [ ] Create `sg/version/goproxy.go` - `GetLatestGoModuleVersion()`
- [ ] Create `sg/version/check.go` - `CheckAll()` and `Check()`
- [ ] Create `sg/version/update.go` - `UpdateVersion()`, `CloseExistingPRs()`,
  `CreatePR()`
- [ ] Add `CheckToolVersions` target to `.sage/main.go`
- [ ] Run `make` to verify build passes

### Commit 2: Add buf (basic GitHub tool)

**Message:** `feat(sgbuf): add to version checker registry`

- [ ] Export `Name`, `Version`, `Repo`, `TagPattern` in `tools/sgbuf/tools.go`
- [ ] Add buf entry to registry
- [ ] Test: `make check-tool-versions -- -tool=buf`

### Commit 3: Add golangci-lint

**Message:** `feat(sggolangcilintv2): add to version checker registry`

- [ ] Export metadata in `tools/sggolangcilintv2/tools.go`
- [ ] Add to registry
- [ ] Test: `make check-tool-versions -- -tool=golangci-lint`

### Commit 4: Add protoc-gen-go-grpc (non-standard tag pattern)

**Message:** `feat(sgprotocgengogrpc): add to version checker registry`

- [ ] Export metadata with custom `TagPattern`
- [ ] Add to registry
- [ ] Test: `make check-tool-versions -- -tool=protoc-gen-go-grpc`

### Commit 5: Add go-licenses (Go proxy source)

**Message:** `feat(sggolicenses): add to version checker registry`

- [ ] Export `Name`, `Version`, `Module` in `tools/sggolicenses/tools.go`
- [ ] Add to registry with `SourceType: SourceGoProxy`
- [ ] Test: `make check-tool-versions -- -tool=go-licenses`

### Commit 6: Add GitHub Actions workflow

**Message:** `ci: add monthly tool version check workflow`

- [ ] Create `.github/workflows/tool-updates.yml`
- [ ] Test with manual trigger

______________________________________________________________________

## Files to Create

- `sg/version/version.go` - types
- `sg/version/registry.go` - tool list
- `sg/version/github.go` - GitHub version fetching
- `sg/version/goproxy.go` - Go proxy version fetching
- `sg/version/check.go` - check logic
- `sg/version/update.go` - update + PR logic
- `.github/workflows/tool-updates.yml` - monthly workflow

## Files to Modify

Sage target:

- `.sage/main.go` - add `CheckToolVersions` target

Pilot tools (export metadata):

- `tools/sgbuf/tools.go` - export Name, Version, Repo, TagPattern
- `tools/sggolangcilintv2/tools.go` - export metadata
- `tools/sgprotocgengogrpc/tools.go` - export metadata with custom TagPattern
- `tools/sggolicenses/tools.go` - export Name, Version, Module
