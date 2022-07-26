package sgfirebasetools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	version    = "v11.2.2"
	binaryName = "firebase-tools"
)

type DeployPreviewOptions struct {
	// Project to deploy in.
	Project string
	// Site can be set if the firebase config contains config for multiple sites. Channel will be deployed to this site.
	Site string
	// ChannelID of the preview channel. If it does not exist, it will be created.
	ChannelID string
	// CmdDir can be set if the firebase.json file is not in root of repository.
	CmdDir string
}

// DeployPreview deploy static files to a Firebase hosting channel. Returns the generated URL
// for the hosting channel.
func DeployPreview(ctx context.Context, opts DeployPreviewOptions) (string, error) {
	args := []string{
		"hosting:channel:deploy",
		opts.ChannelID,
		"--project", opts.Project,
		"--json",
	}
	if opts.Site != "" {
		args = append(args, "--only", opts.Site)
	}
	cmd := Command(
		ctx,
		args...,
	)
	if opts.CmdDir != "" {
		cmd.Dir = opts.CmdDir
	}
	cmd.Stdout = nil
	cmd.Stderr = nil
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("deploy to hosting channel: %w\n%s", err, output)
	}
	var parsedOutput struct {
		Result map[string]struct {
			URL string
		}
	}
	if err := json.Unmarshal(output, &parsedOutput); err != nil {
		return "", fmt.Errorf("parse output: %w\n%s", err, output)
	}
	// assume that site name is equal to project name if site option not set
	url := parsedOutput.Result[opts.Project].URL
	if opts.Site != "" {
		url = parsedOutput.Result[opts.Site].URL
	}
	// deployed preview defaulted to another site than specified project name or site name
	if url == "" {
		return "", fmt.Errorf("could not resolve preview url. Make sure to specify site in the deploy preview options")
	}
	return url, nil
}

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(binaryName), args...)
}

func PrepareCommand(ctx context.Context) error {
	binOS := "linux"
	if runtime.GOOS == "darwin" {
		binOS = "macos"
	}
	binaryDir := sg.FromToolsDir(binaryName, version)
	binary := filepath.Join(binaryDir, binaryName)
	filename := fmt.Sprintf("firebase-tools-%s", binOS)
	binURL := fmt.Sprintf(
		"https://github.com/firebase/firebase-tools/releases/download/%s/%s",
		version,
		filename,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(binaryDir),
		sgtool.WithRenameFile("", binaryName),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	return nil
}
