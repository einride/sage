package sggit_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"go.einride.tech/sage/tools/sggit"
)

// initRepo creates a temporary git repo with an initial commit and returns
// a context whose working directory is set to the repo root.
func initRepo(t *testing.T) (string, context.Context) {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("commit", "--allow-empty", "-m", "initial")

	// sg.Command uses FromGitRoot to set cmd.Dir, so we need to run
	// tests from inside the temp repo.
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	return dir, context.Background()
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyNoDiff_CleanRepo(t *testing.T) {
	_, ctx := initRepo(t)
	if err := sggit.VerifyNoDiff(ctx); err != nil {
		t.Fatalf("expected no error on clean repo, got: %v", err)
	}
}

func TestVerifyNoDiff_DirtyRepo(t *testing.T) {
	dir, ctx := initRepo(t)
	writeFile(t, dir, "dirty.txt", "hello")
	if err := sggit.VerifyNoDiff(ctx); err == nil {
		t.Fatal("expected error on dirty repo, got nil")
	}
}

func TestVerifyNoDiff_PathspecScopesToMatchingFiles(t *testing.T) {
	dir, ctx := initRepo(t)
	writeFile(t, dir, "README.md", "# hello")
	writeFile(t, dir, "Makefile", "all:")

	// Scoped to *.md — should detect the dirty .md file.
	if err := sggit.VerifyNoDiff(ctx, "*.md"); err == nil {
		t.Fatal("expected error for dirty .md file, got nil")
	}

	// Scoped to *.go — should pass since no .go files are dirty.
	if err := sggit.VerifyNoDiff(ctx, "*.go"); err != nil {
		t.Fatalf("expected no error for clean .go pathspec, got: %v", err)
	}
}

func TestVerifyNoDiff_PathspecExclude(t *testing.T) {
	dir, ctx := initRepo(t)
	writeFile(t, dir, "Makefile", "all:")
	writeFile(t, dir, "README.md", "# hello")

	// Exclude Makefile — should still detect dirty README.md.
	if err := sggit.VerifyNoDiff(ctx, ":(exclude)Makefile"); err == nil {
		t.Fatal("expected error when only Makefile is excluded, got nil")
	}

	// Exclude both files — should pass.
	if err := sggit.VerifyNoDiff(ctx, ":(exclude)Makefile", ":(exclude)README.md"); err != nil {
		t.Fatalf("expected no error when all dirty files excluded, got: %v", err)
	}
}
