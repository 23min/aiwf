package verb_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/verb"
)

// The commit-side write guard (ADR-0038, M-0283). A verb commits the
// bytes on disk for every path its plan touches, so an uncommitted edit
// at one of those paths lands inside the verb's commit, under the verb's
// trailer, attributed to an act that did not make it.
//
// Each test below drives one measured vector end-to-end: dirty the disk,
// run the verb, and assert nothing was committed. Refusal is asserted at
// verb.Apply rather than at a helper, because Apply is the seam every
// route reaches and the only place the pre-mutation working copy is
// still readable.

// dirtyEntity rewrites one line of an entity's on-disk file and leaves
// the edit uncommitted, returning the path it dirtied. The replacement
// must occur, or the fixture is not staging the state the test claims.
func dirtyEntity(t *testing.T, r *runner, id, oldLine, newLine string) string {
	t.Helper()
	e := r.tree().ByID(id)
	if e == nil {
		t.Fatalf("%s missing from the fixture tree", id)
	}
	abs := filepath.Join(r.root, e.Path)
	raw, err := os.ReadFile(abs) //nolint:gosec // fixture path inside the test's own temp root
	if err != nil {
		t.Fatalf("reading %s: %v", id, err)
	}
	patched := strings.Replace(string(raw), oldLine, newLine, 1)
	if patched == string(raw) {
		t.Fatalf("fixture did not dirty %s: %q not present in\n%s", id, oldLine, raw)
	}
	if err := os.WriteFile(abs, []byte(patched), 0o600); err != nil {
		t.Fatalf("writing %s: %v", id, err)
	}
	return filepath.ToSlash(e.Path)
}

// assertRefusedAndUncommitted applies a plan expecting the guard to
// refuse, and pins the two properties a refusal must have: HEAD does not
// advance, and the error names the path that blocked it so the operator
// can act on it without guessing.
func assertRefusedAndUncommitted(t *testing.T, r *runner, p *verb.Plan, wantPath string) {
	t.Helper()
	before := headSHA(t, r.root)
	_, err := verb.Apply(r.ctx, r.root, p)
	if err == nil {
		t.Fatal("Apply succeeded; the verb committed an uncommitted edit as its own")
	}
	if !strings.Contains(err.Error(), wantPath) {
		t.Errorf("error does not name the blocking path %q:\n%v", wantPath, err)
	}
	if after := headSHA(t, r.root); after != before {
		t.Errorf("HEAD advanced to %s on a refusal; want it parked at %s", after, before)
	}
}

// newGapRunner builds a repo carrying one gap, the smallest fixture that
// carries both a body and a priority field.
func newGapRunner(t *testing.T) *runner {
	t.Helper()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Some gap", testActor, verb.AddOptions{
		BodyOverride: bornCompleteFixtureBody(entity.KindGap),
	}))
	return r
}

// TestApply_DirtyBodyNotCommittedBySerializingVerb pins M-0283/AC-1.
// A serializing verb re-serializes the whole file around the frontmatter
// it computed, so an unblessed body edit travels into its commit. The
// body half matters as much as the frontmatter half: the epic's title
// says frontmatter, the defect is whole-file.
func TestApply_DirtyBodyNotCommittedBySerializingVerb(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)

	// The plan is computed against the committed tree and the body edit
	// planted afterwards, which reaches Apply's guard in isolation. The
	// same edit planted first is refused earlier, by the claim-side guard
	// — see claim_divergence_guard_test.go. Both seams are load-bearing:
	// this one covers every path a plan carries, including the nested
	// entities no claim names.
	res, err := verb.SetPriority(r.ctx, r.tree(), "G-0001", "high", false, testActor)
	if err != nil {
		t.Fatalf("SetPriority: %v", err)
	}
	path := dirtyEntity(t, r, "G-0001", "## Why it matters", "## Why it matters\n\nUNBLESSED BODY EDIT.\n")

	assertRefusedAndUncommitted(t, r, res.Plan, path)
}

