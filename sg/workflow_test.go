package sg

import (
	"context"
	"flag"
	"go/ast"
	"go/doc"
	"os"
	"reflect"
	"strings"
	"testing"
)

//nolint:gochecknoglobals
var updateGolden = flag.Bool("update-golden", false, "update golden test fixtures")

func TestRenderWorkflow_Golden(t *testing.T) {
	// Mirrors the shape of Sage's own Default target:
	//   sg.Deps(ctx, ConvcoCheck, GoLint, GoTest)
	//   sg.SerialDeps(ctx, GoModTidy)
	//   sg.SerialDeps(ctx, GoLicenses, GitVerifyNoDiff)
	groups := []workflowGroup{
		{Mode: PlanModeParallel, Targets: []string{"convco-check", "go-lint", "go-test"}},
		{Mode: PlanModeSerial, Targets: []string{"go-mod-tidy"}},
		{Mode: PlanModeSerial, Targets: []string{"go-licenses", "git-verify-no-diff"}},
	}
	got, err := renderWorkflow(GitHubWorkflow{}, groups, nil)
	if err != nil {
		t.Fatalf("renderWorkflow: %v", err)
	}
	goldenPath := "testdata/workflow_golden.yml"
	if *updateGolden {
		if err := os.WriteFile(goldenPath, got, 0o600); err != nil {
			t.Fatalf("update golden: %v", err)
		}
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("workflow YAML mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestBuildJobs(t *testing.T) {
	tests := []struct {
		name   string
		groups []workflowGroup
		want   []workflowJob
	}{
		{
			name:   "empty",
			groups: nil,
			want:   nil,
		},
		{
			name: "single parallel group",
			groups: []workflowGroup{
				{Mode: PlanModeParallel, Targets: []string{"a", "b"}},
			},
			want: []workflowJob{
				{Name: "a", Target: "a"},
				{Name: "b", Target: "b"},
			},
		},
		{
			name: "serial within group",
			groups: []workflowGroup{
				{Mode: PlanModeSerial, Targets: []string{"a", "b", "c"}},
			},
			want: []workflowJob{
				{Name: "a", Target: "a"},
				{Name: "b", Target: "b", Needs: []string{"a"}},
				{Name: "c", Target: "c", Needs: []string{"b"}},
			},
		},
		{
			name: "parallel then serial",
			groups: []workflowGroup{
				{Mode: PlanModeParallel, Targets: []string{"a", "b"}},
				{Mode: PlanModeSerial, Targets: []string{"c", "d"}},
			},
			want: []workflowJob{
				{Name: "a", Target: "a"},
				{Name: "b", Target: "b"},
				{Name: "c", Target: "c", Needs: []string{"a", "b"}},
				{Name: "d", Target: "d", Needs: []string{"c"}},
			},
		},
		{
			name: "parallel then parallel",
			groups: []workflowGroup{
				{Mode: PlanModeParallel, Targets: []string{"a", "b"}},
				{Mode: PlanModeParallel, Targets: []string{"c", "d"}},
			},
			want: []workflowJob{
				{Name: "a", Target: "a"},
				{Name: "b", Target: "b"},
				{Name: "c", Target: "c", Needs: []string{"a", "b"}},
				{Name: "d", Target: "d", Needs: []string{"a", "b"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildJobs(GitHubWorkflow{}, tt.groups, nil)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildJobs mismatch\n got: %#v\nwant: %#v", got, tt.want)
			}
		})
	}
}

func TestBuildJobs_Overrides(t *testing.T) {
	cfg := GitHubWorkflow{
		RunsOn:      "ubuntu-latest",
		SetupAction: "./actions/setup",
		SetupWith:   map[string]string{"go-version-file": "go.mod"},
	}
	groups := []workflowGroup{
		{Mode: PlanModeParallel, Targets: []string{"go-test", "go-lint"}},
	}
	overrides := map[string]JobOverride{
		"go-test": {
			RunsOn:    "ubuntu-24.04",
			SetupWith: map[string]string{"go-version-file": "backend/go.mod", "cache": "true"},
		},
	}
	got := buildJobs(cfg, groups, overrides)
	want := []workflowJob{
		{
			Name: "go-test", Target: "go-test",
			RunsOn:      "ubuntu-24.04",
			SetupAction: "./actions/setup",
			SetupWith: []workflowSetupKV{
				{Key: "cache", Value: "true"},
				{Key: "go-version-file", Value: "backend/go.mod"},
			},
		},
		{
			Name: "go-lint", Target: "go-lint",
			RunsOn:      "ubuntu-latest",
			SetupAction: "./actions/setup",
			SetupWith: []workflowSetupKV{
				{Key: "go-version-file", Value: "go.mod"},
			},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildJobs overrides mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

// workflowTestPlain is referenced by override-resolution tests via its
// function pointer so runtime.FuncForPC resolves its name.
func workflowTestPlain(_ context.Context) error     { return nil }
func workflowTestUnreached(_ context.Context) error { return nil }

// workflowTestNamespace is used to exercise namespace-aware resolution.
type workflowTestNamespace Namespace

func (workflowTestNamespace) All(_ context.Context) error   { return nil }
func (workflowTestNamespace) Other(_ context.Context) error { return nil }

func TestSplitPlanName(t *testing.T) {
	tests := []struct {
		in       string
		wantPkg  string
		wantName string
	}{
		{"main.GoLint", "main", "GoLint"},
		{"main.Proto.All", "main", "Proto.All"},
		{"github.com/foo/bar.GoLint", "github.com/foo/bar", "GoLint"},
		{"github.com/foo/bar.Type.Method", "github.com/foo/bar", "Type.Method"},
		{"loose", "", "loose"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			gotPkg, gotName := splitPlanName(tt.in)
			if gotPkg != tt.wantPkg || gotName != tt.wantName {
				t.Errorf("splitPlanName(%q) = (%q, %q); want (%q, %q)",
					tt.in, gotPkg, gotName, tt.wantPkg, tt.wantName)
			}
		})
	}
}

func TestBuildNamespaceProxies(t *testing.T) {
	mks := []Makefile{
		{Path: "/repo/Makefile", DefaultTarget: workflowTestPlain},
		{
			Path:          "/repo/ns/Makefile",
			DefaultTarget: workflowTestNamespace.All,
			Namespace:     workflowTestNamespace{},
		},
	}
	got, err := buildNamespaceProxies(mks)
	if err != nil {
		t.Fatalf("buildNamespaceProxies: %v", err)
	}
	want := map[string]string{
		"go.einride.tech/sage/sg.workflowTestNamespace.All": "workflow-test-namespace",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildNamespaceProxies mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestPlanTargetToMakeTarget_Namespaces(t *testing.T) {
	pkg := &doc.Package{
		Funcs: []*doc.Func{
			{Name: "workflowTestPlain", Decl: &ast.FuncDecl{}},
		},
	}
	namespaces := map[string]string{
		"go.einride.tech/sage/sg.workflowTestNamespace.All": "workflow-test-namespace",
	}
	tests := []struct {
		name       string
		planName   string
		want       string
		wantErrSub string
	}{
		{
			name:     "namespace default maps to proxy",
			planName: "go.einride.tech/sage/sg.workflowTestNamespace.All",
			want:     "workflow-test-namespace",
		},
		{
			name:       "namespace non-default rejected",
			planName:   "go.einride.tech/sage/sg.workflowTestNamespace.Other",
			wantErrSub: "namespace method",
		},
		{
			name:     "top-level target resolves via doc.Func",
			planName: "go.einride.tech/sage/sg.workflowTestPlain",
			want:     "workflow-test-plain",
		},
		{
			name:       "unknown top-level target errors",
			planName:   "go.einride.tech/sage/sg.workflowTestMissing",
			wantErrSub: "not found in sagefile package",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := planTargetToMakeTarget(pkg, namespaces, tt.planName)
			if tt.wantErrSub != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErrSub)
				}
				if !strings.Contains(err.Error(), tt.wantErrSub) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.wantErrSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveJobOverrides(t *testing.T) {
	pkg := &doc.Package{
		Funcs: []*doc.Func{
			{Name: "workflowTestPlain", Decl: &ast.FuncDecl{}},
			{Name: "workflowTestUnreached", Decl: &ast.FuncDecl{}},
		},
	}
	groups := []workflowGroup{
		{Mode: PlanModeParallel, Targets: []string{"workflow-test-plain"}},
	}
	tests := []struct {
		name       string
		overrides  []JobOverride
		wantKeys   []string
		wantErrSub string
	}{
		{
			name:     "no overrides",
			wantKeys: nil,
		},
		{
			name: "valid override",
			overrides: []JobOverride{
				{Target: workflowTestPlain, RunsOn: "ubuntu-24.04"},
			},
			wantKeys: []string{"workflow-test-plain"},
		},
		{
			name: "nil target",
			overrides: []JobOverride{
				{Target: nil},
			},
			wantErrSub: "target is nil",
		},
		{
			name: "non-function target",
			overrides: []JobOverride{
				{Target: 42},
			},
			wantErrSub: "target is not a function",
		},
		{
			name: "unreached target",
			overrides: []JobOverride{
				{Target: workflowTestUnreached},
			},
			wantErrSub: "is not reached by the default target",
		},
		{
			name: "duplicate override",
			overrides: []JobOverride{
				{Target: workflowTestPlain, RunsOn: "ubuntu-24.04"},
				{Target: workflowTestPlain, RunsOn: "ubuntu-latest"},
			},
			wantErrSub: "duplicate override",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveJobOverrides(pkg, nil, groups, tt.overrides)
			if tt.wantErrSub != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErrSub)
				}
				if !strings.Contains(err.Error(), tt.wantErrSub) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.wantErrSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var gotKeys []string
			for k := range got {
				gotKeys = append(gotKeys, k)
			}
			if !reflect.DeepEqual(gotKeys, tt.wantKeys) {
				t.Errorf("keys mismatch\n got: %v\nwant: %v", gotKeys, tt.wantKeys)
			}
		})
	}
}
