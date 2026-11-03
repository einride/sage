package sggolangcilintv2

import (
	"path/filepath"
	"testing"
)

func TestShouldSkipDir(t *testing.T) {
	root := filepath.FromSlash("/repo")
	for _, tt := range []struct {
		name       string
		ignoreDirs []string
		path       string
		want       bool
	}{
		{name: "root", path: "/repo", want: false},
		{name: "module", path: "/repo/foo", want: false},
		{name: "dot dir not ignored", path: "/repo/.claude", want: false},
		{name: "ignored dir", ignoreDirs: []string{"foo/bar"}, path: "/repo/foo/bar", want: true},
		{name: "ignored dir trailing slash", ignoreDirs: []string{"foo/bar/"}, path: "/repo/foo/bar", want: true},
		{name: "parent of ignored dir", ignoreDirs: []string{"foo/bar"}, path: "/repo/foo", want: false},
		{name: "same name elsewhere", ignoreDirs: []string{"bar"}, path: "/repo/foo/bar", want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.FromSlash(tt.path)
			got, err := shouldSkipDir(Config{IgnoreDirs: tt.ignoreDirs}, root, path)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("shouldSkipDir(%v, %q, %q) = %v, want %v", tt.ignoreDirs, root, path, got, tt.want)
			}
		})
	}
}