// TestApply_DirtyPriorityNotCommittedByRetitle pins M-0283/AC-2 — the
// vector measured in E-0075's own body. retitle sits in both mechanisms
// at once: it builds an OpMove and an OpWrite, so covering it covers the
// overlap between the serializing and move-shaped routes.
func TestApply_DirtyPriorityNotCommittedByRetitle(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)

	// Commit a priority through its own verb, so the record says `high`
	// and only the working copy says `low`.
	r.must(verb.SetPriority(r.ctx, r.tree(), "G-0001", "high", false, testActor))

	res, err := verb.Retitle(r.ctx, r.tree(), "G-0001", "Renamed gap title", testActor, "", 0)
	if err != nil {
		t.Fatalf("Retitle: %v", err)
	}
	// Planted after the plan, so the refusal under test is Apply's. retitle
	// sits in both mechanisms at once — an OpMove and an OpWrite — so this
	// covers the overlap between the serializing and move-shaped routes.
	path := dirtyEntity(t, r, "G-0001", "priority: high", "priority: low")

	assertRefusedAndUncommitted(t, r, res.Plan, path)
}

// TestDirtyDiskNeverYieldsTreeIdenticalToParent pins M-0283/AC-3.
// A same-state comparison against the loaded tree reads the dirty disk,
// so asking for HEAD's own value looks like a real change: the verb
// writes, and commits a tree byte-identical to its parent — the class
// M-0281 existed to eliminate. Measured to also destroy the operator's
// edit, since the verb's re-serialization overwrites it.
//
// The property is what is pinned, not the seam that delivers it. The
// claim-side guard refuses this sequence before a plan exists, so the
// empty-diff commit is unreachable rather than caught at Apply — which
// is why the assertion is on the verb's own return.
func TestDirtyDiskNeverYieldsTreeIdenticalToParent(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)

	r.must(verb.SetPriority(r.ctx, r.tree(), "G-0001", "high", false, testActor))
	path := dirtyEntity(t, r, "G-0001", "priority: high", "priority: low")
	before := headSHA(t, r.root)

	// Ask for the value HEAD already carries. The loaded tree reads the
	// dirty disk, so the request reads as low -> high: a real change whose
	// write would land a tree byte-identical to its parent.
	res, err := verb.SetPriority(r.ctx, r.tree(), "G-0001", "high", false, testActor)
	if err == nil {
		t.Fatalf("SetPriority returned res=%+v, want a refusal — writing here commits an empty diff", res)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error does not name the blocking path %q:\n%v", path, err)
	}
	if after := headSHA(t, r.root); after != before {
		t.Errorf("HEAD advanced to %s on a refusal; want it parked at %s", after, before)
	}

	// The operator's edit survives the refusal — a guard that refused but
	// still overwrote the working copy would lose the same work silently.
	raw, readErr := os.ReadFile(filepath.Join(r.root, path)) //nolint:gosec // fixture path inside the test's own temp root
	if readErr != nil {
		t.Fatalf("reading %s: %v", path, readErr)
	}
	if !strings.Contains(string(raw), "priority: low") {
		t.Errorf("the refusal destroyed the operator's edit; working copy:\n%s", raw)
	}
}

// TestApply_DirtyNestedEntityNotCommittedByParentRename pins
// M-0283/AC-5 — the worst measured vector. A directory OpMove carries
// every file inside it, so a hand-edited milestone rides into the parent
// epic's rename commit: attributed to the epic, invisible to
// `aiwf history` on the milestone, and past the FSM walker, which skips
// a commit that both renames and changes status.
func TestApply_DirtyNestedEntityNotCommittedByParentRename(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Alpha epic", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First milestone", testActor, verb.AddOptions{
		EpicID: "E-0001",
		TDD:    "none",
	}))

	// tdd: decides whether acs-tdd-audit fires — a policy field, changed
	// under another entity's trailer with no event of its own.
	path := dirtyEntity(t, r, "M-0001", "tdd: none", "tdd: required")

	res, err := verb.Rename(r.ctx, r.tree(), "E-0001", "renamed-epic-slug", testActor, 0)
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	assertRefusedAndUncommitted(t, r, res.Plan, path)
}

