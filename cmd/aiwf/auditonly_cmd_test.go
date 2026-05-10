package main

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
)

// TestAuditOnly_CancelG24Recovery walks the load-bearing G24 scenario
// end-to-end: a gap reaches `wontfix` via a manual commit (no aiwf-
// trailers); `aiwf cancel --audit-only --reason "..."` produces an
// empty-diff commit carrying the audit trail; `aiwf history` renders
// the [audit-only] chip with the reason.
func TestAuditOnly_CancelG24Recovery(t *testing.T) {
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
	if out, err := runBin(t, root, binDir, nil, "add", "gap", "--title", "Validators leak temp files"); err != nil {
		t.Fatalf("aiwf add gap: %v\n%s", err, out)
	}
	// Simulate the manual commit that reached `wontfix` outside the
	// kernel: directly edit the gap file to flip status, stage, and
	// commit with no aiwf trailers.
	gapRel := mustFindFile(t, root, "G-0001-")
	manualFlipStatus(t, filepath.Join(root, gapRel), "open", "wontfix")
	if out, err := runGit(root, "add", gapRel); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	if out, err := runGit(root, "commit", "-m", "manually mark G-001 wontfix"); err != nil {
		t.Fatalf("manual commit: %v\n%s", err, out)
	}

	// Now the audit-only recovery commit. After this `aiwf history`
	// must show the [audit-only: ...] chip on the new event.
	out, err := runBin(t, root, binDir, nil,
		"cancel", "G-0001", "--audit-only", "--reason", "manual flip from earlier")
	if err != nil {
		t.Fatalf("aiwf cancel --audit-only: %v\n%s", err, out)
	}

	tr, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	hasTrailer(t, tr, "aiwf-verb", "cancel")
	hasTrailer(t, tr, "aiwf-entity", "G-0001")
	hasTrailer(t, tr, "aiwf-audit-only", "manual flip from earlier")

	historyOut, err := runBin(t, root, binDir, nil, "history", "G-0001")
	if err != nil {
		t.Fatalf("aiwf history: %v\n%s", err, historyOut)
	}
	if !strings.Contains(historyOut, "[audit-only: manual flip from earlier]") {
		t.Errorf("expected [audit-only:] chip in history; got:\n%s", historyOut)
	}
}

// TestAuditOnly_PromoteRefusesWhenNotAtTarget: the entity is not at
// the named state. `aiwf promote --audit-only` exits non-zero.
func TestAuditOnly_PromoteRefusesWhenNotAtTarget(t *testing.T) {
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
	// E-01 is `proposed`; audit-only against `done` must refuse.
	out, err := runBin(t, root, binDir, nil,
		"promote", "E-0001", "done", "--audit-only", "--reason", "trying to skip ahead")
	if err == nil {
		t.Fatalf("expected refusal; got:\n%s", out)
	}
	if !strings.Contains(out, "audit-only records what's already true") {
		t.Errorf("expected state-mismatch message; got:\n%s", out)
	}
}

// TestAuditOnly_RejectsForceCombination: --force and --audit-only are
// mutually exclusive (one transitions, the other backfills). The
// dispatcher catches this before invoking the verb.
func TestAuditOnly_RejectsForceCombination(t *testing.T) {
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
		"cancel", "G-0001", "--audit-only", "--force", "--reason", "both")
	if err == nil {
		t.Fatalf("expected mutex error; got:\n%s", out)
	}
	if !strings.Contains(out, "cannot coexist") {
		t.Errorf("expected mutex message; got:\n%s", out)
	}
}

// mustFindFile returns the repo-relative path of the first regular
// file under root whose base name starts with prefix. Fails the test
// if no match exists (the caller's setup must have produced one).
func mustFindFile(t *testing.T, root, prefix string) string {
	t.Helper()
	var found string
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasPrefix(d.Name(), prefix) {
			rel, _ := filepath.Rel(root, path)
			found = rel
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk: %v", walkErr)
	}
	if found == "" {
		t.Fatalf("no file with prefix %q under %s", prefix, root)
	}
	return found
}

// manualFlipStatus rewrites the entity file's `status: <oldStatus>`
// frontmatter line to `status: <newStatus>` in place. The substitution
// is anchored on the exact `status: <old>` form, so unrelated
// occurrences of the status string in the body aren't touched.
func manualFlipStatus(t *testing.T, absPath, oldStatus, newStatus string) {
	t.Helper()
	raw, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("read %s: %v", absPath, err)
	}
	target := "status: " + oldStatus
	replacement := "status: " + newStatus
	if !strings.Contains(string(raw), target) {
		t.Fatalf("file %s has no %q line", absPath, target)
	}
	updated := strings.Replace(string(raw), target, replacement, 1)
	if err := os.WriteFile(absPath, []byte(updated), 0o644); err != nil {
		t.Fatalf("write %s: %v", absPath, err)
	}
}
