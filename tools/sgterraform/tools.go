package sgterraform

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
	"go.einride.tech/sage/tools/sgghcomment"
)

const (
	version    = "1.2.1"
	binaryName = "terraform"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(binaryName), args...)
}

func CommentOnPullRequestWithPlan(ctx context.Context, prNumber, environment, planFilePath string) *exec.Cmd {
	cmd := Command(
		ctx,
		"show",
		"-no-color",
		filepath.Base(planFilePath),
	)
	cmd.Dir = filepath.Dir(planFilePath)
	cmd.Stdout = nil
	out, err := cmd.Output()
	if err != nil {
		sg.Logger(ctx).Fatal(err)
	}
	comment := fmt.Sprintf(`
<div>
<img
  align="right"
  width="120"
  src="https://upload.wikimedia.org/wikipedia/commons/0/04/Terraform_Logo.svg" />
<h2>Terraform Plan (%s)</h2>
</div>

%s
`, environment, fmt.Sprintf("```"+"hcl\n%s\n"+"```", strings.TrimSpace(string(out))))

	return sgghcomment.Command(
		ctx,
		"--pr",
		prNumber,
		"--signkey",
		environment,
		"--comment",
		comment,
	)
}

func PrepareCommand(ctx context.Context) error {
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	binaryDir := sg.FromToolsDir(binaryName, version)
	binary := filepath.Join(binaryDir, binaryName)
	terraform := fmt.Sprintf("terraform_%s_%s_%s", version, hostOS, hostArch)
	binURL := fmt.Sprintf(
		"https://releases.hashicorp.com/terraform/%s/%s.zip",
		version,
		terraform,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(binaryDir),
		sgtool.WithUnzip(),
		sgtool.WithRenameFile(fmt.Sprintf("%s/terraform", terraform), binaryName),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}

	return nil
}
