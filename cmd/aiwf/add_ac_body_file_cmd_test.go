package main

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/gitops"
)

// TestAddAC_BodyFile_BinaryEndToEnd is the M-067/AC-1 closure: drive
// the dispatcher seam against a real binary and assert that
// `aiwf add ac M-NNN --title "..." --body-file ./body.md` produces an
// AC whose body section under `### AC-N — <title>` contains the file's
// content, in the same atomic commit as the AC creation.
func TestAddAC_BodyFile_BinaryEndToEnd(t *testing.T) {
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
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Body epic"); err != nil {
		t.Fatalf("add epic: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "milestone", "--tdd", "none", "--epic", "E-01", "--title", "Body milestone"); err != nil {
		t.Fatalf("add milestone: %v\n%s", err, out)
	}

	bodyText := "Concrete pass criteria: the verb populates the body in the same commit.\n\nEdge case: an empty file produces an empty body section.\n"
	bodyPath := filepath.Join(root, "ac-body.md")
	if err := os.WriteFile(bodyPath, []byte(bodyText), 0o644); err != nil {
		t.Fatalf("write body file: %v", err)
	}

	// Head commit count before add-ac, so atomicity can be asserted.
	headBefore, err := runGit(root, "rev-list", "--count", "HEAD")
	if err != nil {
		t.Fatalf("rev-list before: %v\n%s", err, headBefore)
	}

	out, err := runBin(t, root, binDir, nil,
		"add", "ac", "M-001",
		"--title", "First AC",
		"--body-file", bodyPath)
	if err != nil {
		t.Fatalf("aiwf add ac --body-file: %v\n%s", err, out)
	}

	// Atomicity: exactly one commit was added.
	headAfter, err := runGit(root, "rev-list", "--count", "HEAD")
	if err != nil {
		t.Fatalf("rev-list after: %v\n%s", err, headAfter)
	}
	before, err := strconv.Atoi(strings.TrimSpace(headBefore))
	if err != nil {
		t.Fatalf("parse rev-list before: %v", err)
	}
	after, err := strconv.Atoi(strings.TrimSpace(headAfter))
	if err != nil {
		t.Fatalf("parse rev-list after: %v", err)
	}
	if after != before+1 {
		t.Errorf("commit count after add-ac = %d, want %d (one new commit)", after, before+1)
	}

	// Trailer carries the AC composite id.
	tr, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	var sawEntity bool
	for _, trailer := range tr {
		if trailer.Key == "aiwf-entity" && trailer.Value == "M-001/AC-1" {
			sawEntity = true
		}
	}
	if !sawEntity {
		t.Errorf("HEAD missing aiwf-entity: M-001/AC-1 trailer; got %v", tr)
	}

	// Milestone file contains the AC heading and body content beneath it.
	matches, err := filepath.Glob(filepath.Join(root, "work", "epics", "E-01-*", "M-001-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("glob milestone: matches=%v err=%v", matches, err)
	}
	got, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	gotStr := string(got)

	headingIdx := strings.Index(gotStr, "### AC-1 — First AC")
	if headingIdx < 0 {
		t.Fatalf("milestone missing AC-1 heading:\n%s", gotStr)
	}
	bodyIdx := strings.Index(gotStr, "Concrete pass criteria")
	if bodyIdx < 0 {
		t.Fatalf("milestone missing AC-1 body content from --body-file:\n%s", gotStr)
	}
	if bodyIdx < headingIdx {
		t.Errorf("body content appeared before AC-1 heading (offset %d vs %d):\n%s", bodyIdx, headingIdx, gotStr)
	}
}

// TestAddAC_BodyFile_MissingFile_ExitsUsage covers the defensive
// branch in runAddACCmd's body-file loop: when the path does not
// resolve, the verb exits with the usage code (2) and creates no AC.
// This pins the error-path coverage that the AC-1 happy path leaves
// unexercised.
func TestAddAC_BodyFile_MissingFile_ExitsUsage(t *testing.T) {
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
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Body epic"); err != nil {
		t.Fatalf("add epic: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "milestone", "--tdd", "none", "--epic", "E-01", "--title", "Body milestone"); err != nil {
		t.Fatalf("add milestone: %v\n%s", err, out)
	}

	out, err := runBin(t, root, binDir, nil,
		"add", "ac", "M-001",
		"--title", "First AC",
		"--body-file", filepath.Join(root, "definitely-not-a-file.md"))
	if err == nil {
		t.Fatalf("expected error on missing body file; got:\n%s", out)
	}
	// Output should name the offending path so the operator knows
	// which --body-file failed to resolve.
	if !strings.Contains(out, "definitely-not-a-file.md") {
		t.Errorf("expected error to name the missing path; got:\n%s", out)
	}

	// No AC was added — milestone should still have len(acs) == 0.
	matches, _ := filepath.Glob(filepath.Join(root, "work", "epics", "E-01-*", "M-001-*.md"))
	if len(matches) != 1 {
		t.Fatalf("milestone glob: %v", matches)
	}
	got, _ := os.ReadFile(matches[0])
	if strings.Contains(string(got), "### AC-1") {
		t.Errorf("AC was created despite missing body file:\n%s", got)
	}
}
