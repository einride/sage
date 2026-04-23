package sg

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/doc"
	"sort"
	"strings"
	"text/template"
)

// GitHubWorkflow configures generation of a GitHub Actions workflow file
// that mirrors the call graph of Makefile.DefaultTarget.
//
// When set, GenerateMakefiles invokes the compiled sagefile binary in plan
// mode (see PlanOutputEnv) to record how the default target composes its
// sub-targets, then emits one job per recorded target connected by
// `needs:` edges that preserve the ordering implied by sg.SerialDeps.
//
// Constraints on the default target:
//   - It may only compose sub-targets via sg.Deps and sg.SerialDeps. Any
//     other work (writing files, running shell commands, calling arbitrary
//     helpers) will execute while the plan is captured and cause surprise.
//   - Sub-targets must be plain func(context.Context) error. Targets that
//     take arguments (via sg.Fn with args) are not supported because the
//     generated Makefile targets are invoked as `make <target>`.
type GitHubWorkflow struct {
	// Path is the output path of the generated workflow YAML,
	// e.g. ".github/workflows/ci.yml".
	Path string
	// Name is the workflow name used for the YAML "name:" field.
	// Defaults to "CI".
	Name string
	// SetupAction is the value of "uses:" on the setup step of each job.
	// Defaults to "einride/sage/actions/setup@master".
	SetupAction string
	// SetupWith populates the "with:" block on the "Setup Sage" step of
	// every job. Each key/value pair becomes one input to the action named
	// by SetupAction. It does not affect the checkout step or the
	// `make <target>` step, and the same map is applied identically to
	// every job (per-job overrides are not supported).
	// Defaults to {"go-version-file": "go.mod"}.
	SetupWith map[string]string
	// RunsOn is the runner label for each job.
	// Defaults to "ubuntu-latest".
	RunsOn string
}

func (w *GitHubWorkflow) applyDefaults() {
	if w.Name == "" {
		w.Name = "CI"
	}
	if w.SetupAction == "" {
		w.SetupAction = "einride/sage/actions/setup@master"
	}
	if w.SetupWith == nil {
		w.SetupWith = map[string]string{"go-version-file": "go.mod"}
	}
	if w.RunsOn == "" {
		w.RunsOn = "ubuntu-latest"
	}
}

// workflowGroup is a PlanGroup whose Go target names have already been
// resolved to their corresponding Make target names.
type workflowGroup struct {
	Mode    PlanMode
	Targets []string
}

type workflowJob struct {
	Name   string
	Target string
	Needs  []string
}

type workflowSetupKV struct {
	Key   string
	Value string
}

type workflowData struct {
	Name        string
	RunsOn      string
	SetupAction string
	SetupWith   []workflowSetupKV
	Jobs        []workflowJob
}

//go:embed workflow_template.yml
var workflowTemplate string

// renderWorkflow renders a GitHub Actions workflow YAML from a sequence of
// resolved groups. Edges: within a serial group each job needs the previous
// one, and the first job of any group needs all jobs of the preceding group.
func renderWorkflow(cfg GitHubWorkflow, groups []workflowGroup) ([]byte, error) {
	cfg.applyDefaults()
	data := workflowData{
		Name:        cfg.Name,
		RunsOn:      cfg.RunsOn,
		SetupAction: cfg.SetupAction,
		SetupWith:   sortedSetupWith(cfg.SetupWith),
		Jobs:        buildJobs(groups),
	}
	tmpl, err := template.New("workflow").Parse(workflowTemplate)
	if err != nil {
		return nil, fmt.Errorf("sage: parse workflow template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("sage: execute workflow template: %w", err)
	}
	return buf.Bytes(), nil
}

func buildJobs(groups []workflowGroup) []workflowJob {
	var jobs []workflowJob
	var prev []string
	for _, g := range groups {
		for i, target := range g.Targets {
			job := workflowJob{Name: target, Target: target}
			switch g.Mode {
			case PlanModeSerial:
				if i == 0 {
					job.Needs = append(job.Needs, prev...)
				} else {
					job.Needs = []string{g.Targets[i-1]}
				}
			case PlanModeParallel:
				job.Needs = append(job.Needs, prev...)
			}
			jobs = append(jobs, job)
		}
		prev = g.Targets
	}
	return jobs
}

func sortedSetupWith(m map[string]string) []workflowSetupKV {
	kvs := make([]workflowSetupKV, 0, len(m))
	for k, v := range m {
		kvs = append(kvs, workflowSetupKV{Key: k, Value: v})
	}
	sort.Slice(kvs, func(i, j int) bool { return kvs[i].Key < kvs[j].Key })
	return kvs
}

// planToWorkflowGroups maps the Go-qualified target names recorded by plan
// mode onto their corresponding Make target names using the parsed sagefile
// package (so //sage:target overrides are honored).
func planToWorkflowGroups(pkg *doc.Package, plan []PlanGroup) ([]workflowGroup, error) {
	groups := make([]workflowGroup, 0, len(plan))
	for _, g := range plan {
		targets := make([]string, 0, len(g.Targets))
		for _, raw := range g.Targets {
			target, err := planTargetToMakeTarget(pkg, raw)
			if err != nil {
				return nil, err
			}
			targets = append(targets, target)
		}
		groups = append(groups, workflowGroup{Mode: g.Mode, Targets: targets})
	}
	return groups, nil
}

// planTargetToMakeTarget resolves a Go-qualified target name (e.g. "main.GoLint")
// to a Make target name (e.g. "go-lint").
func planTargetToMakeTarget(pkg *doc.Package, planName string) (string, error) {
	name := planName
	if i := strings.LastIndex(name, "."); i != -1 {
		name = name[i+1:]
	}
	f := findDocFunc(pkg, name)
	if f == nil {
		return "", fmt.Errorf("sage: workflow: target %q not found in sagefile package", planName)
	}
	return effectiveMakeTarget(f), nil
}
