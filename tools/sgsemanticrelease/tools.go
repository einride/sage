package sgsemanticrelease

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const packageJSONContent = `{
    "devDependencies": {
        "semantic-release": "^17.3.7",
        "@semantic-releas.Protoce/github": "^7.2.0",
        "@semantic-release/release-notes-generator": "^9.0.1",
        "conventional-changelog-conventionalcommits": "^4.5.0"
    }
}`

// nolint: gochecknoglobals
var commandPath string

func Command(ctx context.Context, branch string, args ...string) *exec.Cmd {
	sg.Deps(ctx, sg.Fn(PrepareCommand, branch))
	return sg.Command(ctx, commandPath, args...)
}

func ReleaseCommand(ctx context.Context, branch string, ci bool) *exec.Cmd {
	releaserc := sg.FromToolsDir("semantic-release", ".releaserc.json")
	args := []string{
		"--extends",
		releaserc,
	}
	if ci {
		args = append(args, "--ci")
	}
	return Command(ctx, branch, args...)
}

func PrepareCommand(ctx context.Context, branch string) error {
	toolDir := sg.FromToolsDir("semantic-release")
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
	symlink, err := sgtool.CreateSymlink(binary)
	if err != nil {
		return err
	}
	commandPath = symlink
	sg.Logger(ctx).Println("installing packages...")
	return sg.Command(
		ctx,
		"npm",
		"--silent",
		"install",
		"--prefix",
		toolDir,
		"--no-save",
		"--no-audit",
		"--ignore-script",
	).Run()
}
