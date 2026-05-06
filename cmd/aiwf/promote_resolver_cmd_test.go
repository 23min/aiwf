package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/gitops"
)

// TestPromote_ByFlag_BinaryEndToEnd drives the dispatcher seam
// (cmd/aiwf/verbs_cmd.go runPromote) against a real binary and a
// real consumer repo. Without this test, a regression that drops
// the --by flag from the dispatcher (parses it but never threads
// it into PromoteOptions) would still pass internal/verb tests.
//
// This is the M-059 closure check: after `aiwf promote G-NNN
// addressed --by <id>`, the addressed_by frontmatter is set, the
// commit subject + trailers are correct, and the post-promote tree
// validates clean (no gap-resolved-has-resolver hint).
func TestPromote_ByFlag_BinaryEndToEnd(t *testing.T) {
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
		t.Fatalf("aiwf add epic: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "milestone", "--epic", "E-01", "--title", "Resolver"); err != nil {
		t.Fatalf("aiwf add milestone: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "gap", "--title", "Hand-edit gap"); err != nil {
		t.Fatalf("aiwf add gap: %v\n%s", err, out)
	}

	out, err := runBin(t, root, binDir, nil, "promote", "G-001", "addressed", "--by", "M-001")
	if err != nil {
		t.Fatalf("aiwf promote --by: %v\n%s", err, out)
	}

	tr, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	hasTrailer(t, tr, "aiwf-verb", "promote")
	hasTrailer(t, tr, "aiwf-entity", "G-001")
	hasTrailer(t, tr, "aiwf-to", "addressed")

	// Post-promote tree validates clean — the resolver write happened
	// in the same commit as the status flip, so the standing
	// gap-resolved-has-resolver finding never fires.
	checkOut, err := runBin(t, root, binDir, nil, "check")
	if err != nil {
		t.Fatalf("aiwf check after resolver promote: %v\n%s", err, checkOut)
	}
	if strings.Contains(checkOut, "gap-resolved-has-resolver") {
		t.Errorf("post-promote check still surfaces gap-resolved-has-resolver:\n%s", checkOut)
	}
}

// TestPromote_SupersededByFlag_BinaryEndToEnd is the ADR analogue —
// the dispatcher accepts --superseded-by, threads it through, and
// the post-promote tree validates clean (mutual link satisfied via
// supersedes on ADR-0002, written via the same flag below).
func TestPromote_SupersededByFlag_BinaryEndToEnd(t *testing.T) {
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
	if out, err := runBin(t, root, binDir, nil, "add", "adr", "--title", "Old call"); err != nil {
		t.Fatalf("aiwf add adr: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "adr", "--title", "New call"); err != nil {
		t.Fatalf("aiwf add adr: %v\n%s", err, out)
	}
	for _, id := range []string{"ADR-0001", "ADR-0002"} {
		if out, err := runBin(t, root, binDir, nil, "promote", id, "accepted"); err != nil {
			t.Fatalf("promote %s accepted: %v\n%s", id, err, out)
		}
	}

	out, err := runBin(t, root, binDir, nil,
		"promote", "ADR-0001", "superseded", "--superseded-by", "ADR-0002")
	if err != nil {
		t.Fatalf("aiwf promote --superseded-by: %v\n%s", err, out)
	}

	tr, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	hasTrailer(t, tr, "aiwf-entity", "ADR-0001")
	hasTrailer(t, tr, "aiwf-to", "superseded")
}

// TestPromote_ByFlag_RejectsAuditOnlyCombination: dispatcher-level
// mutex. Without it, a user would be able to combine the resolver
// flags (a mutation) with --audit-only (an empty-diff record),
// which contradicts audit-only's semantics. The dispatcher catches
// this before any verb work.
func TestPromote_ByFlag_RejectsAuditOnlyCombination(t *testing.T) {
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

	out, err := runBin(t, root, binDir, nil,
		"promote", "G-001", "addressed",
		"--by", "M-001",
		"--audit-only", "--reason", "should never get here")
	if err == nil {
		t.Fatalf("expected mutex refusal; got:\n%s", out)
	}
	if !strings.Contains(out, "not allowed with --audit-only") {
		t.Errorf("expected resolver/audit-only mutex message; got:\n%s", out)
	}
}
