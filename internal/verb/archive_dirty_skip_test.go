package verb_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// archiveMovesFor returns the OpMove source paths a plan carries, or nil
// when the result converged.
func archiveMovesFor(res *verb.Result) []string {
	if res.Plan == nil {
		return nil
	}
	var out []string
	for _, op := range res.Plan.Ops {
		if op.Type == verb.OpMove {
			out = append(out, op.Path)
		}
	}
	return out
}

// linkingGapRunner builds a repo where G-0002's body links to G-0001's
// file, and G-0001 is terminal — so a clean sweep both moves G-0001 and
// rewrites G-0002's link to the archived path.
func linkingGapRunner(t *testing.T) (r *runner, targetPath, linkerPath string) {
	t.Helper()
	r = newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Target gap", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
	target := r.tree().ByID("G-0001")
	if target == nil {
		t.Fatal("G-0001 missing from the fixture tree")
	}
	targetPath = filepath.ToSlash(target.Path)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Linking gap", testActor,
		verb.AddOptions{BodyOverride: []byte(
			"## What's missing\n\nSee [the target](" + targetPath + ") for context.\n\n" +
				"## Why it matters\n\nFixture prose.\n",
		)}))
	linker := r.tree().ByID("G-0002")
	if linker == nil {
		t.Fatal("G-0002 missing from the fixture tree")
	}
	r.must(verb.Cancel(r.ctx, r.tree(), "G-0001", testActor, "fixture", false))
	return r, targetPath, filepath.ToSlash(linker.Path)
}

// TestArchive_DirtyReferrer_SkipsTheMoveItWouldDangle is the measured
// defect this behaviour exists to prevent.
//
// planArchiveRewrites reads each referring entity's body off the working
// copy to decide whether it needs a link rewrite, and emits nothing when
// the working copy no longer carries the committed link. So a referrer
// mid-edit — the bless rhythm the shipped guidance recommends — used to
// let the target's move land while its rewrite did not, committing a link
// to a path that does not exist at HEAD.
//
// That damage is permanent: once the target is archived, IsArchivedPath
// excludes it from every later scan, so no re-run repairs the link and
// `aiwf check` reports no error. The sweep therefore declines the move
// whose rewrite it cannot compute, and says why.
func TestArchive_DirtyReferrer_SkipsTheMoveItWouldDangle(t *testing.T) {
	t.Parallel()
	r, targetPath, linkerPath := linkingGapRunner(t)

	// The operator is part-way through rewording the paragraph that holds
	// the link, so the working copy no longer carries it.
	dirtyEntity(t, r, "G-0002", "See [the target]("+targetPath+") for context.",
		"Draft rewording, link not yet re-added.")

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	for _, src := range archiveMovesFor(res) {
		if src == targetPath {
			t.Errorf("archive planned to sweep %s while its referrer %s is mid-edit; "+
				"the link rewrite cannot be computed, so the move would dangle at HEAD",
				targetPath, linkerPath)
		}
	}
	if !strings.Contains(skipReport(res), targetPath) && !strings.Contains(skipReport(res), "G-0001") {
		t.Errorf("the declined move is not reported; operator sees no reason for the omission:\n%s", skipReport(res))
	}
}

// TestArchive_DirtyTargetItself_SkipsIt covers the simpler direction: the
// entity's own file is mid-edit, so whether it is terminal at all is
// disputed. Sweeping it would move a file on the strength of a status no
// verb committed.
func TestArchive_DirtyTargetItself_SkipsIt(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)
	// HEAD says open; only the working copy says otherwise.
	path := dirtyEntity(t, r, "G-0001", "status: open", "status: wontfix")

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	for _, src := range archiveMovesFor(res) {
		if src == path {
			t.Errorf("archive planned to sweep %s on a terminal status HEAD does not carry", path)
		}
	}
}

