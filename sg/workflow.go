package sg

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/doc"
	"reflect"
	"runtime"
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
	// `make <target>` step. Applied identically to every job unless a
	// JobOverride specifies its own SetupWith (in which case the
	// workflow-wide map and the per-job map are merged, with per-job
	// entries taking precedence).
	// Defaults to {"go-version-file": "go.mod"}.
	SetupWith map[string]string
	// RunsOn is the runner label for each job.
	// Defaults to "ubuntu-latest".
	RunsOn string
	// JobOverrides optionally customizes individual jobs. Each entry's
	// Target is a reference to the sagefile function (e.g. GoTest) whose
	// generated job should be customized; the function must be reached by
	// the default target via sg.Deps or sg.SerialDeps. Unknown targets and
	// duplicate entries cause GenerateMakefiles to fail.
	JobOverrides []JobOverride
}

// JobOverride customizes the generated job for a single target.
// Unset fields inherit the workflow-wide value from GitHubWorkflow.
type JobOverride struct {
	// Target is a reference to the sagefile function whose generated job
	// should be customized, e.g. GoTest. Passing the function itself (not
	// its name as a string) means the reference is type-checked by the Go
	// compiler and survives renames via LSP refactors.
	Target any
	// RunsOn overrides GitHubWorkflow.RunsOn for this job.
	RunsOn string
	// SetupAction overrides GitHubWorkflow.SetupAction for this job.
	SetupAction string
	// SetupWith is merged over GitHubWorkflow.SetupWith for this job;
	// entries here take precedence.
	SetupWith map[string]string
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
	Name        string
	Target      string
	Needs       []string
	RunsOn      string
	SetupAction string
	SetupWith   []workflowSetupKV
}

type workflowSetupKV struct {
	Key   string
	Value string
}

type workflowData struct {
	Name string
	Jobs []workflowJob
}

//go:embed workflow_template.yml
var workflowTemplate string

