package sg

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// PlanOutputEnv, when set to a file path, puts the Sage runtime in "plan mode".
// In plan mode, calls to Deps and SerialDeps record the target names they would
// have executed instead of actually running them, and append the recorded group
// as a single JSON object (one per line, JSONL) to the file at the given path.
//
// Plan mode exists so that the generator in GenerateMakefiles can invoke the
// compiled sagefile binary, observe the call graph of the default target, and
// translate that graph into an equivalent GitHub Actions workflow.
const PlanOutputEnv = "SAGE_PLAN_OUTPUT"

// PlanMode identifies how the targets in a PlanGroup relate to each other.
type PlanMode string

const (
	// PlanModeParallel records a call to Deps: targets are independent of one another.
	PlanModeParallel PlanMode = "parallel"
	// PlanModeSerial records a call to SerialDeps: targets must run one after the other.
	PlanModeSerial PlanMode = "serial"
)

// PlanGroup is a single recorded group of targets from a Deps or SerialDeps call.
type PlanGroup struct {
	Mode    PlanMode `json:"mode"`
	Targets []string `json:"targets"`
}

//nolint:gochecknoglobals
var planMu sync.Mutex

// planOutputPath returns the file path plan mode writes to, or "" if plan mode is inactive.
func planOutputPath() string {
	return os.Getenv(PlanOutputEnv)
}

// planRecording reports whether plan mode is active.
func planRecording() bool {
	return planOutputPath() != ""
}

// argCarrier is implemented by Target values that carry user-supplied arguments.
// Plan mode refuses to record such targets because the generated Makefile targets
// are invoked without arguments (`make <target>`), so args would never make it
// through the generated GitHub Actions workflow.
type argCarrier interface {
	hasArgs() bool
}

// recordPlanGroup appends a PlanGroup to the plan output file.
// Returns an error if any target carries arguments.
func recordPlanGroup(mode PlanMode, targets []Target) error {
	names := make([]string, 0, len(targets))
	for _, t := range targets {
		if ac, ok := t.(argCarrier); ok && ac.hasArgs() {
			return fmt.Errorf(
				"sage: plan mode does not support targets with arguments (%s); "+
					"the default target must only compose parameter-less functions via sg.Deps/sg.SerialDeps",
				t.Name(),
			)
		}
		names = append(names, t.Name())
	}
	planMu.Lock()
	defer planMu.Unlock()
	f, err := os.OpenFile(planOutputPath(), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return fmt.Errorf("sage: open plan output file: %w", err)
	}
	if err := json.NewEncoder(f).Encode(PlanGroup{Mode: mode, Targets: names}); err != nil {
		_ = f.Close()
		return fmt.Errorf("sage: write plan group: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("sage: close plan output file: %w", err)
	}
	return nil
}

// ReadPlan parses the JSONL plan file at path and returns the recorded groups in order.
func ReadPlan(path string) ([]PlanGroup, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("sage: open plan file: %w", err)
	}
	defer f.Close()
	var groups []PlanGroup
	dec := json.NewDecoder(f)
	for dec.More() {
		var g PlanGroup
		if err := dec.Decode(&g); err != nil {
			return nil, fmt.Errorf("sage: decode plan group: %w", err)
		}
		groups = append(groups, g)
	}
	return groups, nil
}
