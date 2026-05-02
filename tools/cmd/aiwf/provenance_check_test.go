package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestProvenanceCheck_CleanRepoSilent: after `aiwf init` + a couple
// of normal verb invocations, `aiwf check` produces no provenance
// findings. The trailer set is well-formed at every step, so all
// eleven standing rules stay quiet.
func TestProvenanceCheck_CleanRepoSilent(t *testing.T) {
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
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Engine refactor"); err != nil {
		t.Fatalf("aiwf add epic: %v\n%s", err, out)
	}
	out, err := runBin(t, root, binDir, nil, "check")
	if err != nil {
		t.Fatalf("aiwf check: %v\n%s", err, out)
	}
	if strings.Contains(out, "provenance-") {
		t.Errorf("clean repo produced provenance findings:\n%s", out)
	}
}

// TestProvenanceCheck_HandEditedAgentCommit: simulates an external
// agent commit (no scope, ai/... actor, no on-behalf-of) hand-crafted
// onto the repo. `aiwf check` fires
// provenance-no-active-scope.
func TestProvenanceCheck_HandEditedAgentCommit(t *testing.T) {
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
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Engine"); err != nil {
		t.Fatalf("aiwf add: %v\n%s", err, out)
	}
	// Hand-craft a malformed commit: ai actor, no principal, no
	// on-behalf-of. The trailer-coherence rule (non-human actor
	// without principal) plus the no-active-scope rule both fire.
	msg := "chore: hand-crafted ai commit\n\n" +
		"aiwf-verb: promote\n" +
		"aiwf-entity: E-001\n" +
		"aiwf-actor: ai/claude\n"
	if out, err := runGit(root, "commit", "--allow-empty", "-m", msg); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	out, _ := runBin(t, root, binDir, nil, "check")
	// `aiwf check` exits 1 on findings, so we ignore the Go error and
	// inspect stdout.
	if !strings.Contains(out, "provenance-no-active-scope") {
		t.Errorf("expected provenance-no-active-scope finding; got:\n%s", out)
	}
	if !strings.Contains(out, "provenance-trailer-incoherent") {
		t.Errorf("expected provenance-trailer-incoherent (missing principal); got:\n%s", out)
	}
}

// TestProvenanceCheck_AuthorizationMissing: a hand-crafted commit
// references an authorize SHA that doesn't exist.
func TestProvenanceCheck_AuthorizationMissing(t *testing.T) {
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
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Engine"); err != nil {
		t.Fatalf("aiwf add: %v\n%s", err, out)
	}
	msg := "chore: hand-crafted scoped commit\n\n" +
		"aiwf-verb: promote\n" +
		"aiwf-entity: E-001\n" +
		"aiwf-actor: ai/claude\n" +
		"aiwf-principal: human/peter\n" +
		"aiwf-on-behalf-of: human/peter\n" +
		"aiwf-authorized-by: 0000000000000000000000000000000000000000\n"
	if out, err := runGit(root, "commit", "--allow-empty", "-m", msg); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	out, _ := runBin(t, root, binDir, nil, "check")
	if !strings.Contains(out, "provenance-authorization-missing") {
		t.Errorf("expected provenance-authorization-missing; got:\n%s", out)
	}
}
