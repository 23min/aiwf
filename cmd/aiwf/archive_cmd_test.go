package main

// archive_cmd_test.go — M-0085 AC-1..AC-7 dispatcher-level tests for
// the `aiwf archive` verb (per ADR-0004).
//
// Per CLAUDE.md "Test the seam, not just the layer": these tests drive
// the verb through `run([]string{"archive", ...})` (the in-process
// dispatcher) so the wiring from main.go through Cobra to the verb
// body is exercised end-to-end. Pure unit tests of the verb body live
// in `internal/verb/archive_test.go`.
//
// AC-1: dry-run is the default — `aiwf archive` with no flags prints
// the planned moves and exits 0 without producing a commit or
// touching the worktree. CLAUDE.md §7 (every mutating verb produces
// exactly one git commit) — dry-run is the read-only branch that
// produces zero.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
)

// seedArchiveFixture writes a small synthetic active tree at root with
// one entity of every directly-archivable kind, plus a parent epic
// holding a milestone. Each entity carries a terminal status so
// `aiwf archive` has work to do.
//
// Per CLAUDE.md "Spec-sourced inputs for upstream-defined input
// spaces" — the storage table from ADR-0004 §"Storage — per-kind
// layout" is the input space; this fixture enumerates every populated
// row:
//
//	| Kind     | Active                              | Terminal status |
//	|----------|-------------------------------------|-----------------|
//	| Epic     | work/epics/E-NNNN-<slug>/           | done, cancelled |
//	| Milestone| work/epics/E-.../M-NNNN-<slug>.md   | (rides w/ epic) |
//	| Contract | work/contracts/C-NNNN-<slug>/       | retired,rejected|
//	| Gap      | work/gaps/G-NNNN-<slug>.md          | addressed,wontfix|
//	| Decision | work/decisions/D-NNNN-<slug>.md     | superseded,rejected|
//	| ADR      | docs/adr/ADR-NNNN-<slug>.md         | superseded,rejected|
//
// Plus one ACTIVE-status entity per kind to verify the verb leaves
// non-terminal entities in place.
func seedArchiveFixture(t *testing.T, root string) {
	t.Helper()
	mustWrite := func(rel, body string) {
		t.Helper()
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(rel), err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	// Terminal-status epic (rides whole subtree, including its milestone).
	mustWrite("work/epics/E-0010-done-epic/epic.md",
		"---\nid: E-0010\ntitle: Done epic\nstatus: done\n---\n## Goal\n\nA finished epic.\n")
	mustWrite("work/epics/E-0010-done-epic/M-0020-done-milestone.md",
		"---\nid: M-0020\ntitle: Done milestone\nstatus: done\nparent: E-0010\n---\n## Goal\n\nRides with parent epic.\n")

	// Active-status epic (stays in active dir).
	mustWrite("work/epics/E-0011-active-epic/epic.md",
		"---\nid: E-0011\ntitle: Active epic\nstatus: active\n---\n## Goal\n\nStill running.\n")
	mustWrite("work/epics/E-0011-active-epic/M-0021-running-milestone.md",
		"---\nid: M-0021\ntitle: Running milestone\nstatus: in_progress\nparent: E-0011\n---\n## Goal\n\nStill running.\n")

	// Terminal-status gap.
	mustWrite("work/gaps/G-0010-addressed-gap.md",
		"---\nid: G-0010\ntitle: Addressed gap\nstatus: addressed\n---\n## What's missing\n\nFixed.\n")

	// Active-status gap.
	mustWrite("work/gaps/G-0011-open-gap.md",
		"---\nid: G-0011\ntitle: Open gap\nstatus: open\n---\n## What's missing\n\nStill open.\n")

	// Terminal-status decision.
	mustWrite("work/decisions/D-0010-superseded-decision.md",
		"---\nid: D-0010\ntitle: Superseded decision\nstatus: superseded\n---\n## Question\n\nQ\n## Decision\n\nD\n## Reasoning\n\nR\n")

	// Active-status decision.
	mustWrite("work/decisions/D-0011-proposed-decision.md",
		"---\nid: D-0011\ntitle: Proposed decision\nstatus: proposed\n---\n## Question\n\nQ\n## Decision\n\nD\n## Reasoning\n\nR\n")

	// Terminal-status contract (whole subtree).
	mustWrite("work/contracts/C-0010-retired-contract/contract.md",
		"---\nid: C-0010\ntitle: Retired contract\nstatus: retired\n---\n## Purpose\n\nP\n## Stability\n\nS\n")

	// Active-status contract.
	mustWrite("work/contracts/C-0011-proposed-contract/contract.md",
		"---\nid: C-0011\ntitle: Proposed contract\nstatus: proposed\n---\n## Purpose\n\nP\n## Stability\n\nS\n")

	// Terminal-status ADR.
	mustWrite("docs/adr/ADR-0010-superseded-adr.md",
		"---\nid: ADR-0010\ntitle: Superseded ADR\nstatus: superseded\n---\n## Context\n\nC\n## Decision\n\nD\n## Consequences\n\nC\n")

	// Active-status ADR.
	mustWrite("docs/adr/ADR-0011-proposed-adr.md",
		"---\nid: ADR-0011\ntitle: Proposed ADR\nstatus: proposed\n---\n## Context\n\nC\n## Decision\n\nD\n## Consequences\n\nC\n")
}

// commitArchiveFixture stages and commits whatever's currently in the
// working tree, with a one-line message and the test author identity.
// Used to put the seedArchiveFixture content under git so `git mv`
// in the verb's Apply step works.
func commitArchiveFixture(t *testing.T, root, msg string) {
	t.Helper()
	if err := osExec(t, root, "git", "add", "-A"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := osExec(t, root, "git", "commit", "-q", "-m", msg); err != nil {
		t.Fatalf("git commit: %v", err)
	}
}

// archiveCommitCount returns the number of commits reachable from HEAD;
// 0 when HEAD doesn't exist yet. Mirrors commitCountSafe so tests can
// assert pre/post deltas around verb invocations.
func archiveCommitCount(t *testing.T, root string) int {
	t.Helper()
	return commitCountSafe(t, root)
}

// TestArchive_DryRunByDefault — AC-1: invoking `aiwf archive` with no
// flags prints the planned operations and exits 0 without producing
// a commit or mutating the worktree.
func TestArchive_DryRunByDefault(t *testing.T) {
	root := setupCLITestRepo(t)
	seedArchiveFixture(t, root)
	commitArchiveFixture(t, root, "seed archive fixture")

	before := archiveCommitCount(t, root)
	if rc := run([]string{"archive", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("archive (dry-run) rc = %d, want exitOK", rc)
	}
	after := archiveCommitCount(t, root)
	if delta := after - before; delta != 0 {
		t.Errorf("archive dry-run produced %d commit(s), want 0 (CLAUDE.md §7 — dry-run is read-only)", delta)
	}

	// Terminal-status files must still exist in their active locations:
	// the verb did not touch the worktree.
	for _, rel := range []string{
		"work/epics/E-0010-done-epic/epic.md",
		"work/epics/E-0010-done-epic/M-0020-done-milestone.md",
		"work/gaps/G-0010-addressed-gap.md",
		"work/decisions/D-0010-superseded-decision.md",
		"work/contracts/C-0010-retired-contract/contract.md",
		"docs/adr/ADR-0010-superseded-adr.md",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Errorf("dry-run mutated the worktree: %s missing: %v", rel, err)
		}
	}

	// Archive directories must NOT have been created.
	for _, archDir := range []string{
		"work/epics/archive",
		"work/gaps/archive",
		"work/decisions/archive",
		"work/contracts/archive",
		"docs/adr/archive",
	} {
		if _, err := os.Stat(filepath.Join(root, archDir)); err == nil {
			t.Errorf("dry-run created %s — must be a no-op on disk", archDir)
		}
	}
}

// TestArchive_HelpAvailable — AC-1: `aiwf archive --help` returns
// non-empty short text. Mirrors TestRewidth_HelpAvailable.
func TestArchive_HelpAvailable(t *testing.T) {
	root := newRootCmd()
	archive, _, err := root.Find([]string{"archive"})
	if err != nil {
		t.Fatalf("archive not registered on the root command tree: %v", err)
	}
	if strings.TrimSpace(archive.Short) == "" {
		t.Errorf("archive.Short is empty — every verb must declare a one-line summary for `aiwf --help`")
	}
}

// TestArchive_ApplyProducesSingleCommit — AC-3: `aiwf archive --apply`
// produces exactly one commit per invocation. CLAUDE.md §7.
func TestArchive_ApplyProducesSingleCommit(t *testing.T) {
	root := setupCLITestRepo(t)
	seedArchiveFixture(t, root)
	commitArchiveFixture(t, root, "seed archive fixture")

	before := archiveCommitCount(t, root)
	if rc := run([]string{"archive", "--apply", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("archive --apply rc = %d, want exitOK", rc)
	}
	after := archiveCommitCount(t, root)
	if delta := after - before; delta != 1 {
		t.Errorf("archive --apply produced %d commit(s), want exactly 1 (CLAUDE.md §7)", delta)
	}
}

// TestArchive_TrailerShape — AC-3: the `--apply` commit carries
// trailer `aiwf-verb: archive` and `aiwf-actor: <actor>` but no
// `aiwf-entity:` trailer (multi-entity sweep, same shape as
// `aiwf rewidth`). ADR-0004 §"`aiwf archive` verb": "the trailer is
// `aiwf-verb: archive` (no `aiwf-entity:` trailer — multi-entity
// sweeps are a special case in the trailer-keys policy)."
func TestArchive_TrailerShape(t *testing.T) {
	root := setupCLITestRepo(t)
	seedArchiveFixture(t, root)
	commitArchiveFixture(t, root, "seed archive fixture")

	if rc := run([]string{"archive", "--apply", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("archive --apply rc = %d, want exitOK", rc)
	}

	tr, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatalf("HeadTrailers: %v", err)
	}
	byKey := map[string][]string{}
	for _, e := range tr {
		byKey[e.Key] = append(byKey[e.Key], e.Value)
	}

	if got := byKey[gitops.TrailerVerb]; len(got) != 1 || got[0] != "archive" {
		t.Errorf("aiwf-verb = %v, want exactly [\"archive\"]\n  trailers: %+v", got, tr)
	}
	if got := byKey[gitops.TrailerActor]; len(got) != 1 || got[0] != "human/test" {
		t.Errorf("aiwf-actor = %v, want exactly [\"human/test\"]\n  trailers: %+v", got, tr)
	}
	if got := byKey[gitops.TrailerEntity]; len(got) != 0 {
		t.Errorf("aiwf-entity present (%v) — archive is a multi-entity sweep and must NOT carry per-entity trailers", got)
	}
}

// TestArchive_KindGapScopesSweep — AC-2: `aiwf archive --apply --kind gap`
// scopes the sweep to gaps only. Other terminal-status entities (epic,
// decision, contract, adr) stay in the active tree.
func TestArchive_KindGapScopesSweep(t *testing.T) {
	root := setupCLITestRepo(t)
	seedArchiveFixture(t, root)
	commitArchiveFixture(t, root, "seed archive fixture")

	if rc := run([]string{"archive", "--apply", "--kind", "gap", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("archive --apply --kind gap rc = %d, want exitOK", rc)
	}

	// Gap should have moved into archive/.
	if _, err := os.Stat(filepath.Join(root, "work", "gaps", "archive", "G-0010-addressed-gap.md")); err != nil {
		t.Errorf("--kind gap did not move terminal gap to archive/: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "work", "gaps", "G-0010-addressed-gap.md")); err == nil {
		t.Errorf("--kind gap left terminal gap in active dir")
	}

	// Other kinds' terminal entities stay in active dirs.
	for _, rel := range []string{
		"work/epics/E-0010-done-epic/epic.md",
		"work/decisions/D-0010-superseded-decision.md",
		"work/contracts/C-0010-retired-contract/contract.md",
		"docs/adr/ADR-0010-superseded-adr.md",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Errorf("--kind gap should not have touched %s: %v", rel, err)
		}
	}
}

// TestArchive_ApplyIdempotent — AC-4: running `aiwf archive --apply`
// twice produces exactly one commit (the second invocation is a
// no-op). The CLI exits 0 on the no-op path.
func TestArchive_ApplyIdempotent(t *testing.T) {
	root := setupCLITestRepo(t)
	seedArchiveFixture(t, root)
	commitArchiveFixture(t, root, "seed archive fixture")

	if rc := run([]string{"archive", "--apply", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("first archive --apply rc = %d, want exitOK", rc)
	}
	afterFirst := archiveCommitCount(t, root)

	if rc := run([]string{"archive", "--apply", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("second archive --apply rc = %d, want exitOK", rc)
	}
	afterSecond := archiveCommitCount(t, root)

	if afterSecond != afterFirst {
		t.Errorf("second --apply produced %d commit(s); want 0 (no-op on already-swept tree)", afterSecond-afterFirst)
	}
}

// TestArchive_EmptyTreeApply_NoOp — AC-4: `aiwf archive --apply` on
// an empty repo (no terminal entities) is a no-op with exit 0 and
// zero commits.
func TestArchive_EmptyTreeApply_NoOp(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	commitArchiveFixture(t, root, "init scaffolding")

	before := archiveCommitCount(t, root)
	if rc := run([]string{"archive", "--apply", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("archive --apply on empty tree rc = %d, want exitOK", rc)
	}
	after := archiveCommitCount(t, root)
	if delta := after - before; delta != 0 {
		t.Errorf("archive --apply on empty tree produced %d commit(s), want 0", delta)
	}
}

// TestArchive_PerKindStorageLayout — AC-5: the verb implements the
// per-kind storage table from ADR-0004 §"Storage — per-kind layout"
// verbatim. Each populated row of the table is enumerated explicitly:
// directory-shaped kinds (epic, contract) move whole subtrees;
// flat-file kinds (gap, decision, adr) move individual files;
// milestones do not archive independently — they ride with the
// parent epic.
//
// Per CLAUDE.md "Spec-sourced inputs for upstream-defined input
// spaces" — the storage table is the input space; this test
// enumerates every row.
func TestArchive_PerKindStorageLayout(t *testing.T) {
	root := setupCLITestRepo(t)
	seedArchiveFixture(t, root)
	commitArchiveFixture(t, root, "seed archive fixture")

	if rc := run([]string{"archive", "--apply", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("archive --apply rc = %d, want exitOK", rc)
	}

	// Per ADR-0004's storage table:
	cases := []struct {
		desc        string
		movedFrom   string
		movedTo     string
		mustExistAt string
	}{
		{
			desc:        "epic (directory-shaped) moves whole subtree to archive/",
			movedFrom:   "work/epics/E-0010-done-epic/epic.md",
			movedTo:     "work/epics/archive/E-0010-done-epic/epic.md",
			mustExistAt: "work/epics/archive/E-0010-done-epic/epic.md",
		},
		{
			desc:        "milestone rides with parent epic (does not archive independently)",
			movedFrom:   "work/epics/E-0010-done-epic/M-0020-done-milestone.md",
			movedTo:     "work/epics/archive/E-0010-done-epic/M-0020-done-milestone.md",
			mustExistAt: "work/epics/archive/E-0010-done-epic/M-0020-done-milestone.md",
		},
		{
			desc:        "contract (directory-shaped) moves whole subtree to archive/",
			movedFrom:   "work/contracts/C-0010-retired-contract/contract.md",
			movedTo:     "work/contracts/archive/C-0010-retired-contract/contract.md",
			mustExistAt: "work/contracts/archive/C-0010-retired-contract/contract.md",
		},
		{
			desc:        "gap (flat-file) moves individual file to archive/",
			movedFrom:   "work/gaps/G-0010-addressed-gap.md",
			movedTo:     "work/gaps/archive/G-0010-addressed-gap.md",
			mustExistAt: "work/gaps/archive/G-0010-addressed-gap.md",
		},
		{
			desc:        "decision (flat-file) moves individual file to archive/",
			movedFrom:   "work/decisions/D-0010-superseded-decision.md",
			movedTo:     "work/decisions/archive/D-0010-superseded-decision.md",
			mustExistAt: "work/decisions/archive/D-0010-superseded-decision.md",
		},
		{
			desc:        "ADR (flat-file) moves individual file to docs/adr/archive/",
			movedFrom:   "docs/adr/ADR-0010-superseded-adr.md",
			movedTo:     "docs/adr/archive/ADR-0010-superseded-adr.md",
			mustExistAt: "docs/adr/archive/ADR-0010-superseded-adr.md",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if _, err := os.Stat(filepath.Join(root, tc.mustExistAt)); err != nil {
				t.Errorf("expected %s to exist after archive --apply: %v", tc.mustExistAt, err)
			}
			if _, err := os.Stat(filepath.Join(root, tc.movedFrom)); err == nil {
				t.Errorf("expected %s to no longer exist (it was moved)", tc.movedFrom)
			}
		})
	}

	// Active-status entities must NOT have moved.
	for _, rel := range []string{
		"work/epics/E-0011-active-epic/epic.md",
		"work/epics/E-0011-active-epic/M-0021-running-milestone.md",
		"work/gaps/G-0011-open-gap.md",
		"work/decisions/D-0011-proposed-decision.md",
		"work/contracts/C-0011-proposed-contract/contract.md",
		"docs/adr/ADR-0011-proposed-adr.md",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Errorf("active-status entity %s should not have moved: %v", rel, err)
		}
	}
}

// TestArchive_NoPositionalIDArg — AC-6: the verb does not accept a
// positional id argument. ADR-0004 §"`aiwf archive` verb": "No id
// positional. The verb sweeps by status, not by id. There is no
// 'archive this specific entity' mode — that would be a hand-edit
// detour, not a verb."
func TestArchive_NoPositionalIDArg(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"archive", "G-0010", "--root", root, "--actor", "human/test"}); rc != exitUsage {
		t.Errorf("archive with positional id arg rc = %d, want exitUsage (the verb sweeps by status, not by id — ADR-0004)", rc)
	}
}

// TestArchive_NonHumanActorRequiresPrincipal — a non-human actor
// without --principal fails fast with exit-usage (mirrors rewidth's
// shape). Covers the principal-coherence guard.
func TestArchive_NonHumanActorRequiresPrincipal(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"archive", "--root", root, "--actor", "ai/claude"}); rc != exitUsage {
		t.Errorf("expected exitUsage for non-human actor without --principal; got %d", rc)
	}
}

// TestArchive_HumanActorForbidsPrincipal — a human actor that
// supplies --principal also fails fast.
func TestArchive_HumanActorForbidsPrincipal(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"archive", "--root", root, "--actor", "human/test", "--principal", "human/test"}); rc != exitUsage {
		t.Errorf("expected exitUsage for human actor with --principal; got %d", rc)
	}
}

