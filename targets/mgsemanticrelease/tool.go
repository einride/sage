package mgsemanticrelease

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgtool"
)

const packageJSONContent = `{
    "devDependencies": {
        "semantic-release": "^17.3.7",
        "@semantic-release/github": "^7.2.0",
        "@semantic-release/release-notes-generator": "^9.0.1",
        "conventional-changelog-conventionalcommits": "^4.5.0"
    }
}`

// nolint: gochecknoglobals
var executable string

func SemanticRelease(ctx context.Context, branch string, ci bool) error {
	logger := mglog.Logger("semantic-release")
	ctx = logr.NewContext(ctx, logger)
	mg.CtxDeps(ctx, mg.F(prepare, branch))
	releaserc := filepath.Join(mgtool.GetPath(), "semantic-release", ".releaserc.json")
	args := []string{
		"--extends",
		releaserc,
	}
	if ci {
		args = append(args, "--ci")
	}
	logger.Info("creating release...")
	return sh.RunV(executable, args...)
}

func prepare(ctx context.Context, branch string) error {
	// Check if npm is installed
	if err := sh.Run("npm", "version"); err != nil {
		return err
	}

	toolDir := filepath.Join(mgtool.GetPath(), "semantic-release")
	binary := filepath.Join(toolDir, "node_modules", ".bin", "semantic-release")
	releasercJSON := filepath.Join(toolDir, ".releaserc.json")
	packageJSON := filepath.Join(toolDir, "package.json")

	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return err
	}

	releasercFileContent := fmt.Sprintf(`{
  "plugins": [
    [
      "@semantic-release/commit-analyzer",
      {
        "preset": "conventionalcommits",
        "releaseRules": [
          {
            "type": "chore",
            "release": "patch"
          },
          {
            "breaking": true,
            "release": "minor"
          }
        ]
      }
    ],
    "@semantic-release/release-notes-generator",
    "@semantic-release/github"
  ],
  "branches": [
    "%s"
  ],
  "success": false,
  "fail": false
}`, branch)

	if err := os.WriteFile(packageJSON, []byte(packageJSONContent), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(releasercJSON, []byte(releasercFileContent), 0o600); err != nil {
		return err
	}

	executable = binary

	logr.FromContextOrDiscard(ctx).Info("installing packages...")
	return sh.Run(
		"npm",
		"--silent",
		"install",
		"--prefix",
		toolDir,
		"--no-save",
		"--no-audit",
		"--ignore-script",
	)
}
