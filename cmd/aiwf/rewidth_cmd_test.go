package main

// rewidth_cmd_test.go — M-082 AC-1..AC-4 dispatcher-level tests for
// the `aiwf rewidth` verb.
//
// AC-1 covers the Cobra command structure: dry-run by default, --apply
// produces exactly one commit with the multi-entity-sweep trailer
// shape (aiwf-verb: rewidth, no aiwf-entity:), the new verb shows up
// in the help-quality drift surface, and completion wiring threads
// through completion_drift_test.go cleanly. AC-2/AC-3/AC-4 cover the
// active-tree rename, in-body reference rewrite, and idempotence
// behaviors against synthetic narrow-width fixtures.
//
// Per CLAUDE.md "Test the seam, not just the layer": these tests drive
// the verb through `run([]string{"rewidth", ...})` (the in-process
// dispatcher) so the wiring from main.go through Cobra to the verb
// body is exercised end-to-end. Pure unit tests of the verb body live
// in `internal/verb/rewidth_test.go`.

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/gitops"
)

// seedNarrowFixture writes a small narrow-width synthetic tree at root
// with one of every kind that has a narrow form. Each fixture file is
// committed so `git mv` can rename it. Returns the list of relative
// paths written so callers can assert about them.
//
// The fixtures use narrow IDs by design — that's the verb's input
// space. The tests are allowlisted in the narrow-id sweep policy
// for the same reason the parser-tolerance tests are.
func seedNarrowFixture(t *testing.T, root string) {
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

	// Epic dir narrow + one milestone narrow inside.
	mustWrite("work/epics/E-22-mixed-widths/epic.md",
		"---\nid: E-22\ntitle: Mixed widths\nstatus: active\n---\n## Goal\n\nDeals with M-77 and refs E-22 itself.\n")
	mustWrite("work/epics/E-22-mixed-widths/M-77-some-milestone.md",
		"---\nid: M-77\ntitle: Some milestone\nstatus: in_progress\nparent: E-22\n---\n## Goal\n\nLink: [parent epic](work/epics/E-22-mixed-widths/epic.md). Composite ref: M-77/AC-1.\n")

	// Gap.
	mustWrite("work/gaps/G-9-some-gap.md",
		"---\nid: G-9\ntitle: Some gap\nstatus: open\n---\n## What's missing\n\nReferences E-22 and M-77.\n")

	// Decision.
	mustWrite("work/decisions/D-3-some-decision.md",
		"---\nid: D-3\ntitle: Some decision\nstatus: proposed\n---\n## Question\n\nNo refs.\n## Decision\n\nNo refs.\n## Reasoning\n\nNo refs.\n")
}