// TestArchive_InvalidKindRejected — the --kind flag's closed-set
// validation fires before the verb runs. A typo or an unknown kind
// returns exit-usage with an actionable error message.
func TestArchive_InvalidKindRejected(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"archive", "--kind", "widget", "--root", root, "--actor", "human/test"}); rc != exitUsage {
		t.Errorf("expected exitUsage for invalid --kind; got %d", rc)
	}
}

// TestArchive_NonHumanActorWithPrincipal_StampsTrailer — when a
// non-human actor supplies a valid --principal, the apply path runs
// and stamps aiwf-principal on the commit. Mirrors rewidth's same-
// shape test.
func TestArchive_NonHumanActorWithPrincipal_StampsTrailer(t *testing.T) {
	root := setupCLITestRepo(t)
	seedArchiveFixture(t, root)
	commitArchiveFixture(t, root, "seed archive fixture")

	if rc := run([]string{
		"archive", "--apply", "--root", root,
		"--actor", "ai/claude", "--principal", "human/test",
	}); rc != exitOK {
		t.Fatalf("archive --apply (non-human + principal) rc = %d, want exitOK", rc)
	}

	tr, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatalf("HeadTrailers: %v", err)
	}
	hasPrincipal := false
	for _, e := range tr {
		if e.Key == gitops.TrailerPrincipal && e.Value == "human/test" {
			hasPrincipal = true
		}
	}
	if !hasPrincipal {
		t.Errorf("aiwf-principal trailer missing on non-human-actor commit (provenance model violation):\n  trailers: %+v", tr)
	}
}

