package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAdd_TitleMaxLength_DefaultRejectsLong is the G-0102 binary-level
// seam test: a title over the kernel default (80 chars) is hard-rejected
// at the dispatcher seam, regardless of whether internal tests pass.
// Without this test, a regression in the dispatcher that fails to
// thread TitleMaxLength into AddOptions would still let long titles
// through.
func TestAdd_TitleMaxLength_DefaultRejectsLong(t *testing.T) {
	t.Parallel()
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

	// 150-char title — well over the 80-char default cap.
	longTitle := strings.Repeat("a", 150)
	out, err := runBin(t, root, binDir, nil,
		"add", "gap", "--title", longTitle)
	if err == nil {
		t.Fatalf("aiwf add gap (long title) succeeded but should have been rejected:\n%s", out)
	}
	if !strings.Contains(out, "title length") {
		t.Errorf("rejection output should name the offending dimension (title length):\n%s", out)
	}
	if !strings.Contains(out, "entities.title_max_length") {
		t.Errorf("rejection output should cite the config knob for discoverability:\n%s", out)
	}

	// And: no gap was created on disk — the verb aborted before commit.
	matches, _ := filepath.Glob(filepath.Join(root, "work", "gaps", "G-*.md"))
	if len(matches) != 0 {
		t.Errorf("aiwf add should not have created a gap file when title was rejected; found: %v", matches)
	}
}

// TestAdd_TitleMaxLength_AcceptsShort confirms the happy-path: a
// short title is accepted, a gap file lands on disk, and the
// frontmatter title matches what the operator passed. Pairs with the
// rejection test above to pin both arms of the seam.
func TestAdd_TitleMaxLength_AcceptsShort(t *testing.T) {
	t.Parallel()
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

	title := "Short and descriptive gap title"
	if out, err := runBin(t, root, binDir, nil,
		"add", "gap", "--title", title); err != nil {
		t.Fatalf("aiwf add gap (short title): %v\n%s", err, out)
	}

	matches, err := filepath.Glob(filepath.Join(root, "work", "gaps", "G-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("glob gap files: matches=%v err=%v", matches, err)
	}
	body, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read gap: %v", err)
	}
	if !strings.Contains(string(body), "title: "+title) {
		t.Errorf("frontmatter title missing or mangled in gap body:\n%s", body)
	}
}

// TestAdd_TitleMaxLength_ConfiguredOverride pins the configurability
// arc: a consumer who sets `entities.title_max_length: 30` in
// aiwf.yaml gets a tighter cap. Without this test, a regression where
// the dispatcher ignores the configured value and falls back to the
// default would still pass the default-path tests above.
func TestAdd_TitleMaxLength_ConfiguredOverride(t *testing.T) {
	t.Parallel()
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

	cfgPath := filepath.Join(root, "aiwf.yaml")
	cfgBytes, readErr := os.ReadFile(cfgPath)
	if readErr != nil {
		t.Fatalf("read aiwf.yaml: %v", readErr)
	}
	if writeErr := os.WriteFile(cfgPath, append(cfgBytes, []byte("\nentities:\n  title_max_length: 30\n")...), 0o644); writeErr != nil {
		t.Fatalf("rewrite aiwf.yaml: %v", writeErr)
	}

	// 50-char title — under the default 80, over the configured 30.
	title := strings.Repeat("a", 50)
	out, err := runBin(t, root, binDir, nil,
		"add", "gap", "--title", title)
	if err == nil {
		t.Fatalf("aiwf add gap (50-char title under configured cap 30) succeeded but should have been rejected:\n%s", out)
	}
	if !strings.Contains(out, "title length") {
		t.Errorf("rejection should cite title length:\n%s", out)
	}
}
