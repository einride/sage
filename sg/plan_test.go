package sg

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func planTargetA(_ context.Context) error { return nil }
func planTargetB(_ context.Context) error { return nil }
func planTargetC(_ context.Context) error { return nil }

func planTargetWithArg(_ context.Context, _ string) error { return nil }

func TestPlanRecorder(t *testing.T) {
	tests := []struct {
		name string
		call func(ctx context.Context)
		want []PlanGroup
	}{
		{
			name: "single parallel group",
			call: func(ctx context.Context) {
				Deps(ctx, planTargetA, planTargetB)
			},
			want: []PlanGroup{
				{Mode: PlanModeParallel, Targets: []string{
					"go.einride.tech/sage/sg.planTargetA",
					"go.einride.tech/sage/sg.planTargetB",
				}},
			},
		},
		{
			name: "single serial group",
			call: func(ctx context.Context) {
				SerialDeps(ctx, planTargetA, planTargetB)
			},
			want: []PlanGroup{
				{Mode: PlanModeSerial, Targets: []string{
					"go.einride.tech/sage/sg.planTargetA",
					"go.einride.tech/sage/sg.planTargetB",
				}},
			},
		},
		{
			name: "parallel then serial",
			call: func(ctx context.Context) {
				Deps(ctx, planTargetA, planTargetB)
				SerialDeps(ctx, planTargetC)
			},
			want: []PlanGroup{
				{Mode: PlanModeParallel, Targets: []string{
					"go.einride.tech/sage/sg.planTargetA",
					"go.einride.tech/sage/sg.planTargetB",
				}},
				{Mode: PlanModeSerial, Targets: []string{
					"go.einride.tech/sage/sg.planTargetC",
				}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "plan.jsonl")
			t.Setenv(PlanOutputEnv, path)
			tt.call(context.Background())
			got, err := ReadPlan(path)
			if err != nil {
				t.Fatalf("ReadPlan: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("plan mismatch\n got: %#v\nwant: %#v", got, tt.want)
			}
		})
	}
}

func TestPlanRecorder_ArgCarrierRejected(t *testing.T) {
	target := Fn(planTargetWithArg, "x")
	err := recordPlanGroup(PlanModeParallel, []Target{target})
	if err == nil {
		t.Fatal("expected error for target with arguments, got nil")
	}
	if !strings.Contains(err.Error(), "plan mode does not support targets with arguments") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPlanRecorder_Inactive(t *testing.T) {
	t.Setenv(PlanOutputEnv, "")
	if planRecording() {
		t.Error("planRecording should be false when SAGE_PLAN_OUTPUT is unset")
	}
}