// TestApply_UntrackedNestedFileNotCommittedByParentRename covers the
// same vector for a file git has never tracked. gatherCommitOps walks a
// moved directory and commits whatever it finds, so an untracked scratch
// file inside an epic directory lands in the rename commit — content no
// verb computed and nobody staged.
func TestApply_UntrackedNestedFileNotCommittedByParentRename(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Alpha epic", testActor, verb.AddOptions{}))

	e := r.tree().ByID("E-0001")
	scratch := filepath.Join(filepath.Dir(e.Path), "scratch-notes.md")
	if err := os.WriteFile(filepath.Join(r.root, scratch), []byte("untracked scratch\n"), 0o600); err != nil {
		t.Fatalf("writing scratch file: %v", err)
	}

	res, err := verb.Rename(r.ctx, r.tree(), "E-0001", "renamed-epic-slug", testActor, 0)
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	assertRefusedAndUncommitted(t, r, res.Plan, filepath.ToSlash(scratch))
}

// TestApply_BlessModeStillCommitsAWorkingCopyEdit pins M-0283/AC-6.
// Bless mode's precondition is that the working copy diverges from HEAD,
// so a guard refusing divergence would block the one verb whose job is
// to commit it — and would make the recovery every other refusal message
// recommends unreachable.
func TestApply_BlessModeStillCommitsAWorkingCopyEdit(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)

	dirtyEntity(t, r, "G-0001", "## Why it matters", "## Why it matters\n\nBLESSED BODY EDIT.\n")

	// body == nil selects bless mode: the working copy is the input.
	res, err := verb.EditBody(r.ctx, r.tree(), "G-0001", nil, testActor, "")
	if err != nil {
		t.Fatalf("EditBody bless mode: %v", err)
	}
	if res.Plan == nil {
		t.Fatalf("bless mode produced no plan: %+v", res)
	}
	if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply: the guard blocked the verb whose job is to commit a working-copy edit: %v", applyErr)
	}

	committed := showHEADFile(t, r.root, r.tree().ByID("G-0001").Path)
	if !strings.Contains(committed, "BLESSED BODY EDIT.") {
		t.Errorf("the blessed edit is not in HEAD:\n%s", committed)
	}
}

// TestApply_AdoptingWriteOverHandEditedFrontmatterIsRefused pins one
// half of M-0283/AC-6's verification: the working copy's own frontmatter
// must still match HEAD's, so the exemption cannot carry a field the
// operator edited by hand.
//
// The other half — that the write's content carries nothing beyond that
// working copy — is pinned in adopting_write_reconstruction_test.go.
// Both are needed: this case alone never inspects the write's bytes, so
// it says nothing about content the plan computed on its own.
//
// The plan is built directly rather than obtained from a verb, because
// every verb that sets the flag refuses this state itself.
func TestApply_AdoptingWriteOverHandEditedFrontmatterIsRefused(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)

	path := dirtyEntity(t, r, "G-0001", "status: open", "status: addressed")
	raw, err := os.ReadFile(filepath.Join(r.root, path)) //nolint:gosec // fixture path inside the test's own temp root
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	// Content equal to the bytes on disk: the most favourable case an
	// adopting write can present, and still refused, because those bytes
	// carry a hand-edited field.
	p := &verb.Plan{
		Subject:  "test adopting write over hand-edited frontmatter",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops: []verb.FileOp{
			{Type: verb.OpWrite, Path: path, Content: raw, AdoptsWorkingCopy: true},
		},
	}
	assertRefusedAndUncommitted(t, r, p, path)
}

