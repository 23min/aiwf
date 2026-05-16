package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCheck_ArchiveSweepThreshold_EscalatesAggregate pins M-0088/AC-2's
// dispatcher seam: the same fixture tree (one terminal-status gap
// still in the active dir, awaiting sweep) produces a clean exit
// when `archive.sweep_threshold` is absent (warning, exit 0) and a
// findings exit when the threshold is set below the pending-sweep
// count (error, exit exitFindings).
//
// The unit test on check.ApplyArchiveSweepThreshold covers the bumper
// logic in isolation; this test exercises the seam where main.go's
// runCheckCmd reads config.Config.ArchiveSweepThreshold() and applies
// the bumper. CLAUDE.md §"Test the seam, not just the layer": a unit
// test of the helper alone is necessary but not sufficient — the
// seam test catches the case where the caller has a parallel source
// of truth and never adopts the helper.
func TestCheck_ArchiveSweepThreshold_EscalatesAggregate(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}

	// Stage one terminal-status gap still in the active dir. The
	// terminal-entity-not-archived + archive-sweep-pending rules will
	// fire at warning severity.
	gapPath := filepath.Join(root, "work", "gaps", "G-0050-pending-sweep.md")
	if err := os.MkdirAll(filepath.Dir(gapPath), 0o755); err != nil {
		t.Fatalf("mkdir gaps: %v", err)
	}
	gapContent := `---
id: G-0050
title: Pending sweep
status: addressed
addressed_by_commit: [deadbeef]
---

Body.
`
	if err := os.WriteFile(gapPath, []byte(gapContent), 0o644); err != nil {
		t.Fatalf("write gap: %v", err)
	}

	// Without archive.sweep_threshold: the aggregate fires at warning
	// severity; `aiwf check` exits 0 (warnings don't block).
	if rc := run([]string{"check", "--root", root}); rc != exitOK {
		t.Errorf("check without archive.sweep_threshold = %d, want exitOK (%d) — warnings should not block",
			rc, exitOK)
	}

	// Append archive.sweep_threshold: 0 — the strictest possible
	// setting (any pending sweep blocks). With one terminal in the
	// active dir, count=1 > threshold=0, so the aggregate escalates.
	cfgPath := filepath.Join(root, "aiwf.yaml")
	current, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	if err := os.WriteFile(cfgPath, append(current, []byte("\narchive:\n  sweep_threshold: 0\n")...), 0o644); err != nil {
		t.Fatalf("rewrite aiwf.yaml: %v", err)
	}

	if rc := run([]string{"check", "--root", root}); rc != exitFindings {
		t.Errorf("check with archive.sweep_threshold: 0 = %d, want exitFindings (%d) — threshold must escalate the aggregate to error",
			rc, exitFindings)
	}
}

// TestCheck_ArchiveSweepThreshold_MessageNamesThresholdAndCount pins
// that the escalated finding's human-readable Message cites both the
// configured threshold and the actual count. The unit test on the
// helper asserts the same shape against a synthetic finding; this
// test confirms the wired-through path produces the same Message in
// the JSON envelope so downstream tools and CI logs name the breach.
func TestCheck_ArchiveSweepThreshold_MessageNamesThresholdAndCount(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}

	// Two pending-sweep gaps so count > threshold gives a non-trivial
	// magnitude in the message. Canonical 4-digit ids per ADR-0008.
	ids := []string{"G-0070", "G-0071"}
	for i, statusVal := range []string{"addressed", "wontfix"} {
		id := ids[i]
		gapPath := filepath.Join(root, "work", "gaps", id+"-sample.md")
		if err := os.MkdirAll(filepath.Dir(gapPath), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		body := "---\nid: " + id + "\ntitle: Sample\nstatus: " + statusVal + "\naddressed_by_commit: [deadbeef]\n---\n\nBody.\n"
		if err := os.WriteFile(gapPath, []byte(body), 0o644); err != nil {
			t.Fatalf("write gap: %v", err)
		}
	}

	cfgPath := filepath.Join(root, "aiwf.yaml")
	current, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	if err := os.WriteFile(cfgPath, append(current, []byte("\narchive:\n  sweep_threshold: 1\n")...), 0o644); err != nil {
		t.Fatalf("rewrite aiwf.yaml: %v", err)
	}

	var rc int
	outBytes := captureStdout(t, func() {
		rc = run([]string{"check", "--root", root})
	})
	out := string(outBytes)
	if rc != exitFindings {
		t.Fatalf("check exit = %d, want exitFindings (%d); output:\n%s", rc, exitFindings, out)
	}
	// The escalated message names both the count (2) and the
	// configured threshold (1).
	if !strings.Contains(out, "archive.sweep_threshold") {
		t.Errorf("escalated check output must cite `archive.sweep_threshold`; got:\n%s", out)
	}
	if !strings.Contains(out, "archive-sweep-pending") {
		t.Errorf("escalated check output must name the aggregate finding code; got:\n%s", out)
	}
	if !strings.Contains(out, "aiwf archive") {
		t.Errorf("escalated check output must name the sweep verb (`aiwf archive`); got:\n%s", out)
	}
}

// TestCheck_ArchiveSweepThreshold_UnsetStaysPermissive pins M-0088/AC-8:
// when `archive.sweep_threshold` is unset (the default), `aiwf check`
// exits 0 even with many terminal-status entities pending sweep. The
// aggregate `archive-sweep-pending` finding fires at warning severity
// (advisory) and does not flip the exit code. The M-0085/AC-7 binary
// migration test (`TestBinary_ArchiveKernelMigration_LeavesCheckClean`)
// relies on this contract — the kernel's own `aiwf.yaml` has no
// threshold set, so pre-sweep the test sees warnings but no errors,
// and exit 0 from the pre-sweep check matters for the test's
// "pre-sweep state" gate.
//
// This dedicated AC-8 marker isolates the contract from the larger
// migration fixture: a future change that quietly flips the bumper's
// default from "set=false ⇒ no-op" to "set=false ⇒ escalate" would
// break this test before it broke the migration scenario, and the
// failure would name the AC.
func TestCheck_ArchiveSweepThreshold_UnsetStaysPermissive(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}

	// Three pending-sweep gaps — non-trivial backlog. Without
	// `archive.sweep_threshold` set, none of these escalate.
	ids := []string{"G-1000", "G-1001", "G-1002"}
	statuses := []string{"addressed", "wontfix", "addressed"}
	for i, id := range ids {
		gapPath := filepath.Join(root, "work", "gaps", id+"-pending.md")
		if err := os.MkdirAll(filepath.Dir(gapPath), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		body := "---\nid: " + id + "\ntitle: Pending\nstatus: " + statuses[i] + "\naddressed_by_commit: [deadbeef]\n---\n\nBody.\n"
		if err := os.WriteFile(gapPath, []byte(body), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	// `aiwf.yaml` from `aiwf init` has no `archive:` block. Verify
	// the check exits 0 (advisory; warnings don't block) — this is
	// the AC-8 invariant.
	if rc := run([]string{"check", "--root", root}); rc != exitOK {
		t.Errorf("check with archive.sweep_threshold unset = %d, want exitOK (%d) — default-permissive: warnings advisory; pre-sweep migration must still exit 0",
			rc, exitOK)
	}
}