// commitFixture stages and commits whatever's currently in the
// working tree, with a one-line message and the test author identity.
// Used to put the seedNarrowFixture content under git so `git mv`
// in the verb's Apply step works.
func commitFixture(t *testing.T, root, msg string) {
	t.Helper()
	if err := osExec(t, root, "git", "add", "-A"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := osExec(t, root, "git", "commit", "-q", "-m", msg); err != nil {
		t.Fatalf("git commit: %v", err)
	}
}

// commitCount returns the number of commits reachable from HEAD; 0
// when HEAD doesn't exist yet. Mirrors commitCountSafe so tests can
// assert pre/post deltas around verb invocations.
func rewidthCommitCount(t *testing.T, root string) int {
	t.Helper()
	return commitCountSafe(t, root)
}

// TestRewidth_DryRunByDefault — AC-1: invoking `aiwf rewidth` with no
// flags prints the planned operations and exits 0 without producing a
// commit.
func TestRewidth_DryRunByDefault(t *testing.T) {
	root := setupCLITestRepo(t)
	seedNarrowFixture(t, root)
	commitFixture(t, root, "seed narrow fixture")

	before := rewidthCommitCount(t, root)
	if rc := run([]string{"rewidth", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("rewidth (dry-run) rc = %d, want exitOK", rc)
	}
	after := rewidthCommitCount(t, root)
	if delta := after - before; delta != 0 {
		t.Errorf("rewidth dry-run produced %d commit(s), want 0 (CLAUDE.md §7 — dry-run is read-only)", delta)
	}

	// Narrow files must still exist on disk.
	if _, err := os.Stat(filepath.Join(root, "work", "epics", "E-22-mixed-widths", "epic.md")); err != nil {
		t.Errorf("dry-run mutated the worktree: epic.md missing: %v", err)
	}
}

// TestRewidth_ApplyProducesSingleCommit — AC-1: `aiwf rewidth --apply`
// produces exactly one commit per invocation. CLAUDE.md §7.
func TestRewidth_ApplyProducesSingleCommit(t *testing.T) {
	root := setupCLITestRepo(t)
	seedNarrowFixture(t, root)
	commitFixture(t, root, "seed narrow fixture")

	before := rewidthCommitCount(t, root)
	if rc := run([]string{"rewidth", "--apply", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("rewidth --apply rc = %d, want exitOK", rc)
	}
	after := rewidthCommitCount(t, root)
	if delta := after - before; delta != 1 {
		t.Errorf("rewidth --apply produced %d commit(s), want exactly 1 (CLAUDE.md §7)", delta)
	}
}

// TestRewidth_TrailerShape — AC-1: the `--apply` commit carries
// trailer `aiwf-verb: rewidth` and `aiwf-actor: <actor>` but no
// `aiwf-entity:` trailer (multi-entity sweep, same shape as
// `aiwf archive`).
func TestRewidth_TrailerShape(t *testing.T) {
	root := setupCLITestRepo(t)
	seedNarrowFixture(t, root)
	commitFixture(t, root, "seed narrow fixture")

	if rc := run([]string{"rewidth", "--apply", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("rewidth --apply rc = %d, want exitOK", rc)
	}

	tr, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatalf("HeadTrailers: %v", err)
	}
	byKey := map[string][]string{}
	for _, e := range tr {
		byKey[e.Key] = append(byKey[e.Key], e.Value)
	}

	if got := byKey[gitops.TrailerVerb]; len(got) != 1 || got[0] != "rewidth" {
		t.Errorf("aiwf-verb = %v, want exactly [\"rewidth\"]\n  trailers: %+v", got, tr)
	}
	if got := byKey[gitops.TrailerActor]; len(got) != 1 || got[0] != "human/test" {
		t.Errorf("aiwf-actor = %v, want exactly [\"human/test\"]\n  trailers: %+v", got, tr)
	}
	if got := byKey[gitops.TrailerEntity]; len(got) != 0 {
		t.Errorf("aiwf-entity present (%v) — rewidth is a multi-entity sweep and must NOT carry per-entity trailers (same shape as `aiwf archive`)", got)
	}
}

// TestRewidth_HelpAvailable — AC-1: `aiwf rewidth --help` returns
// non-empty short/long text. The help-quality drift test would also
// catch a totally empty Short, but this is the more pointed check —
// the verb's existence on the help surface is part of the AC.
func TestRewidth_HelpAvailable(t *testing.T) {
	root := newRootCmd()
	rewidth, _, err := root.Find([]string{"rewidth"})
	if err != nil {
		t.Fatalf("rewidth not registered on the root command tree: %v", err)
	}
	if strings.TrimSpace(rewidth.Short) == "" {
		t.Errorf("rewidth.Short is empty — every verb must declare a one-line summary for `aiwf --help`")
	}
}

// AC-2 dispatcher-level seam test — `aiwf rewidth --apply` end-to-end
// on a synthetic narrow tree leaves no narrow-width filenames in the
// active tree post-run. Mirrors the AC-5 verification grep that the
// human will run against this repo at wrap.
func TestRewidth_PostApply_NoNarrowFilenamesInActiveTree(t *testing.T) {
	root := setupCLITestRepo(t)
	seedNarrowFixture(t, root)
	commitFixture(t, root, "seed narrow fixture")

	if rc := run([]string{"rewidth", "--apply", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("rewidth --apply rc = %d, want exitOK", rc)
	}

	// Verify every active-tree file has a canonical-width id in its
	// filename. The post-rename path is the assertion: all narrow
	// filenames (E-NN, M-NN, etc.) must have widened.
	narrowPattern := regexp.MustCompile(`/[EMGDCF]-\d{1,3}(-|/|\.)`)
	roots := []string{
		filepath.Join(root, "work", "epics"),
		filepath.Join(root, "work", "gaps"),
		filepath.Join(root, "work", "decisions"),
		filepath.Join(root, "work", "contracts"),
		filepath.Join(root, "docs", "adr"),
	}
	for _, base := range roots {
		err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if os.IsNotExist(err) {
				return filepath.SkipDir
			}
			if err != nil {
				return err
			}
			if strings.Contains(path, string(filepath.Separator)+"archive"+string(filepath.Separator)) {
				return nil
			}
			rel := filepath.ToSlash(path)
			if narrowPattern.MatchString(rel) {
				t.Errorf("post-apply tree still contains a narrow-width filename: %s", rel)
			}
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			t.Errorf("walk %s: %v", base, err)
		}
	}
}

// AC-3 dispatcher-level seam test — body content rewrites land at
// the post-rename file paths and contain canonical-width ids.
func TestRewidth_PostApply_BodyContentRewritten(t *testing.T) {
	root := setupCLITestRepo(t)
	seedNarrowFixture(t, root)
	commitFixture(t, root, "seed narrow fixture")

	if rc := run([]string{"rewidth", "--apply", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("rewidth --apply rc = %d, want exitOK", rc)
	}

	// The milestone file at the post-move path should have its prose
	// rewritten — composite ref `M-77/AC-1` → `M-0077/AC-1`, link
	// `(work/epics/E-22-mixed-widths/epic.md)` → `(work/epics/E-0022-mixed-widths/epic.md)`.
	mPath := filepath.Join(root, "work", "epics", "E-0022-mixed-widths", "M-0077-some-milestone.md")
	body, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatalf("read milestone file post-apply: %v", err)
	}
	got := string(body)
	if strings.Contains(got, "M-77/AC-1") {
		t.Errorf("composite-id `M-77/AC-1` survived rewrite in %s:\n%s", mPath, got)
	}
	if !strings.Contains(got, "M-0077/AC-1") {
		t.Errorf("composite-id `M-0077/AC-1` not present in %s after rewrite:\n%s", mPath, got)
	}
	if strings.Contains(got, "work/epics/E-22-mixed-widths/epic.md") {
		t.Errorf("narrow-link `work/epics/E-22-mixed-widths/epic.md` survived rewrite in %s:\n%s", mPath, got)
	}
	if !strings.Contains(got, "work/epics/E-0022-mixed-widths/epic.md") {
		t.Errorf("canonical-link not present in %s after rewrite:\n%s", mPath, got)
	}
}

// AC-4 dispatcher-level seam test — running `aiwf rewidth --apply`
// twice produces exactly one commit (the second invocation is a no-op).
func TestRewidth_ApplyIdempotent(t *testing.T) {
	root := setupCLITestRepo(t)
	seedNarrowFixture(t, root)
	commitFixture(t, root, "seed narrow fixture")

	if rc := run([]string{"rewidth", "--apply", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("first rewidth --apply rc = %d, want exitOK", rc)
	}
	afterFirst := rewidthCommitCount(t, root)

	if rc := run([]string{"rewidth", "--apply", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("second rewidth --apply rc = %d, want exitOK", rc)
	}
	afterSecond := rewidthCommitCount(t, root)

	if afterSecond != afterFirst {
		t.Errorf("second --apply produced %d commit(s); want 0 (no-op on already-canonical tree)", afterSecond-afterFirst)
	}
}

// AC-4 dispatcher-level seam test — `aiwf rewidth --apply` on an
// empty consumer repo (no entity files anywhere) is a no-op with
// exit 0 and zero commits.
func TestRewidth_EmptyTreeApply_NoOp(t *testing.T) {
	root := setupCLITestRepo(t)
	// init produces no commits but creates aiwf.yaml + scaffolding;
	// commit it so we have a base HEAD.
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	commitFixture(t, root, "init scaffolding")

	before := rewidthCommitCount(t, root)
	if rc := run([]string{"rewidth", "--apply", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("rewidth --apply on empty tree rc = %d, want exitOK", rc)
	}
	after := rewidthCommitCount(t, root)
	if delta := after - before; delta != 0 {
		t.Errorf("rewidth --apply on empty tree produced %d commit(s), want 0", delta)
	}
}

// TestRewidth_NonHumanActorRequiresPrincipal — a non-human actor
// without --principal fails fast with exit-usage (mirrors import's
// shape). Covers the principal-coherence guard.
func TestRewidth_NonHumanActorRequiresPrincipal(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"rewidth", "--root", root, "--actor", "ai/claude"}); rc != exitUsage {
		t.Errorf("expected exitUsage for non-human actor without --principal; got %d", rc)
	}
}

// TestRewidth_HumanActorForbidsPrincipal — a human actor that
// supplies --principal also fails fast.
func TestRewidth_HumanActorForbidsPrincipal(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"rewidth", "--root", root, "--actor", "human/test", "--principal", "human/test"}); rc != exitUsage {
		t.Errorf("expected exitUsage for human actor with --principal; got %d", rc)
	}
}

// TestRewidth_NonHumanActorWithPrincipal_StampsTrailer — when a
// non-human actor supplies a valid --principal, the apply path
// runs and stamps aiwf-principal on the commit. Covers the
// principal-trailer-append branch.
func TestRewidth_NonHumanActorWithPrincipal_StampsTrailer(t *testing.T) {
	root := setupCLITestRepo(t)
	seedNarrowFixture(t, root)
	commitFixture(t, root, "seed narrow fixture")

	if rc := run([]string{
		"rewidth", "--apply", "--root", root,
		"--actor", "ai/claude", "--principal", "human/test",
	}); rc != exitOK {
		t.Fatalf("rewidth --apply (non-human + principal) rc = %d, want exitOK", rc)
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
		t.Errorf("aiwf-principal trailer not stamped on commit: %+v", tr)
	}
}

// AC-2 dispatcher-level — archive entries are byte-for-byte preserved
// across an --apply. Per ADR-0004 forget-by-default.
func TestRewidth_ArchivePreservedByteIdentical(t *testing.T) {
	root := setupCLITestRepo(t)
	seedNarrowFixture(t, root)

	// Plant an archive entry with narrow-width id. It must survive
	// --apply byte-for-byte.
	archivePath := filepath.Join(root, "work", "gaps", "archive", "G-2-archived.md")
	archiveBody := "---\nid: G-2\ntitle: Old\nstatus: wontfix\n---\n## What's missing\n\nReferences E-22 in archive prose; not rewritten per ADR-0004.\n"
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o755); err != nil {
		t.Fatalf("mkdir archive: %v", err)
	}
	if err := os.WriteFile(archivePath, []byte(archiveBody), 0o644); err != nil {
		t.Fatalf("write archive: %v", err)
	}
	commitFixture(t, root, "seed narrow fixture + archive entry")

	if rc := run([]string{"rewidth", "--apply", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("rewidth --apply rc = %d, want exitOK", rc)
	}

	got, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("read archive post-apply: %v", err)
	}
	if string(got) != archiveBody {
		t.Errorf("archive entry diverged byte-for-byte from pre-apply:\n  before: %q\n  after:  %q", archiveBody, string(got))
	}
}