// TestArchive_UnaffectedEntitiesStillSweep is the property that makes
// skipping the right answer rather than refusing. A sweep declined for one
// entity must not become a sweep declined for all of them — that was the
// cost of the whole-verb refusal this replaces.
func TestArchive_UnaffectedEntitiesStillSweep(t *testing.T) {
	t.Parallel()
	r, targetPath, _ := linkingGapRunner(t)

	// A third gap, terminal, with nothing referring to it.
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Independent gap", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
	r.must(verb.Cancel(r.ctx, r.tree(), "G-0003", testActor, "fixture", false))
	independent := r.tree().ByID("G-0003")
	if independent == nil {
		t.Fatal("G-0003 missing from the fixture tree")
	}
	independentPath := filepath.ToSlash(independent.Path)

	dirtyEntity(t, r, "G-0002", "See [the target]("+targetPath+") for context.",
		"Draft rewording, link not yet re-added.")

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	moves := archiveMovesFor(res)
	var sweptIndependent bool
	for _, src := range moves {
		if src == independentPath {
			sweptIndependent = true
		}
		if src == targetPath {
			t.Errorf("swept %s despite its mid-edit referrer", targetPath)
		}
	}
	if !sweptIndependent {
		t.Errorf("an unrelated draft blocked the sweep of %s; moves were %v", independentPath, moves)
	}
}

// TestArchive_UnrelatedDraft_DoesNotBlockTheSweep pins the scope from the
// other side: a mid-edit entity that refers to nothing being swept has no
// bearing on the sweep at all.
func TestArchive_UnrelatedDraft_DoesNotBlockTheSweep(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)
	r.must(verb.Cancel(r.ctx, r.tree(), "G-0001", testActor, "fixture", false))
	target := r.tree().ByID("G-0001")
	if target == nil {
		t.Fatal("G-0001 missing from the fixture tree")
	}
	targetPath := filepath.ToSlash(target.Path)

	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Unrelated gap", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
	dirtyEntity(t, r, "G-0002", "## Why it matters", "## Why it matters\n\nA draft that links to nothing.\n")

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	var swept bool
	for _, src := range archiveMovesFor(res) {
		if src == targetPath {
			swept = true
		}
	}
	if !swept {
		t.Errorf("an unrelated draft blocked the sweep of %s", targetPath)
	}
}

// TestArchive_DirtyChildStatus_DoesNotAssertAFalseSkipReason covers the
// second measured defect. An epic's skip verdict is decided by reading its
// children's statuses off the working copy, so a mid-edit milestone made
// the sweep commit a body asserting that milestone is non-terminal — under
// its own trailer — while HEAD says it is done.
//
// The verdict is unavailable, not false: the epic is declined for the
// uncommitted change rather than accused of owning live children.
func TestArchive_DirtyChildStatus_DoesNotAssertAFalseSkipReason(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	// Staged as committed state rather than driven through the FSM: a
	// milestone activation expects the epic's own branch, which is
	// scaffolding for this test rather than its subject.
	writeEntityStatus(t, r, "M-0001", "done")
	writeEntityStatus(t, r, "E-0001", "done")

	// HEAD says the milestone is done; only the working copy disagrees.
	dirtyEntity(t, r, "M-0001", "status: done", "status: in_progress")

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	report := skipReport(res)
	if strings.Contains(report, "non-terminal") {
		t.Errorf("the sweep asserts a non-terminal child that HEAD does not carry:\n%s", report)
	}
}

// skipReport returns whatever operator-facing text the result carries
// about declined moves — the NoOp message when nothing swept, the commit
// body otherwise.
func skipReport(res *verb.Result) string {
	if res.Plan != nil {
		return res.Plan.Body
	}
	return res.NoOpMessage
}

// TestArchive_SkipReportNamesTheDirtyFile pins that the operator can act
// on a declined move without guessing which draft caused it.
func TestArchive_SkipReportNamesTheDirtyFile(t *testing.T) {
	t.Parallel()
	r, targetPath, linkerPath := linkingGapRunner(t)
	dirtyEntity(t, r, "G-0002", "See [the target]("+targetPath+") for context.",
		"Draft rewording, link not yet re-added.")

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if !strings.Contains(skipReport(res), linkerPath) {
		t.Errorf("the report does not name the mid-edit file %q:\n%s", linkerPath, skipReport(res))
	}
}

