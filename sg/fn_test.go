package sg

import (
	"context"
	"testing"
)

func TestFn_Name(t *testing.T) {
	ns := namespace{}
	for _, tt := range []struct {
		name     string
		fn       interface{}
		expected string
	}{
		{
			name:     "func",
			fn:       MyFunc,
			expected: "go.einride.tech/sage/sg.MyFunc",
		},
		{
			name:     "anonymous",
			fn:       func(_ context.Context) error { return nil },
			expected: "go.einride.tech/sage/sg.TestFn_Name.func1",
		},
		{
			name:     "namespace",
			fn:       namespace.MyFunc,
			expected: "go.einride.tech/sage/sg.namespace.MyFunc",
		},
		{
			name:     "namespace value",
			fn:       ns.MyFunc,
			expected: "go.einride.tech/sage/sg.namespace.MyFunc",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fn := Fn(tt.fn)
			got := fn.Name()
			if got != tt.expected {
				t.Fatalf("expected %q to be %q", got, tt.expected)
			}
		})
	}
}

func MyFunc(_ context.Context) error {
	return nil
}

type namespace Namespace

func (namespace) MyFunc(_ context.Context) error {
	return nil
}
