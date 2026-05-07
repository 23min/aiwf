package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/gitops"
)

// TestAddAC_RepeatedTitle_BinaryEndToEnd is the M-057/AC-1 +
// dispatcher-seam closure: a real subprocess invocation of `aiwf
// add ac M-001 --title "..." --title "..." --title "..."` produces
// a single commit with three ACs, three aiwf-entity trailers, and
// a single OpWrite to the milestone file. Without a binary-level
// test, a regression that loses the repeatedString accumulator
// (e.g. someone reverts to fs.String) would still pass internal/verb
// tests.
func TestAddAC_RepeatedTitle_BinaryEndToEnd(t *testing.T) {
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
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Platform"); err != nil {
		t.Fatalf("add epic: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "milestone", "--tdd", "none", "--epic", "E-01", "--title", "Batch"); err != nil {
		t.Fatalf("add milestone: %v\n%s", err, out)
	}

	out, err := runBin(t, root, binDir, nil,
		"add", "ac", "M-001",
		"--title", "first criterion",
		"--title", "second criterion",
		"--title", "third criterion")
	if err != nil {
		t.Fatalf("add ac with repeated --title: %v\n%s", err, out)
	}

	tr, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	var entityTrailers []string
	for _, trailer := range tr {
		if trailer.Key == "aiwf-entity" {
			entityTrailers = append(entityTrailers, trailer.Value)
		}
	}
	want := []string{"M-001/AC-1", "M-001/AC-2", "M-001/AC-3"}
	if len(entityTrailers) != len(want) {
		t.Fatalf("aiwf-entity count = %d (%v), want %d (%v)",
			len(entityTrailers), entityTrailers, len(want), want)
	}
	for i, w := range want {
		if entityTrailers[i] != w {
			t.Errorf("aiwf-entity[%d] = %q, want %q", i, entityTrailers[i], w)
		}
	}

	// `aiwf history` for any one of the new ACs should find the
	// shared commit — the multi-trailer pattern lets a per-AC query
	// hit the batch commit naturally.
	for _, acID := range want {
		histOut, histErr := runBin(t, root, binDir, nil, "history", acID)
		if histErr != nil {
			t.Fatalf("aiwf history %s: %v\n%s", acID, histErr, histOut)
		}
		if !strings.Contains(histOut, "add") {
			t.Errorf("history for %s missing add event:\n%s", acID, histOut)
		}
	}
}

// TestAddAC_SingleTitle_BinaryUnchanged: the single-title path
// continues to work exactly as before (M-057/AC-5). A regression
// that broke the length-1 case would surface here even if every
// internal test still passed.
func TestAddAC_SingleTitle_BinaryUnchanged(t *testing.T) {
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
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Platform"); err != nil {
		t.Fatalf("add epic: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "milestone", "--tdd", "none", "--epic", "E-01", "--title", "Single"); err != nil {
		t.Fatalf("add milestone: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "ac", "M-001", "--title", "lone criterion"); err != nil {
		t.Fatalf("add ac single: %v\n%s", err, out)
	}

	subj, err := gitops.HeadSubject(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if subj != `aiwf add ac M-001/AC-1 "lone criterion"` {
		t.Errorf("single-title subject = %q, want pre-batch shape", subj)
	}
}

// TestAddAC_MissingTitle_RejectsCleanly: when no --title is given,
// the dispatcher refuses with a clear message rather than blindly
// passing an empty slice through to the verb.
func TestAddAC_MissingTitle_RejectsCleanly(t *testing.T) {
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

	out, err := runBin(t, root, binDir, nil, "add", "ac", "M-001")
	if err == nil {
		t.Fatalf("expected refusal with no --title; got:\n%s", out)
	}
	if !strings.Contains(out, "--title") {
		t.Errorf("expected --title-required message; got:\n%s", out)
	}
}
