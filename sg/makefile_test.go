package sg

import (
	"go/ast"
	"go/doc"
	"go/token"
	"testing"
)

// makeDocFunc creates a doc.Func with the given name and raw doc comment lines.
// Each line should be a full comment line including the "//" prefix.
func makeDocFunc(name string, commentLines ...string) *doc.Func {
	f := &doc.Func{
		Name: name,
		Decl: &ast.FuncDecl{},
	}
	if len(commentLines) > 0 {
		cg := &ast.CommentGroup{}
		for _, line := range commentLines {
			cg.List = append(cg.List, &ast.Comment{
				Slash: token.NoPos,
				Text:  line,
			})
		}
		f.Decl.Doc = cg
	}
	return f
}

func TestGetMakeTargetOverride(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		function *doc.Func
		expected string
	}{
		{
			name:     "no annotation",
			function: makeDocFunc("BuildImageV2", "// BuildImageV2 builds the image."),
			expected: "",
		},
		{
			name:     "annotation only",
			function: makeDocFunc("BuildImageV2", "//sage:target build-image-v2"),
			expected: "build-image-v2",
		},
		{
			name:     "annotation with description",
			function: makeDocFunc("BuildImageV2", "//sage:target build-image-v2", "// BuildImageV2 builds the image."),
			expected: "build-image-v2",
		},
		{
			name:     "annotation after description",
			function: makeDocFunc("BuildImageV2", "// BuildImageV2 builds the image.", "//sage:target build-image-v2"),
			expected: "build-image-v2",
		},
		{
			name:     "nil decl",
			function: &doc.Func{Name: "Foo"},
			expected: "",
		},
		{
			name:     "no doc comment",
			function: makeDocFunc("Foo"),
			expected: "",
		},
		{
			name:     "extra whitespace around value",
			function: makeDocFunc("BuildImageV2", "//sage:target   build-image-v2  "),
			expected: "build-image-v2",
		},
		{
			name:     "space after slashes",
			function: makeDocFunc("BuildImageV2", "// sage:target build-image-v2"),
			expected: "build-image-v2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := getMakeTargetOverride(tt.function)
			if got != tt.expected {
				t.Errorf("getMakeTargetOverride() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestValidateMakeTargetOverride(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		value     string
		wantPanic bool
	}{
		{name: "valid kebab", value: "build-image-v2", wantPanic: false},
		{name: "valid with dots", value: "build.image", wantPanic: false},
		{name: "valid with underscore", value: "build_image", wantPanic: false},
		{name: "single char", value: "a", wantPanic: false},
		{name: "empty", value: "", wantPanic: true},
		{name: "starts with dot", value: ".hidden", wantPanic: true},
		{name: "starts with dash", value: "-flag", wantPanic: true},
		{name: "uppercase", value: "BuildImage", wantPanic: true},
		{name: "spaces", value: "build image", wantPanic: true},
		{name: "special chars", value: "build@image", wantPanic: true},
		{name: "no trailing dots", value: "build-image.", wantPanic: true},
		{name: "no trailing dashes", value: "build-image-", wantPanic: true},
		{name: "no trailing underscores", value: "build-image_", wantPanic: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			didPanic := false
			func() {
				defer func() {
					if r := recover(); r != nil {
						didPanic = true
					}
				}()
				validateMakeTargetOverride(tt.value)
			}()
			if didPanic != tt.wantPanic {
				t.Errorf("validateMakeTargetOverride(%q): panicked = %v, want %v", tt.value, didPanic, tt.wantPanic)
			}
		})
	}
}

func TestEffectiveMakeTarget(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		function *doc.Func
		expected string
	}{
		{
			name:     "with override",
			function: makeDocFunc("BuildImageV2", "//sage:target build-image-v2"),
			expected: "build-image-v2",
		},
		{
			name:     "without override",
			function: makeDocFunc("BuildImage"),
			expected: "build-image",
		},
		{
			name:     "without override digits",
			function: makeDocFunc("BuildImageV2"),
			expected: "build-image-v-2",
		},
		{
			name:     "nil decl falls back",
			function: &doc.Func{Name: "BuildImage"},
			expected: "build-image",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := effectiveMakeTarget(tt.function)
			if got != tt.expected {
				t.Errorf("effectiveMakeTarget() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestToMakeTarget(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected string
	}{
		{input: "BuildImage", expected: "build-image"},
		{input: "BuildImageV2", expected: "build-image-v-2"},
		{input: "JSONData", expected: "json-data"},
		{input: "Simple", expected: "simple"},
		{input: "Namespace:Function", expected: "function"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := toMakeTarget(tt.input)
			if got != tt.expected {
				t.Errorf("toMakeTarget(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