// TestArchive_ExplicitDryRunFlag — `aiwf archive --dry-run` is the
// explicit alias for the default behavior. Same observable outcome as
// the no-flag invocation: zero commits, worktree untouched, exit 0.
// The flag exists so the finding hints in internal/check/hint.go and
// ad-hoc user invocations can name it directly without hitting
// "unknown flag".
func TestArchive_ExplicitDryRunFlag(t *testing.T) {
	root := setupCLITestRepo(t)
	seedArchiveFixture(t, root)
	commitArchiveFixture(t, root, "seed archive fixture")

	before := archiveCommitCount(t, root)
	if rc := run([]string{"archive", "--dry-run", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("archive --dry-run rc = %d, want exitOK", rc)
	}
	after := archiveCommitCount(t, root)
	if delta := after - before; delta != 0 {
		t.Errorf("archive --dry-run produced %d commit(s), want 0", delta)
	}

	for _, archDir := range []string{
		"work/epics/archive",
		"work/gaps/archive",
		"work/decisions/archive",
		"work/contracts/archive",
		"docs/adr/archive",
	} {
		if _, err := os.Stat(filepath.Join(root, archDir)); err == nil {
			t.Errorf("archive --dry-run created %s — must be a no-op on disk", archDir)
		}
	}
}

// TestArchive_DryRunAndApplyMutuallyExclusive — passing both flags
// fails fast with exit-usage. The combination has no coherent meaning
// (one is the read-only branch; the other commits) and silently
// preferring either would surprise the operator.
func TestArchive_DryRunAndApplyMutuallyExclusive(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"archive", "--dry-run", "--apply", "--root", root, "--actor", "human/test"}); rc != exitUsage {
		t.Errorf("archive --dry-run --apply rc = %d, want exitUsage", rc)
	}
}
