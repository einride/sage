package semanticrelease

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/einride/mage-tools/tools"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var Binary string

func Run(branch string, ci bool) error {
	mg.Deps(mg.F(semanticRelease, branch))
	releaserc := filepath.Join(tools.GetPath(), "semantic-release", ".releaserc.json")
	args := []string{
		"--extends",
		releaserc,
	}
	if ci {
		args = append(args, "--ci")
	}
	fmt.Println("[semantic-release] creating release...")
	return sh.RunV(Binary, args...)
}

func semanticRelease(branch string) error {
	// Check if npm is installed
	if err := sh.Run("npm", "version"); err != nil {
		return err
	}

	toolDir := filepath.Join(tools.GetPath(), "semantic-release")
	binary := filepath.Join(toolDir, "node_modules", ".bin", "semantic-release")
	releasercJSON := filepath.Join(toolDir, ".releaserc.json")
	packageJSON := filepath.Join(toolDir, "package.json")

	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return err
	}

	packageFileContent := `{
    "devDependencies": {
        "semantic-release": "^17.3.7",
        "@semantic-release/github": "^7.2.0",
        "@semantic-release/release-notes-generator": "^9.0.1",
        "conventional-changelog-conventionalcommits": "^4.5.0"
    }
}`
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

	if err := os.WriteFile(packageJSON, []byte(packageFileContent), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(releasercJSON, []byte(releasercFileContent), 0o644); err != nil {
		return err
	}

	Binary = binary

	fmt.Println("[semantic-release] installing packages...")
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