// renderWorkflow renders a GitHub Actions workflow YAML from a sequence of
// resolved groups and per-target overrides keyed by Make target name. Edges:
// within a serial group each job needs the previous one, and the first job of
// any group needs all jobs of the preceding group.
func renderWorkflow(cfg GitHubWorkflow, groups []workflowGroup, overrides map[string]JobOverride) ([]byte, error) {
	cfg.applyDefaults()
	data := workflowData{
		Name: cfg.Name,
		Jobs: buildJobs(cfg, groups, overrides),
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

func buildJobs(cfg GitHubWorkflow, groups []workflowGroup, overrides map[string]JobOverride) []workflowJob {
	var jobs []workflowJob
	var prev []string
	for _, g := range groups {
		for i, target := range g.Targets {
			ov := overrides[target]
			runsOn := cfg.RunsOn
			if ov.RunsOn != "" {
				runsOn = ov.RunsOn
			}
			setupAction := cfg.SetupAction
			if ov.SetupAction != "" {
				setupAction = ov.SetupAction
			}
			job := workflowJob{
				Name:        target,
				Target:      target,
				RunsOn:      runsOn,
				SetupAction: setupAction,
				SetupWith:   sortedSetupWith(mergeSetupWith(cfg.SetupWith, ov.SetupWith)),
			}
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

// mergeSetupWith returns a new map containing base's entries overlaid by
// override's entries. Either input may be nil.
func mergeSetupWith(base, override map[string]string) map[string]string {
	merged := make(map[string]string, len(base)+len(override))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range override {
		merged[k] = v
	}
	return merged
}

func sortedSetupWith(m map[string]string) []workflowSetupKV {
	if len(m) == 0 {
		return nil
	}
	kvs := make([]workflowSetupKV, 0, len(m))
	for k, v := range m {
		kvs = append(kvs, workflowSetupKV{Key: k, Value: v})
	}
	sort.Slice(kvs, func(i, j int) bool { return kvs[i].Key < kvs[j].Key })
	return kvs
}

// planToWorkflowGroups maps the Go-qualified target names recorded by plan
// mode onto their corresponding Make target names using the parsed sagefile
// package (so //sage:target overrides are honored) and the namespace proxy
// map (so namespaced default targets resolve to their root proxy target).
func planToWorkflowGroups(pkg *doc.Package, namespaces map[string]string, plan []PlanGroup) ([]workflowGroup, error) {
	groups := make([]workflowGroup, 0, len(plan))
	for _, g := range plan {
		targets := make([]string, 0, len(g.Targets))
		for _, raw := range g.Targets {
			target, err := planTargetToMakeTarget(pkg, namespaces, raw)
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
// to a Make target name (e.g. "go-lint"). If the name matches a namespace
// Makefile's DefaultTarget (see buildNamespaceProxies), it returns the root
// Makefile's proxy target for that namespace instead (e.g. "proto" for
// "main.Proto.All").
func planTargetToMakeTarget(pkg *doc.Package, namespaces map[string]string, planName string) (string, error) {
	if proxy, ok := namespaces[planName]; ok {
		return proxy, nil
	}
	_, namePart := splitPlanName(planName)
	if strings.Contains(namePart, ".") {
		return "", fmt.Errorf(
			"sage: workflow: target %q appears to be a namespace method that is not any namespace "+
				"Makefile's DefaultTarget; only namespace default targets can be composed via "+
				"sg.Deps in the root default - either wrap it in a top-level function or compose "+
				"the namespace's default method (e.g. Proto.All)",
			planName,
		)
	}
	f := findDocFunc(pkg, namePart)
	if f == nil {
		return "", fmt.Errorf("sage: workflow: target %q not found in sagefile package", planName)
	}
	return effectiveMakeTarget(f), nil
}

// splitPlanName splits a plan-style runtime name into its package path and
// in-package name. For "main.GoLint" this returns ("main", "GoLint"); for
// "main.Proto.All" it returns ("main", "Proto.All"); for
// "github.com/foo/bar.GoLint" it returns ("github.com/foo/bar", "GoLint").
// The separator is the first "." after the last "/".
func splitPlanName(planName string) (pkg, name string) {
	start := 0
	if i := strings.LastIndex(planName, "/"); i != -1 {
		start = i + 1
	}
	if i := strings.Index(planName[start:], "."); i != -1 {
		return planName[:start+i], planName[start+i+1:]
	}
	return "", planName
}

// funcPlanName returns the plan-style runtime name of a function reference,
// mirroring the name-stripping applied by the plan recorder so lookups match.
func funcPlanName(target any) (string, error) {
	if target == nil {
		return "", fmt.Errorf("target is nil")
	}
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Func {
		return "", fmt.Errorf("target is not a function: %T", target)
	}
	return trimRuntimeFuncName(runtime.FuncForPC(v.Pointer()).Name()), nil
}

// targetRefToMakeTarget resolves a function reference to its Make target name.
func targetRefToMakeTarget(pkg *doc.Package, namespaces map[string]string, target any) (string, error) {
	name, err := funcPlanName(target)
	if err != nil {
		return "", err
	}
	return planTargetToMakeTarget(pkg, namespaces, name)
}

// buildNamespaceProxies maps the plan-style name of each namespace Makefile's
// DefaultTarget (e.g. "main.Proto.All") to the root Makefile's proxy target
// for that namespace (e.g. "proto"). At runtime the proxy target delegates to
// the nested Makefile via `make -C <dir>`, so a workflow job that invokes
// `make proto` is equivalent to invoking Proto.All from the root.
func buildNamespaceProxies(mks []Makefile) (map[string]string, error) {
	m := make(map[string]string)
	for _, mk := range mks {
		if mk.namespaceName() == "" || mk.DefaultTarget == nil {
			continue
		}
		name, err := funcPlanName(mk.DefaultTarget)
		if err != nil {
			return nil, fmt.Errorf("sage: workflow: namespace Makefile %q: %w", mk.namespaceName(), err)
		}
		m[name] = toMakeTarget(mk.namespaceName())
	}
	return m, nil
}

// resolveJobOverrides validates a list of JobOverride entries against the
// plan's recorded Make targets and returns them keyed by Make target name.
// Returns an error if any target is unknown, unreachable from the default
// target, or specified more than once.
func resolveJobOverrides(
	pkg *doc.Package,
	namespaces map[string]string,
	groups []workflowGroup,
	overrides []JobOverride,
) (map[string]JobOverride, error) {
	known := make(map[string]bool)
	for _, g := range groups {
		for _, t := range g.Targets {
			known[t] = true
		}
	}
	result := make(map[string]JobOverride, len(overrides))
	for _, ov := range overrides {
		target, err := targetRefToMakeTarget(pkg, namespaces, ov.Target)
		if err != nil {
			return nil, fmt.Errorf("GitHubWorkflow.JobOverrides: %w", err)
		}
		if !known[target] {
			return nil, fmt.Errorf(
				"GitHubWorkflow.JobOverrides: target %q is not reached by the default target; "+
					"ensure it is invoked via sg.Deps or sg.SerialDeps",
				target,
			)
		}
		if _, dup := result[target]; dup {
			return nil, fmt.Errorf("GitHubWorkflow.JobOverrides: duplicate override for target %q", target)
		}
		result[target] = ov
	}
	return result, nil
}