// TestArchive_SkippedMoveLeavesTheRecordResolvable is the end-to-end
// property: after applying whatever the sweep did plan, the committed link
// still resolves.
func TestArchive_SkippedMoveLeavesTheRecordResolvable(t *testing.T) {
	t.Parallel()
	r, targetPath, _ := linkingGapRunner(t)
	dirtyEntity(t, r, "G-0002", "See [the target]("+targetPath+") for context.",
		"Draft rewording, link not yet re-added.")

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if res.Plan != nil {
		if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
			t.Fatalf("Apply: %v", applyErr)
		}
	}
	// The committed referrer still points at a path that exists at HEAD.
	if _, statErr := os.Stat(filepath.Join(r.root, filepath.FromSlash(targetPath))); statErr != nil {
		t.Errorf("the link target no longer exists at %s: %v", targetPath, statErr)
	}
}

// TestArchive_MaskedTerminalStatus_IsReportedNotSilent covers the case
// where the entity never becomes a candidate at all. HEAD carries a
// terminal status; only the working copy says otherwise, and the sweep
// reads the working copy — so nothing declines the move because there was
// no move to decline.
//
// Measured before this: `aiwf archive` answered "tree is converged" at
// exit 0 with a sweep genuinely due against the record. Nothing is written
// either way, so what this fixes is what the operator is told.
func TestArchive_MaskedTerminalStatus_IsReportedNotSilent(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)
	r.must(verb.Cancel(r.ctx, r.tree(), "G-0001", testActor, "fixture", false))

	// HEAD says wontfix; the working copy hides it.
	path := dirtyEntity(t, r, "G-0001", "status: wontfix", "status: open")

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if !res.NoOp {
		t.Fatalf("archive planned a sweep for an entity the working copy calls non-terminal: %+v", res.Plan)
	}
	report := skipReport(res)
	if strings.Contains(report, "converged") {
		t.Errorf("archive calls the tree converged while the record carries a pending sweep:\n%s", report)
	}
	if !strings.Contains(report, path) && !strings.Contains(report, "G-0001") {
		t.Errorf("the masked entity is not named, so the operator cannot act on it:\n%s", report)
	}
}

// TestArchive_UntrackedReferrer_DoesNotDeclineTheMove pins the
// absent-from-HEAD carve-out the decline pass shares with both guards. A
// file git has never recorded carries no committed link, so there is no
// rewrite for the sweep to lose and nothing to decline — declining here
// would block a sweep on a file the record has never seen.
func TestArchive_UntrackedReferrer_DoesNotDeclineTheMove(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)
	r.must(verb.Cancel(r.ctx, r.tree(), "G-0001", testActor, "fixture", false))
	target := r.tree().ByID("G-0001")
	if target == nil {
		t.Fatal("G-0001 missing from the fixture tree")
	}
	targetPath := filepath.ToSlash(target.Path)

	// An entity file that links to the target and has never been committed.
	untracked := filepath.Join(r.root, "work", "gaps", "G-0009-untracked.md")
	body := "---\nid: G-0009\ntitle: Untracked\nstatus: open\n---\n" +
		"## What's missing\n\nSee [the target](" + targetPath + ") for context.\n\n" +
		"## Why it matters\n\nFixture prose.\n"
	if err := os.WriteFile(untracked, []byte(body), 0o600); err != nil {
		t.Fatalf("writing the untracked gap: %v", err)
	}

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	var planned bool
	for _, src := range archiveMovesFor(res) {
		if src == targetPath {
			planned = true
		}
	}
	if !planned {
		t.Errorf("an untracked referrer declined the sweep of %s; it carries no committed link to lose:\n%s",
			targetPath, skipReport(res))
	}
}

