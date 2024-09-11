package sgcspevaluatorcli_test

import (
	"strings"
	"testing"

	"go.einride.tech/sage/tools/sgcspevaluatorcli"
)

func TestContentSecurityPolicy_Clone(t *testing.T) {
	var policy sgcspevaluatorcli.ContentSecurityPolicy = map[string][]string{
		"directive": {"source"},
	}
	cloned := policy.Clone()

	cloned.With("new directive", "source one")
	cloned.With("directive", "source two")

	if len(policy) != 1 {
		t.Fatalf("changing cloned policy should not change source policy")
	}

	assertSources(t, policy, "directive", "source")
}

func TestContentSecurityPolicy_With(t *testing.T) {
	policy := new(sgcspevaluatorcli.ContentSecurityPolicy).
		With("directive", "source").
		With("directive", "source")

	assertSources(t, policy, "directive", "source")
}

func TestContentSecurityPolicy_String(t *testing.T) {
	policy := new(sgcspevaluatorcli.ContentSecurityPolicy).
		With("directive", "source").
		With("another directive", "source1", "source2")

	expected := "another directive source1 source2; directive source"
	if got := policy.String(); got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestContentSecurityPolicy_Merge(t *testing.T) {
	policy1 := new(sgcspevaluatorcli.ContentSecurityPolicy).
		With("directive", "source")
	policy2 := new(sgcspevaluatorcli.ContentSecurityPolicy).
		With("directive", "source1").
		With("directive2", "source")

	merged := policy1.Merge(policy2)

	if len(merged) != 2 {
		t.Fatalf("expected merged to have 2 directved")
	}

	assertSources(t, merged, "directive", "source", "source1")
	assertSources(t, merged, "directive2", "source")
}

func assertDirectiveExists(t *testing.T, policy sgcspevaluatorcli.ContentSecurityPolicy, directive string) {
	t.Helper()

	if _, ok := policy[directive]; !ok {
		t.Fatalf("directive %q is missing", directive)
	}
}

func assertSources(t *testing.T, policy sgcspevaluatorcli.ContentSecurityPolicy, directive string, sources ...string) {
	t.Helper()

	assertDirectiveExists(t, policy, directive)

	if len(policy[directive]) != len(sources) {
		t.Fatalf("unexpected number of sources: got %d, expected %d", len(policy[directive]), len(sources))
	}

	expected := make(map[string]bool, len(sources))
	for _, source := range sources {
		expected[source] = true
	}

	missingSources := make([]string, 0, len(sources))
	for _, source := range policy[directive] {
		if !expected[source] {
			missingSources = append(missingSources, source)
		}
	}

	if len(missingSources) != 0 {
		t.Fatalf("directive %q missing sources %q", directive, strings.Join(missingSources, ", "))
	}
}
