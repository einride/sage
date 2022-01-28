package main

import (
	"bytes"
	"os"
	"testing"
)

func Test_deps(t *testing.T) {
	modfile, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(modfile, []byte("require")) {
		t.Fatal("go.mod must not contain any 3rd party dependencies")
	}
}