// TestArchive_UntrackedFileInsideAMovedDir_DeclinesThatMove pins what
// dropping the absent-from-HEAD carve-out buys. A directory move carries
// whatever sits beneath it, so a stray file git has never recorded rides
// into the sweep's commit and becomes tracked from it onward.
//
// Declining that epic's move keeps the sweep's own rule — it commits what
// it selected — and leaves the rest of the sweep intact, where refusing at
// the write seam would have stopped everything.
func TestArchive_UntrackedFileInsideAMovedDir_DeclinesThatMove(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	writeEntityStatus(t, r, "E-0001", "done")
	epic := r.tree().ByID("E-0001")
	if epic == nil {
		t.Fatal("E-0001 missing from the fixture tree")
	}
	epicDir := filepath.ToSlash(filepath.Dir(epic.Path))

	stray := filepath.Join(r.root, filepath.FromSlash(epicDir), "scratch-notes.md")
	if err := os.WriteFile(stray, []byte("scratch, never committed\n"), 0o600); err != nil {
		t.Fatalf("writing the stray file: %v", err)
	}

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	for _, src := range archiveMovesFor(res) {
		if src == epicDir {
			t.Errorf("archive planned to move %s, carrying an untracked file into its commit", epicDir)
		}
	}
	if !strings.Contains(skipReport(res), "E-0001") {
		t.Errorf("the declined epic is not reported:\n%s", skipReport(res))
	}
}

// TestArchive_IgnoredFileInsideAMovedDir_DeclinesThatMove closes the
// blind spot both dirty-set queries share. `git ls-files --others`
// excludes ignored files by construction and `git diff HEAD` never knew
// them, so neither reports one — while a directory move carries it into
// the commit regardless, making it tracked from there on.
//
// git's opinion that a file is uninteresting is not the sweep's: what
// decides the move is what the move would carry.
func TestArchive_IgnoredFileInsideAMovedDir_DeclinesThatMove(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	writeEntityStatus(t, r, "E-0001", "done")
	epic := r.tree().ByID("E-0001")
	if epic == nil {
		t.Fatal("E-0001 missing from the fixture tree")
	}
	epicDir := filepath.ToSlash(filepath.Dir(epic.Path))

	if err := os.WriteFile(filepath.Join(r.root, ".gitignore"), []byte("*.scratch\n"), 0o600); err != nil {
		t.Fatalf("writing .gitignore: %v", err)
	}
	commitFixture(t, r.root, "fixture: ignore *.scratch")

	ignored := filepath.Join(r.root, filepath.FromSlash(epicDir), "notes.scratch")
	if err := os.WriteFile(ignored, []byte("ignored, never committed\n"), 0o600); err != nil {
		t.Fatalf("writing the ignored file: %v", err)
	}
	// Precondition: git reports the tree as clean.
	if out := gitStatusPorcelain(t, r.root); out != "" {
		t.Fatalf("fixture is wrong — git reports the tree dirty, so the ignored path is not the blind spot:\n%s", out)
	}

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	for _, src := range archiveMovesFor(res) {
		if src == epicDir {
			t.Errorf("archive planned to move %s, carrying an ignored file into its commit and tracking it", epicDir)
		}
	}
}

// TestArchive_HiddenEditOnACandidate_DeclinesThatMove pins that the
// sweep's per-candidate decision reads the record rather than git's
// report of the working tree (M-0284/AC-5).
//
// `assume-unchanged` is the operator's way of telling git to stop
// reporting a path. The sweep would then see a clean candidate and move
// it — and the commit-side guard would refuse the whole verb, turning a
// partial sweep into a total refusal for a reason it declines to
// attribute to any one candidate.
func TestArchive_HiddenEditOnACandidate_DeclinesThatMove(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Hidden edit gap", testActor, verb.AddOptions{
		BodyOverride: bornCompleteFixtureBody(entity.KindGap),
	}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Clean gap", testActor, verb.AddOptions{
		BodyOverride: bornCompleteFixtureBody(entity.KindGap),
	}))
	writeEntityStatus(t, r, "G-0001", "wontfix")
	writeEntityStatus(t, r, "G-0002", "wontfix")

	hidden := dirtyEntity(t, r, "G-0001", "## Why it matters", "## Why it matters\n\nHIDDEN EDIT.\n")
	hideFromGitReporting(t, r.root, "--assume-unchanged", hidden)

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	var movedHidden, movedClean bool
	for _, src := range archiveMovesFor(res) {
		switch src {
		case hidden:
			movedHidden = true
		case filepath.ToSlash(r.tree().ByID("G-0002").Path):
			movedClean = true
		}
	}
	if movedHidden {
		t.Errorf("archive planned to move %s, carrying an edit no verb computed into its commit", hidden)
	}
	if !movedClean {
		t.Error("the unaffected candidate did not sweep; one mid-edit entity must not block the rest")
	}
	if !strings.Contains(skipReport(res), "G-0001") {
		t.Errorf("the declined candidate is not reported:\n%s", skipReport(res))
	}
}

