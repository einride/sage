package sgcspevaluatorcli

import (
	"fmt"
	"slices"
	"strings"
)

// ContentSecurityPolicy is a map where each key is a Content Security Policy directive,
// and value is a list of sources for the respective directive.
type ContentSecurityPolicy map[string][]string

// String returns ContentSecurityPolicy ready to be send as a HTTP header.
func (csp ContentSecurityPolicy) String() string {
	directives := make([]string, 0, len(csp))
	for directive, rules := range csp {
		slices.Sort(rules)
		directives = append(directives, fmt.Sprintf("%s %s", directive, strings.Join(rules, " ")))
	}
	slices.Sort(directives)
	return strings.Join(directives, "; ")
}

// Clone returns a deep copy of the content security policy.
func (csp ContentSecurityPolicy) Clone() ContentSecurityPolicy {
	cloned := map[string][]string{}
	for key, values := range csp {
		cloned[key] = make([]string, len(values))
		copy(cloned[key], values)
	}
	return cloned
}

// With returns a new content security policy with an additional directive and sources.
//
// Note that the _new_ policy is returned here, so changing the returned value from this function will
// not change original policy.
func (csp ContentSecurityPolicy) With(directive, source string, sources ...string) ContentSecurityPolicy {
	clone := csp.Clone()
	clone[directive] = appendUnique(clone[directive], append(sources, source))
	return clone
}

// Merge returns a new union policy.
func (csp ContentSecurityPolicy) Merge(other ...ContentSecurityPolicy) ContentSecurityPolicy {
	var merged ContentSecurityPolicy = map[string][]string{}
	for _, policy := range append(other, csp) {
		for directive, sources := range policy {
			switch len(sources) {
			case 0:
			case 1:
				merged = merged.With(directive, sources[0])
			default:
				merged = merged.With(directive, sources[0], sources[1:]...)
			}
		}
	}
	return merged
}

func appendUnique(ll ...[]string) []string {
	set := map[string]struct{}{}
	for _, l := range ll {
		for _, item := range l {
			set[item] = struct{}{}
		}
	}
	return sortedKeys(set)
}

func sortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
