package sg

import (
	"reflect"
	"testing"
)

func Test_prependPath(t *testing.T) {
	for _, tt := range []struct {
		name     string
		environ  []string
		paths    []string
		expected []string
	}{
		{
			name:     "no PATH, single path",
			environ:  []string{"FOO=bar"},
			paths:    []string{"/bin"},
			expected: []string{"FOO=bar", "PATH=/bin"},
		},

		{
			name:     "no PATH, multiple paths",
			environ:  []string{"FOO=bar"},
			paths:    []string{"/bin", "/usr/bin"},
			expected: []string{"FOO=bar", "PATH=/bin:/usr/bin"},
		},

		{
			name:     "existing PATH, single path",
			environ:  []string{"FOO=bar", "PATH=/bin"},
			paths:    []string{"/usr/bin"},
			expected: []string{"FOO=bar", "PATH=/usr/bin:/bin"},
		},

		{
			name:     "existing PATH, multiple paths",
			environ:  []string{"FOO=bar", "PATH=/bin"},
			paths:    []string{"/usr/bin", "/usr/local/bin"},
			expected: []string{"FOO=bar", "PATH=/usr/bin:/usr/local/bin:/bin"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			actual := prependPath(tt.environ, tt.paths...)
			if !reflect.DeepEqual(tt.expected, actual) {
				t.Errorf("expected %v but got %v", tt.expected, actual)
			}
		})
	}
}
