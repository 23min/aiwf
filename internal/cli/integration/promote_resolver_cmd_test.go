package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/gitops"
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
// validates clean (no gap-addressed-has-resolver hint).
func TestPromote_ByFlag_BinaryEndToEnd(t *testing.T) {
	t.Parallel()
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := testutil.RunGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "add", "epic", "--title", "Platform"); err != nil {
		t.Fatalf("aiwf add epic: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "Resolver"); err != nil {
		t.Fatalf("aiwf add milestone: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "add", "gap", "--title", "Hand-edit gap"); err != nil {
		t.Fatalf("aiwf add gap: %v\n%s", err, out)
	}

	out, err := testutil.RunBin(t, root, binDir, nil, "promote", "G-0001", "addressed", "--by", "M-0001")
	if err != nil {
		t.Fatalf("aiwf promote --by: %v\n%s", err, out)
	}

	tr, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	hasTrailer(t, tr, "aiwf-verb", "promote")
	hasTrailer(t, tr, "aiwf-entity", "G-0001")
	hasTrailer(t, tr, "aiwf-to", "addressed")

	// Post-promote tree validates clean — the resolver write happened
	// in the same commit as the status flip, so the standing
	// gap-addressed-has-resolver finding never fires.
	checkOut, err := testutil.RunBin(t, root, binDir, nil, "check")
	if err != nil {
		t.Fatalf("aiwf check after resolver promote: %v\n%s", err, checkOut)
	}
	if strings.Contains(checkOut, check.CodeGapAddressedHasResolver) {
		t.Errorf("post-promote check still surfaces gap-addressed-has-resolver:\n%s", checkOut)
	}
}

// TestPromote_SupersededByFlag_BinaryEndToEnd is the ADR analogue of
// TestPromote_ByFlag_BinaryEndToEnd: the dispatcher accepts
// --superseded-by, threads it through, and the post-promote tree
// validates clean of adr-supersession-mutual because the verb records
// BOTH sides of the link in one commit — superseded_by on ADR-0001 and
// the reciprocal supersedes on ADR-0002 (G-0255).
//
// The earlier version asserted only the commit trailers and never ran
// `aiwf check`; its doc comment claimed the supersedes link was written
// "via the same flag", which was false — the reciprocal side was never
// written and the warning fired permanently. This test now drives the
// real check and pins the reciprocal write.
func TestPromote_SupersededByFlag_BinaryEndToEnd(t *testing.T) {
	t.Parallel()
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := testutil.RunGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "add", "adr", "--title", "Old call"); err != nil {
		t.Fatalf("aiwf add adr: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "add", "adr", "--title", "New call"); err != nil {
		t.Fatalf("aiwf add adr: %v\n%s", err, out)
	}
	for _, id := range []string{"ADR-0001", "ADR-0002"} {
		if out, err := testutil.RunBin(t, root, binDir, nil, "promote", id, "accepted"); err != nil {
			t.Fatalf("promote %s accepted: %v\n%s", id, err, out)
		}
	}

	out, err := testutil.RunBin(t, root, binDir, nil,
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

	// The reciprocal supersedes back-link is recorded on the superseding
	// ADR in the same commit — the explicit seam this test now pins.
	adr2, err := os.ReadFile(filepath.Join(root, "docs", "adr", "ADR-0002-new-call.md"))
	if err != nil {
		t.Fatalf("reading ADR-0002: %v", err)
	}
	if !strings.Contains(string(adr2), "supersedes:") || !strings.Contains(string(adr2), "ADR-0001") {
		t.Errorf("ADR-0002 should record supersedes: [ADR-0001]; got:\n%s", adr2)
	}

	// Headline closure (G-0255): the two-sided link means
	// adr-supersession-mutual does not fire. `aiwf check` exits 0 — only
	// advisory warnings (empty bodies, archive-sweep) remain, and that
	// code is not among them.
	checkOut, err := testutil.RunBin(t, root, binDir, nil, "check")
	if err != nil {
		t.Fatalf("aiwf check after supersession: %v\n%s", err, checkOut)
	}
	if strings.Contains(checkOut, check.CodeADRSupersessionMutual) {
		t.Errorf("post-supersession check still surfaces adr-supersession-mutual:\n%s", checkOut)
	}
}

// TestPromote_ByCommitFlag_RejectsUnresolvableSHA_BinaryEndToEnd is
// the CLI-seam companion to the verb-level G-0186 test: it proves the
// real `aiwf promote ... --by-commit <sha>` command path adopts the
// commit-resolvability validation, not a parallel dispatcher path. A
// well-formed-but-fake SHA on the normal (non-force) path produces a
// non-zero exit and the rejection message; the gap is left untouched.
//
// Without this seam test, a regression that threaded --by-commit
// through the dispatcher but skipped the verb's validation (or vice
// versa) would still pass the internal/verb tests.
func TestPromote_ByCommitFlag_RejectsUnresolvableSHA_BinaryEndToEnd(t *testing.T) {
	t.Parallel()
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := testutil.RunGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "add", "gap", "--title", "Closed by some commit"); err != nil {
		t.Fatalf("aiwf add gap: %v\n%s", err, out)
	}

	out, err := testutil.RunBin(t, root, binDir, nil,
		"promote", "G-0001", "addressed", "--by-commit", "deadbeef")
	if err == nil {
		t.Fatalf("expected non-zero exit for unresolvable --by-commit SHA; got:\n%s", out)
	}
	if !strings.Contains(out, "deadbeef") {
		t.Errorf("CLI output should name the bad SHA; got:\n%s", out)
	}
	if !strings.Contains(out, "does not resolve to a commit") {
		t.Errorf("CLI output should carry the resolvability rejection message; got:\n%s", out)
	}

	// The gap stays at open — the rejected promote made no commit, so
	// the tree validates clean without a stray addressed-with-fake-ref
	// resolver.
	checkOut, checkErr := testutil.RunBin(t, root, binDir, nil, "check")
	if checkErr != nil {
		t.Fatalf("aiwf check after rejected promote: %v\n%s", checkErr, checkOut)
	}
}

// TestPromote_ByFlag_RejectsAuditOnlyCombination: dispatcher-level
// mutex. Without it, a user would be able to combine the resolver
// flags (a mutation) with --audit-only (an empty-diff record),
// which contradicts audit-only's semantics. The dispatcher catches
// this before any verb work.
func TestPromote_ByFlag_RejectsAuditOnlyCombination(t *testing.T) {
	t.Parallel()
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := testutil.RunGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}

	out, err := testutil.RunBin(t, root, binDir, nil,
		"promote", "G-0001", "addressed",
		"--by", "M-0001",
		"--audit-only", "--reason", "should never get here")
	if err == nil {
		t.Fatalf("expected mutex refusal; got:\n%s", out)
	}
	if !strings.Contains(out, "not allowed with --audit-only") {
		t.Errorf("expected resolver/audit-only mutex message; got:\n%s", out)
	}
}