// terminalEpicWithDir builds an epic that qualifies for the sweep and
// returns its directory, so a test can put something awkward inside the
// subtree the move would carry.
func terminalEpicWithDir(t *testing.T) (r *runner, epicDir string) {
	t.Helper()
	r = newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	writeEntityStatus(t, r, "E-0001", "done")
	epic := r.tree().ByID("E-0001")
	if epic == nil {
		t.Fatal("E-0001 missing from the fixture tree")
	}
	return r, filepath.ToSlash(filepath.Dir(epic.Path))
}

// TestArchive_UnreadableFileUnderACandidate_FailsLoud pins the fail-loud
// direction for the sweep's own comparison. A committed file the sweep
// cannot read is a verdict it cannot reach, and reading that as "clean"
// would move the directory carrying it.
//
// The file has to be committed for this to bite: a path absent from the
// record needs no bytes to classify, so an unreadable untracked file is
// answered without a read.
func TestArchive_UnreadableFileUnderACandidate_FailsLoud(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	r, epicDir := terminalEpicWithDir(t)
	rel := epicDir + "/notes.md"
	abs := filepath.Join(r.root, filepath.FromSlash(rel))
	if err := os.WriteFile(abs, []byte("committed notes\n"), 0o600); err != nil {
		t.Fatalf("writing the nested file: %v", err)
	}
	commitFixture(t, r.root, "fixture: a committed file inside the epic dir")
	if err := os.Chmod(abs, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(abs, 0o644) })

	_, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err == nil {
		t.Fatal("Archive succeeded over a file it could not compare against the record")
	}
	if !strings.Contains(err.Error(), "notes.md") {
		t.Errorf("error does not name the unreadable path:\n%v", err)
	}
}

// gitStatusPorcelain returns `git status --porcelain` for root.
func gitStatusPorcelain(t *testing.T, root string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", root, "status", "--porcelain").CombinedOutput()
	if err != nil {
		t.Fatalf("git status: %v\n%s", err, out)
	}
	return strings.TrimSpace(string(out))
}

// TestArchive_MaskedTerminalReportCoversSeveralEntities exercises the
// report with more than one masked entity, which is what orders them, and
// with a mid-edit entity that is terminal on disk too — a milestone,
// which never becomes a candidate on its own and so reaches the same scan
// without being masked at all.
func TestArchive_MaskedTerminalReportCoversSeveralEntities(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "First gap", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Second gap", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.Cancel(r.ctx, r.tree(), "G-0001", testActor, "fixture", false))
	r.must(verb.Cancel(r.ctx, r.tree(), "G-0002", testActor, "fixture", false))
	// A milestone terminal in both the record and the working copy: it is
	// never a candidate on its own, so the scan reaches it and moves on.
	writeEntityStatus(t, r, "M-0001", "done")
	dirtyEntity(t, r, "M-0001", "## Goal", "## Goal\n\nA draft that changes no status.\n")

	// Both gaps terminal at HEAD, both hidden by the working copy.
	dirtyEntity(t, r, "G-0001", "status: wontfix", "status: open")
	dirtyEntity(t, r, "G-0002", "status: wontfix", "status: open")

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	report := skipReport(res)
	for _, id := range []string{"G-0001", "G-0002"} {
		if !strings.Contains(report, id) {
			t.Errorf("%s is masked but not reported:\n%s", id, report)
		}
	}
	if strings.Contains(report, "M-0001") {
		t.Errorf("the milestone is terminal in the record too; it is not masked:\n%s", report)
	}
	if i, j := strings.Index(report, "G-0001"), strings.Index(report, "G-0002"); i > j {
		t.Errorf("masked entities are not reported in a stable order:\n%s", report)
	}
}
