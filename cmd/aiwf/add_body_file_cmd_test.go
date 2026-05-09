package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestAdd_BodyFile_BinaryEndToEnd is the M-056/AC-5 closure: drive
// the dispatcher seam against a real binary and a real consumer
// repo. Without this test, a regression that drops --body-file from
// the dispatcher (parses the flag but never threads BodyOverride
// into AddOptions) would still pass internal/verb tests.
func TestAdd_BodyFile_BinaryEndToEnd(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}

	bodyText := "## Goal\n\nFleshed-out goal prose, not the empty template.\n\n## Scope\n\nThe scope.\n"
	bodyPath := filepath.Join(root, "epic-body.md")
	if err := os.WriteFile(bodyPath, []byte(bodyText), 0o644); err != nil {
		t.Fatalf("write body file: %v", err)
	}

	out, err := runBin(t, root, binDir, nil,
		"add", "epic", "--title", "Body-file epic", "--body-file", bodyPath)
	if err != nil {
		t.Fatalf("aiwf add --body-file: %v\n%s", err, out)
	}

	matches, err := filepath.Glob(filepath.Join(root, "work", "epics", "E-*", "epic.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("glob epic.md: matches=%v err=%v", matches, err)
	}
	got, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read epic.md: %v", err)
	}
	if !strings.Contains(string(got), "Fleshed-out goal prose") {
		t.Errorf("epic.md missing user body content:\n%s", got)
	}

	// Frontmatter still rendered correctly — the body wasn't
	// concatenated raw onto the file.
	if !strings.Contains(string(got), "id: E-0001") {
		t.Errorf("epic.md missing serialized frontmatter:\n%s", got)
	}
}

// TestAdd_BodyFile_StdinEndToEnd: --body-file - reads body content
// from stdin, so callers can pipe text without an intermediate
// file. The runBin helper doesn't pipe stdin, so this test invokes
// exec.Command directly with a configured Stdin reader.
func TestAdd_BodyFile_StdinEndToEnd(t *testing.T) {
	bin := aiwfBinary(t)

	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runBin(t, root, filepath.Dir(bin), nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}

	bodyText := "## Goal\n\nBody from stdin pipe.\n"
	cmd := exec.Command(bin, "add", "gap", "--title", "Stdin gap", "--body-file", "-")
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=aiwf-test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=aiwf-test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	cmd.Stdin = strings.NewReader(bodyText)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("aiwf add gap --body-file -: %v\n%s", err, out)
	}

	matches, err := filepath.Glob(filepath.Join(root, "work", "gaps", "G-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("glob G-*.md: matches=%v err=%v", matches, err)
	}
	got, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read gap: %v", err)
	}
	if !strings.Contains(string(got), "Body from stdin pipe") {
		t.Errorf("gap.md missing stdin body content:\n%s", got)
	}
}

// TestAdd_BodyFile_RefusesFrontmatter_BinaryEndToEnd: the dispatcher
// passes through to the verb-side rule check; a body file with its
// own frontmatter exits non-zero and the entity is never created.
func TestAdd_BodyFile_RefusesFrontmatter_BinaryEndToEnd(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}

	bad := "---\nid: PRETEND-1\n---\n\nbody\n"
	bodyPath := filepath.Join(root, "bad-body.md")
	if err := os.WriteFile(bodyPath, []byte(bad), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	out, err := runBin(t, root, binDir, nil,
		"add", "epic", "--title", "Should fail", "--body-file", bodyPath)
	if err == nil {
		t.Fatalf("expected refusal; got:\n%s", out)
	}
	if !strings.Contains(out, "frontmatter delimiter") {
		t.Errorf("expected frontmatter-delimiter message; got:\n%s", out)
	}

	// No epic was created.
	matches, _ := filepath.Glob(filepath.Join(root, "work", "epics", "E-*", "epic.md"))
	if len(matches) != 0 {
		t.Errorf("entity created despite refusal: %v", matches)
	}
}