// TestEditBody_ExplicitOverHandEditedFrontmatterIsRefused pins G-0463 at
// the verb. Explicit mode re-serializes frontmatter from a tree loaded
// off the working copy, so a hand-edited field would land under
// `aiwf-verb: edit-body` and read as part of a body edit. Both modes are
// body-only, and both now say so in the same words.
func TestEditBody_ExplicitOverHandEditedFrontmatterIsRefused(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)

	dirtyEntity(t, r, "G-0001", "status: open", "status: addressed")

	_, err := verb.EditBody(r.ctx, r.tree(), "G-0001",
		[]byte("## What's missing\n\nRewritten.\n\n## Why it matters\n\nRewritten.\n"), testActor, "")
	if err == nil {
		t.Fatal("EditBody accepted a hand-edited frontmatter; want a body-only refusal")
	}
	if !strings.Contains(err.Error(), "body-only by design") {
		t.Errorf("error should name the body-only contract, got: %v", err)
	}
}

// TestApply_ExplicitWriteThenRouteStillCommits guards the flow the
// project's own guidance encourages: write the body to disk, then route
// it through the verb. The content is already on disk and differs from
// HEAD, which is the shape the guard refuses everywhere else — so the
// exemption has to reach explicit mode, not bless mode alone.
//
// Asserted through Apply, because the plan-level assertion this pairs
// with cannot see a guard that only refuses at commit time.
func TestApply_ExplicitWriteThenRouteStillCommits(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	path, _ := epicBodyOnDisk(t, r.root)

	const wanted = "## Goal\n\nWritten to disk first, then routed through the verb.\n"
	writeBodyOnDisk(t, path, wanted)

	res, err := verb.EditBody(r.ctx, r.tree(), "E-0001", []byte(wanted), testActor, "")
	if err != nil {
		t.Fatalf("EditBody over an uncommitted matching body: %v", err)
	}
	if res.Plan == nil {
		t.Fatalf("no plan produced: %+v", res)
	}
	if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply refused the write-then-route flow: %v", applyErr)
	}
	if committed := showHEADFile(t, r.root, r.tree().ByID("E-0001").Path); !strings.Contains(committed, wanted) {
		t.Errorf("the on-disk body is not in HEAD:\n%s", committed)
	}
}

// TestApply_ExplicitRevertOverDirtyWorkingCopyStillCommits guards the
// other explicit-mode shape: an unwanted local edit, and the operator
// asking for the committed body back. The requested content equals
// HEAD's while the disk differs, so a guard keyed on the path being
// dirty would refuse a revert — leaving no aiwf-side way to express one.
func TestApply_ExplicitRevertOverDirtyWorkingCopyStillCommits(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	path, committed := epicBodyOnDisk(t, r.root)

	writeBodyOnDisk(t, path, "## Goal\n\nUnwanted local edit.\n")

	res, err := verb.EditBody(r.ctx, r.tree(), "E-0001", committed, testActor, "")
	if err != nil {
		t.Fatalf("EditBody reverting a dirty working copy: %v", err)
	}
	if res.Plan == nil {
		t.Fatalf("no plan produced: %+v", res)
	}
	if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply refused a declarative revert: %v", applyErr)
	}
	if onDisk := readFileString(t, path); !strings.Contains(onDisk, string(committed)) {
		t.Errorf("the revert did not restore the committed body:\n%s", onDisk)
	}
}

