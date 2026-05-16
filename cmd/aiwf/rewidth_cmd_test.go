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

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/gitops"
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

	// Epic dir narrow + one milestone narrow inside. Narrow widths
	// here are within the per-kind grammar floor (E-\d{2,}, M-\d{3,},
	// etc.) — the typical "valid narrow legacy" shape a real consumer
	// would carry pre-migration. Below-floor edge cases (M-77, G-9)
	// live in internal/verb/rewidth_test.go's unit tests for the
	// padToCanonical helper.
	mustWrite("work/epics/E-22-mixed-widths/epic.md",
		"---\nid: E-22\ntitle: Mixed widths\nstatus: active\n---\n## Goal\n\nDeals with M-077 and refs E-22 itself.\n")
	mustWrite("work/epics/E-22-mixed-widths/M-077-some-milestone.md",
		"---\nid: M-077\ntitle: Some milestone\nstatus: in_progress\nparent: E-22\n---\n## Goal\n\nLink: [parent epic](work/epics/E-22-mixed-widths/epic.md). Composite ref: M-077/AC-1.\n")

	// Gap.
	mustWrite("work/gaps/G-099-some-gap.md",
		"---\nid: G-099\ntitle: Some gap\nstatus: open\n---\n## What's missing\n\nReferences E-22 and M-077.\n")

	// Decision.
	mustWrite("work/decisions/D-003-some-decision.md",
		"---\nid: D-003\ntitle: Some decision\nstatus: proposed\n---\n## Question\n\nNo refs.\n## Decision\n\nNo refs.\n## Reasoning\n\nNo refs.\n")
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
	t.Parallel()
	root := setupCLITestRepo(t)
	seedNarrowFixture(t, root)
	commitFixture(t, root, "seed narrow fixture")

	before := rewidthCommitCount(t, root)
	if rc := run([]string{"rewidth", "--root", root, "--actor", "human/test"}); rc != cliutil.ExitOK {
		t.Fatalf("rewidth (dry-run) rc = %d, want cliutil.ExitOK", rc)
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
	t.Parallel()
	root := setupCLITestRepo(t)
	seedNarrowFixture(t, root)
	commitFixture(t, root, "seed narrow fixture")

	before := rewidthCommitCount(t, root)
	if rc := run([]string{"rewidth", "--apply", "--root", root, "--actor", "human/test"}); rc != cliutil.ExitOK {
		t.Fatalf("rewidth --apply rc = %d, want cliutil.ExitOK", rc)
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
	t.Parallel()
	root := setupCLITestRepo(t)
	seedNarrowFixture(t, root)
	commitFixture(t, root, "seed narrow fixture")

	if rc := run([]string{"rewidth", "--apply", "--root", root, "--actor", "human/test"}); rc != cliutil.ExitOK {
		t.Fatalf("rewidth --apply rc = %d, want cliutil.ExitOK", rc)
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
	t.Parallel()
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
	t.Parallel()
	root := setupCLITestRepo(t)
	seedNarrowFixture(t, root)
	commitFixture(t, root, "seed narrow fixture")

	if rc := run([]string{"rewidth", "--apply", "--root", root, "--actor", "human/test"}); rc != cliutil.ExitOK {
		t.Fatalf("rewidth --apply rc = %d, want cliutil.ExitOK", rc)
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
	t.Parallel()
	root := setupCLITestRepo(t)
	seedNarrowFixture(t, root)
	commitFixture(t, root, "seed narrow fixture")

	if rc := run([]string{"rewidth", "--apply", "--root", root, "--actor", "human/test"}); rc != cliutil.ExitOK {
		t.Fatalf("rewidth --apply rc = %d, want cliutil.ExitOK", rc)
	}

	// The milestone file at the post-move path should have its prose
	// rewritten — composite ref `M-077/AC-1` → `M-0077/AC-1`, link
	// `(work/epics/E-22-mixed-widths/epic.md)` → `(work/epics/E-0022-mixed-widths/epic.md)`.
	mPath := filepath.Join(root, "work", "epics", "E-0022-mixed-widths", "M-0077-some-milestone.md")
	body, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatalf("read milestone file post-apply: %v", err)
	}
	got := string(body)
	if strings.Contains(got, "M-077/AC-1") {
		t.Errorf("composite-id `M-077/AC-1` survived rewrite in %s:\n%s", mPath, got)
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
	t.Parallel()
	root := setupCLITestRepo(t)
	seedNarrowFixture(t, root)
	commitFixture(t, root, "seed narrow fixture")

	if rc := run([]string{"rewidth", "--apply", "--root", root, "--actor", "human/test"}); rc != cliutil.ExitOK {
		t.Fatalf("first rewidth --apply rc = %d, want cliutil.ExitOK", rc)
	}
	afterFirst := rewidthCommitCount(t, root)

	if rc := run([]string{"rewidth", "--apply", "--root", root, "--actor", "human/test"}); rc != cliutil.ExitOK {
		t.Fatalf("second rewidth --apply rc = %d, want cliutil.ExitOK", rc)
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
	t.Parallel()
	root := setupCLITestRepo(t)
	// init produces no commits but creates aiwf.yaml + scaffolding;
	// commit it so we have a base HEAD.
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	commitFixture(t, root, "init scaffolding")

	before := rewidthCommitCount(t, root)
	if rc := run([]string{"rewidth", "--apply", "--root", root, "--actor", "human/test"}); rc != cliutil.ExitOK {
		t.Fatalf("rewidth --apply on empty tree rc = %d, want cliutil.ExitOK", rc)
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
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := run([]string{"rewidth", "--root", root, "--actor", "ai/claude"}); rc != cliutil.ExitUsage {
		t.Errorf("expected cliutil.ExitUsage for non-human actor without --principal; got %d", rc)
	}
}

// TestRewidth_HumanActorForbidsPrincipal — a human actor that
// supplies --principal also fails fast.
func TestRewidth_HumanActorForbidsPrincipal(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := run([]string{"rewidth", "--root", root, "--actor", "human/test", "--principal", "human/test"}); rc != cliutil.ExitUsage {
		t.Errorf("expected cliutil.ExitUsage for human actor with --principal; got %d", rc)
	}
}

// TestRewidth_NonHumanActorWithPrincipal_StampsTrailer — when a
// non-human actor supplies a valid --principal, the apply path
// runs and stamps aiwf-principal on the commit. Covers the
// principal-trailer-append branch.
func TestRewidth_NonHumanActorWithPrincipal_StampsTrailer(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	seedNarrowFixture(t, root)
	commitFixture(t, root, "seed narrow fixture")

	if rc := run([]string{
		"rewidth", "--apply", "--root", root,
		"--actor", "ai/claude", "--principal", "human/test",
	}); rc != cliutil.ExitOK {
		t.Fatalf("rewidth --apply (non-human + principal) rc = %d, want cliutil.ExitOK", rc)
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
	t.Parallel()
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

	// M-0086 AC-7 recovery test: the M-0084 work surfaced a
	// frontmatter-shape finding on narrow archive ids (the
	// loader now walks active+archive). M-0086 scoped
	// frontmatter-shape (and the other shape/health rules) to
	// skip archive per ADR-0004, so this test runs without the
	// `--skip-checks` workaround. If this test starts failing
	// again, the archive scoping has regressed.
	if rc := run([]string{"rewidth", "--apply", "--root", root, "--actor", "human/test"}); rc != cliutil.ExitOK {
		t.Fatalf("rewidth --apply rc = %d, want cliutil.ExitOK", rc)
	}

	got, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("read archive post-apply: %v", err)
	}
	if string(got) != archiveBody {
		t.Errorf("archive entry diverged byte-for-byte from pre-apply:\n  before: %q\n  after:  %q", archiveBody, string(got))
	}
}

// rewidth_cmd_test.go preflight cases — fix/rewidth-preflight-checks
// patch. The verb's --apply path runs aiwf check by default and warns
// on missing expected kind directories before producing the commit.
// --skip-checks bypasses both gates for power-users.

// TestRewidth_PreflightApply_BailsOnAiwfCheckError seeds a narrow
// fixture that triggers an id-path-consistent error (frontmatter id
// disagrees with the on-disk path), then invokes `rewidth --apply`.
// The preflight catches the error, refuses the migration, exits with
// cliutil.ExitFindings, and produces no commit.
func TestRewidth_PreflightApply_BailsOnAiwfCheckError(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	seedNarrowFixture(t, root)

	// Inject a finding: the gap's frontmatter says id: G-099 but
	// we rename the file so the path encodes G-999.
	// id-path-consistent fires error-severity. Other findings may
	// also fire (refs targeting G-099 from other entities); the
	// preflight only needs at least one error to bail.
	gap := filepath.Join(root, "work", "gaps", "G-099-some-gap.md")
	gapMoved := filepath.Join(root, "work", "gaps", "G-999-some-gap.md")
	if err := os.Rename(gap, gapMoved); err != nil {
		t.Fatalf("rename gap to inject id/path mismatch: %v", err)
	}
	commitFixture(t, root, "seed narrow fixture with id/path mismatch")

	before := rewidthCommitCount(t, root)
	rc := run([]string{"rewidth", "--apply", "--root", root, "--actor", "human/test"})
	if rc != cliutil.ExitFindings {
		t.Errorf("rewidth --apply on broken tree rc = %d, want cliutil.ExitFindings (%d)", rc, cliutil.ExitFindings)
	}
	after := rewidthCommitCount(t, root)
	if delta := after - before; delta != 0 {
		t.Errorf("preflight bail produced %d commit(s), want 0", delta)
	}
}

// TestRewidth_PreflightApply_SkipChecksBypasses replays the broken-tree
// fixture from the test above, but adds --skip-checks. The verb runs
// to completion and produces exactly one commit. Power-user opt-out.
func TestRewidth_PreflightApply_SkipChecksBypasses(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	seedNarrowFixture(t, root)
	gap := filepath.Join(root, "work", "gaps", "G-099-some-gap.md")
	gapMoved := filepath.Join(root, "work", "gaps", "G-999-some-gap.md")
	if err := os.Rename(gap, gapMoved); err != nil {
		t.Fatalf("rename gap to inject id/path mismatch: %v", err)
	}
	commitFixture(t, root, "seed narrow fixture with id/path mismatch")

	before := rewidthCommitCount(t, root)
	rc := run([]string{"rewidth", "--apply", "--skip-checks", "--root", root, "--actor", "human/test"})
	if rc != cliutil.ExitOK {
		t.Fatalf("rewidth --apply --skip-checks rc = %d, want cliutil.ExitOK", rc)
	}
	after := rewidthCommitCount(t, root)
	if delta := after - before; delta != 1 {
		t.Errorf("--skip-checks should still produce exactly one commit, got delta=%d", delta)
	}
}

// TestRewidth_PreflightApply_LayoutWarningButRuns asserts that a
// missing kind directory (e.g. work/contracts) emits an advisory
// stderr warning but does not block --apply. The seedNarrowFixture
// already omits work/contracts and docs/adr; this confirms the verb
// continues past those advisory warnings.
func TestRewidth_PreflightApply_LayoutWarningButRuns(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	seedNarrowFixture(t, root)
	commitFixture(t, root, "seed narrow fixture (no contracts/, no docs/adr/)")

	before := rewidthCommitCount(t, root)
	rc := run([]string{"rewidth", "--apply", "--root", root, "--actor", "human/test"})
	if rc != cliutil.ExitOK {
		t.Fatalf("rewidth --apply with missing optional dirs rc = %d, want cliutil.ExitOK", rc)
	}
	after := rewidthCommitCount(t, root)
	if delta := after - before; delta != 1 {
		t.Errorf("apply with layout warnings should produce exactly one commit, got delta=%d", delta)
	}
}

// TestRewidth_PreflightApply_AllExpectedDirsMissingBails covers the
// "operator typed `aiwf rewidth --apply` in a non-aiwf directory"
// case. None of work/epics, work/gaps, work/decisions, work/contracts,
// docs/adr exist; the preflight bails with a usage error rather than
// running an empty migration or flooding stderr with check errors.
func TestRewidth_PreflightApply_AllExpectedDirsMissingBails(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	// No fixture seeded — none of the expected kind directories exist.
	if err := osExec(t, root, "git", "commit", "--allow-empty", "-q", "-m", "empty repo"); err != nil {
		t.Fatalf("git commit --allow-empty: %v", err)
	}

	before := rewidthCommitCount(t, root)
	rc := run([]string{"rewidth", "--apply", "--root", root, "--actor", "human/test"})
	if rc != cliutil.ExitUsage {
		t.Errorf("rewidth --apply on non-aiwf repo rc = %d, want cliutil.ExitUsage (%d)", rc, cliutil.ExitUsage)
	}
	after := rewidthCommitCount(t, root)
	if delta := after - before; delta != 0 {
		t.Errorf("non-aiwf-dir preflight produced %d commit(s), want 0", delta)
	}
}

// TestRewidth_PreflightDryRun_NoGate confirms the preflight is gated
// on --apply. Dry-run is a read-only preview; even on a tree with
// aiwf-check errors it produces the plan output and exits OK.
func TestRewidth_PreflightDryRun_NoGate(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	seedNarrowFixture(t, root)
	gap := filepath.Join(root, "work", "gaps", "G-099-some-gap.md")
	gapMoved := filepath.Join(root, "work", "gaps", "G-999-some-gap.md")
	if err := os.Rename(gap, gapMoved); err != nil {
		t.Fatalf("rename gap to inject id/path mismatch: %v", err)
	}
	commitFixture(t, root, "seed narrow fixture with id/path mismatch")

	before := rewidthCommitCount(t, root)
	rc := run([]string{"rewidth", "--root", root, "--actor", "human/test"})
	if rc != cliutil.ExitOK {
		t.Errorf("dry-run rc = %d on broken tree, want cliutil.ExitOK (preflight is --apply-only)", rc)
	}
	after := rewidthCommitCount(t, root)
	if delta := after - before; delta != 0 {
		t.Errorf("dry-run produced %d commit(s), want 0", delta)
	}
}
