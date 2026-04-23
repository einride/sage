package sg

import (
	"flag"
	"os"
	"reflect"
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
	got, err := renderWorkflow(GitHubWorkflow{}, groups)
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
			got := buildJobs(tt.groups)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildJobs mismatch\n got: %#v\nwant: %#v", got, tt.want)
			}
		})
	}
}
