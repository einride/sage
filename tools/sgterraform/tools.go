package sgterraform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"text/tabwriter"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
	"go.einride.tech/sage/tools/sgghcomment"
)

const (
	version    = "1.5.2"
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

func CommentOnPRWithPlanSummarized(ctx context.Context, prNumber, environment, planFilePath string) *exec.Cmd {
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
	statusIcon, summary := getCommentSummary(ctx, planFilePath)
	comment := fmt.Sprintf(`
<div>
<img
  align="right"
  width="120"
  src="https://upload.wikimedia.org/wikipedia/commons/0/04/Terraform_Logo.svg" />
<h2>%s Terraform Plan (%s) %s</h2>
</div>
<pre><code>%s</code></pre>
<details>
<summary>Details (Click me)</summary>
<p>

%s

</p>
</details>
`,
		statusIcon,
		environment,
		statusIcon,
		summary,
		fmt.Sprintf("```"+"hcl\n%s\n"+"```", strings.TrimSpace(string(out))))
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

func getCommentSummary(ctx context.Context, planFilePath string) (statusIcon, summary string) {
	var jsonBuf bytes.Buffer
	cmdJSON := Command(
		ctx,
		"show",
		"-no-color",
		"-json",
		filepath.Base(planFilePath),
	)
	cmdJSON.Dir = filepath.Dir(planFilePath)
	cmdJSON.Stdout = &jsonBuf
	if err := cmdJSON.Run(); err != nil {
		sg.Logger(ctx).Fatal(err)
	}

	var jsonPlan Plan
	err := json.Unmarshal(jsonBuf.Bytes(), &jsonPlan)
	if err != nil {
		sg.Logger(ctx).Fatal(err)
	}

	create := TfChange{actionName: "Create", changes: make(map[string]int), actionCount: 0}
	destroy := TfChange{actionName: "Destroy", changes: make(map[string]int), actionCount: 0}
	update := TfChange{actionName: "Update", changes: make(map[string]int), actionCount: 0}
	replace := TfChange{actionName: "Replace", changes: make(map[string]int), actionCount: 0}
	for _, res := range jsonPlan.ResourceChanges {
		actions := res.Change.Actions
		resourceType := res.Type
		switch {
		case actions.Create():
			create.add(resourceType)
		case actions.Delete():
			destroy.add(resourceType)
		case actions.Update():
			update.add(resourceType)
		case actions.Replace():
			replace.add(resourceType)
		case
			actions.NoOp(),
			actions.Read():
			// Do nothing
			continue
		default:
			sg.Logger(ctx).Fatal(fmt.Errorf("unable to determine resource operation: %v", res))
		}
	}

	statusIcon = ":green_circle:"
	if update.actionCount > 0 {
		statusIcon = ":orange_circle:"
	}
	if destroy.actionCount > 0 || replace.actionCount > 0 {
		statusIcon = ":red_circle:"
	}

	summary = fmt.Sprintf(`
Plan Summary: %d to create, %d to update, %d to replace, %d to destroy.
<br/>
%s
`,
		create.actionCount,
		update.actionCount,
		replace.actionCount,
		destroy.actionCount,
		mapToHTMLList([]TfChange{create, destroy, update, replace}),
	)

	return statusIcon, summary
}

type TfChange struct {
	actionName  string
	actionCount int
	changes     map[string]int
}

func (t *TfChange) add(resourceType string) {
	t.actionCount++
	t.changes[resourceType]++
}

func mapToHTMLList(input []TfChange) string {
	if len(input) == 0 {
		return ""
	}
	var htmlList string
	for _, v := range input {
		if v.actionCount == 0 {
			continue
		}
		keys := make([]string, 0, len(input))
		for k := range v.changes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var b bytes.Buffer
		w := tabwriter.NewWriter(&b, 0, 0, 3, '.', tabwriter.FilterHTML)
		htmlList += fmt.Sprintf("<b>%s (%d):</b><ul>", v.actionName, v.actionCount)
		for _, key := range keys {
			fmt.Fprintf(w, "%s\t%d\n", key, v.changes[key])
		}
		w.Flush()
		htmlList += b.String() + "</ul>"
	}
	return htmlList
}
