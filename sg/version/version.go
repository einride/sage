package version

// SourceType indicates where to fetch the latest version from.
type SourceType int

const (
	SourceGitHub SourceType = iota
	SourceGoProxy
	SourceSkip // For tools like sgmarkdownfmt with git hashes
)

// Tool represents a versionable tool with metadata for version checking.
type Tool struct {
	Name           string
	FilePath       string // Relative path to tool file (e.g., "tools/sgbuf/tools.go")
	CurrentVersion string // Read from tool's exported Version constant
	SourceType     SourceType

	// For GitHub-based tools
	Repo       string // e.g., "bufbuild/buf"
	TagPattern string // Regex with capture group for version

	// For Go module tools
	Module string // e.g., "github.com/google/go-licenses/v2"
}

// CheckResult contains the result of checking a tool's version.
type CheckResult struct {
	Tool          Tool
	LatestVersion string
	NeedsUpdate   bool
	Error         error
}