// writeBodyOnDisk replaces an entity file's body while preserving its
// frontmatter verbatim, leaving the edit uncommitted.
func writeBodyOnDisk(t *testing.T, path, body string) {
	t.Helper()
	raw, err := os.ReadFile(path) //nolint:gosec // fixture path inside the test's own temp root
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	fm, _, ok := entity.Split(raw)
	if !ok {
		t.Fatalf("file has no frontmatter:\n%s", raw)
	}
	updated := append(append([]byte("---\n"), fm...), append([]byte("---\n"), body...)...)
	if err := os.WriteFile(path, updated, 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// readFileString returns a file's contents as a string.
func readFileString(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path) //nolint:gosec // fixture path inside the test's own temp root
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(raw)
}

// showHEADFile returns a path's committed bytes at HEAD.
func showHEADFile(t *testing.T, root, relPath string) string {
	t.Helper()
	cmd := exec.Command("git", "show", "HEAD:"+filepath.ToSlash(relPath))
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git show HEAD:%s: %v", relPath, err)
	}
	return string(out)
}

// TestApply_IgnoredNestedFileNotCommittedByParentRename covers the third
// way a file reaches a move's commit without any verb naming it. A
// directory move carries whatever is beneath it, and `.gitignore` does
// not keep a file out of a tree git is told to build — while both halves
// of the dirty set omit ignored paths by construction, so a guard reading
// only those reports a clean tree and commits the file anyway.
//
// The consequence is worse than the untracked case it neighbours: the
// path becomes tracked from that commit onward, so a file the operator
// deliberately kept out of the repository is now in its history.
func TestApply_IgnoredNestedFileNotCommittedByParentRename(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Alpha epic", testActor, verb.AddOptions{}))

	if err := os.WriteFile(filepath.Join(r.root, ".gitignore"), []byte("*.log\n"), 0o600); err != nil {
		t.Fatalf("writing .gitignore: %v", err)
	}
	commitFixture(t, r.root, "fixture: ignore log files")

	e := r.tree().ByID("E-0001")
	ignored := filepath.Join(filepath.Dir(e.Path), "debug.log")
	if err := os.WriteFile(filepath.Join(r.root, ignored), []byte("ignored scratch\n"), 0o600); err != nil {
		t.Fatalf("writing the ignored file: %v", err)
	}
	// git considers the tree clean, which is exactly what makes this
	// vector invisible to a guard built on the dirty set alone.
	if out := gitPorcelain(t, r.root); out != "" {
		t.Fatalf("fixture should leave git reporting a clean tree, got:\n%s", out)
	}

	res, err := verb.Rename(r.ctx, r.tree(), "E-0001", "renamed-epic-slug", testActor, 0)
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	assertRefusedAndUncommitted(t, r, res.Plan, filepath.ToSlash(ignored))
}

// TestUncommittedConflictError_RemedyMatchesTrackedness pins that the
// refusal offers a remedy that works. `git restore` errors on a path git
// has never recorded and discards work irrecoverably on one it has, so
// naming it for the wrong class is worse than naming nothing.
func TestUncommittedConflictError_RemedyMatchesTrackedness(t *testing.T) {
	t.Parallel()

	tracked := (&verb.UncommittedConflictError{Tracked: []string{"work/gaps/G-0001-x.md"}}).Error()
	if !strings.Contains(tracked, "aiwf edit-body") {
		t.Errorf("a tracked path's remedy should offer the verb that commits a body edit:\n%s", tracked)
	}
	if !strings.Contains(tracked, "discards it outright") {
		t.Errorf("a tracked path's remedy should say what `git restore` costs:\n%s", tracked)
	}

	untracked := (&verb.UncommittedConflictError{Untracked: []string{"work/epics/E-0001-x/scratch.md"}}).Error()
	if strings.Contains(untracked, "git restore") {
		t.Errorf("an untracked path has nothing to restore; `git restore` errors on it:\n%s", untracked)
	}
	if !strings.Contains(untracked, "git stash -u") {
		t.Errorf("an untracked path's remedy should offer the flag that covers it:\n%s", untracked)
	}
}

// gitPorcelain returns `git status --porcelain` output for root.
func gitPorcelain(t *testing.T, root string) string {
	t.Helper()
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git status: %v", err)
	}
	return strings.TrimSpace(string(out))
}
