package sg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindLocalReplaces(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "single-line replace",
			content: "module test\n\nreplace go.einride.tech/sage => ../\n",
			want:    []string{"../"},
		},
		{
			name:    "block replace",
			content: "module test\n\nreplace (\n\tgo.einride.tech/sage => ../\n)\n",
			want:    []string{"../"},
		},
		{
			name:    "multiple replaces",
			content: "module test\n\nreplace (\n\tgo.einride.tech/sage => ../\n\tgo.einride.tech/other => ./local\n)\n",
			want:    []string{"../", "./local"},
		},
		{
			name:    "remote replace skipped",
			content: "module test\n\nreplace go.einride.tech/sage => github.com/other/sage v1.0.0\n",
			want:    nil,
		},
		{
			name:    "comment with arrow skipped",
			content: "module test\n\n// old => ./local\n",
			want:    nil,
		},
		{
			name:    "no replaces",
			content: "module test\n\nrequire go.einride.tech/sage v0.400.0\n",
			want:    nil,
		},
		{
			name:    "replace with version suffix",
			content: "replace go.einride.tech/sage => ../sage v0.0.0\n",
			want:    []string{"../sage"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			gomod := filepath.Join(dir, "go.mod")
			if err := os.WriteFile(gomod, []byte(tt.content), 0o600); err != nil {
				t.Fatal(err)
			}
			got := findLocalReplaces(gomod)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
