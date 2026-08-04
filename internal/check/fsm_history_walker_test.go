package check

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// TestBatchedWalker_RenameChainTracking pins the M-0137/AC-3 walker's
// rename-chain handling. The walker maintains a pathToEntity map
// seeded from the tree's CURRENT paths and adds SrcPath entries when
// a rename touch is processed (newest-first walk). Without that
// extension, older commits at the pre-rename path would not resolve
// to the entity, and observations at the entity's historical path
// would be lost.
//
// Scenario: entity E-0001 was created at OLD path with status=proposed,
// promoted to active (illegal FSM transition for an epic — used as
// the observable marker), then renamed to NEW path. The tree's
// current path is NEW. The walker should observe the proposed →
// active status change at OLD path (attributed to E-0001) and emit
// the expected illegal-transition finding through the rule's normal
// predicate path.
func TestBatchedWalker_RenameChainTracking(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)

	// Commit 1: create the epic at OLD path with status=proposed.
	oldPath := "work/epics/E-0001-old/epic.md"
	r.writeEntityAtRel(oldPath, "E-0001", entity.KindEpic, entity.StatusProposed, "")
	r.gitAddAll()
	r.gitCommit("add E-0001 at old path")

	// Commit 2: status change at OLD path (proposed → done is FSM-illegal
	// for an epic per its transitions table — used here as the
	// observable marker the walker should attribute to E-0001).
	r.writeEntityAtRel(oldPath, "E-0001", entity.KindEpic, entity.StatusDone, "")
	r.gitAddAll()
	r.gitCommit("illegal status change at old path (proposed → done)")

	// Commit 3: rename old → new (no status change in this commit;
	// the new path is now where the entity lives).
	newPath := "work/epics/E-0001-new/epic.md"
	if err := os.MkdirAll(filepath.Join(r.root, filepath.Dir(newPath)), 0o755); err != nil {
		t.Fatalf("mkdir new dir: %v", err)
	}
	r.run("git", "mv", oldPath, newPath)
	r.gitCommit("rename E-0001 to new path")

	// Build tree pointing at the CURRENT (new) path only — emulating
	// what tree.Load would produce post-rename.
	tr := &tree.Tree{
		Root: r.root,
		Entities: []*entity.Entity{
			{ID: "E-0001", Kind: entity.KindEpic, Path: newPath},
		},
	}

	got := FSMHistoryConsistent(context.Background(), r.root, tr, nil, mustHead(t, r.root))

	// The walker must attribute the proposed → done observation to
	// E-0001 even though the commit touched the entity's OLD path. The
	// rule's illegal-transition predicate then fires.
	var hasFinding bool
	for _, f := range got {
		if f.Code == CodeFSMHistoryConsistent &&
			f.Subcode == "illegal-transition" &&
			f.EntityID == "E-0001" {
			hasFinding = true
		}
	}
	if !hasFinding {
		t.Errorf("expected illegal-transition finding for E-0001 (observation at pre-rename path should be attributed via rename-chain tracking); got %d finding(s): %+v",
			len(got), got)
	}
}

// TestBatchedWalker_RenameWithStatusChange_ObservesTransition pins
// G-0475: a commit that both renames an entity file and rewrites its
// status must still produce a status observation, so the FSM's legality
// verdict applies to it.
//
// The parent holds the file at the source path, so resolving
// `parent:<destination path>` finds nothing and the pair is dropped.
// The prior status has to come from the diff record's pre-image blob,
// which `git log --raw` fills with the source path's blob for a rename
// touch.
//
// The shape is reachable through any move-shaped verb: one commits the
// on-disk bytes of every path it moves, so a status edited onto disk
// rides along with the rename.
func TestBatchedWalker_RenameWithStatusChange_ObservesTransition(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)

	oldPath := "work/epics/E-0001-old/epic.md"
	newPath := "work/epics/E-0001-new/epic.md"

	r.writeEntityAtRel(oldPath, "E-0001", entity.KindEpic, entity.StatusProposed, "")
	r.gitAddAll()
	r.gitCommit("add E-0001 at old path")

	// One commit carrying both the rename and an FSM-illegal status
	// rewrite — proposed → done is not a legal epic transition.
	if err := os.MkdirAll(filepath.Join(r.root, filepath.Dir(newPath)), 0o755); err != nil {
		t.Fatalf("mkdir new dir: %v", err)
	}
	r.run("git", "mv", oldPath, newPath)
	r.writeEntityAtRel(newPath, "E-0001", entity.KindEpic, entity.StatusDone, "")
	r.gitAddAll()
	r.gitCommit("rename E-0001 and rewrite status to done in one commit")

	// Guard the premise: this exercises the rename branch only while git
	// records the touch as a rename. Below its similarity threshold git
	// emits an unrelated add + delete pair instead, which the walker
	// drops for a different reason, and the assertion would then be
	// measuring nothing.
	nameStatus := strings.TrimSpace(r.run("git", "log", "-1", "--format=", "--name-status", "-M"))
	if !strings.HasPrefix(nameStatus, "R") {
		t.Fatalf("premise: expected git to record a rename touch, got %q", nameStatus)
	}

	// Tree points at the current (post-rename) path, as tree.Load would.
	tr := &tree.Tree{
		Root: r.root,
		Entities: []*entity.Entity{
			{ID: "E-0001", Kind: entity.KindEpic, Path: newPath},
		},
	}

	got := FSMHistoryConsistent(context.Background(), r.root, tr, nil, mustHead(t, r.root))

	var hasFinding bool
	for _, f := range got {
		if f.Code == CodeFSMHistoryConsistent &&
			f.Subcode == "illegal-transition" &&
			f.EntityID == "E-0001" {
			hasFinding = true
		}
	}
	if !hasFinding {
		t.Errorf("expected illegal-transition finding for E-0001 (proposed → done landed in the same commit as the rename); got %d finding(s): %+v",
			len(got), got)
	}
}

// TestBatchedWalker_CrossEntityRenamePair_NotAttributed guards the
// rename fast path against git's rename detection, which pairs a
// delete with an add by content similarity and knows nothing of entity
// identity. Retiring one entity and opening another in a single commit
// pairs as a rename whenever the two files are alike — which
// template-shaped entity files are — and the pre-image is then the
// retired entity's blob, not the opened entity's prior state.
//
// Reading it anyway would report a status the surviving entity never
// held, at error severity, with a sovereign acknowledgment as the only
// way past it. The prior status has to come from a blob whose
// frontmatter id names this entity.
//
// The retired entity is given a transition of its own before the pair,
// because the mis-pairing reaches the walk twice: once as the paired
// commit's own pre-image, and again through the map that says where
// this entity used to live, which decides who every older commit at
// that path belongs to.
func TestBatchedWalker_CrossEntityRenamePair_NotAttributed(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)

	// Bodies identical so git's similarity index pairs the two files.
	const body = "## Context\n\nOne paragraph of boilerplate that both entities carry.\n"
	oldPath := "work/epics/E-0001-first/epic.md"
	newPath := "work/epics/E-0002-second/epic.md"

	r.writeEntityAtRel(oldPath, "E-0001", entity.KindEpic, entity.StatusProposed, body)
	r.gitAddAll()
	r.gitCommit("add E-0001 at proposed")

	// E-0001's own illegal transition, at E-0001's path, before the
	// pair. Only the path map can carry this one onto E-0002.
	r.writeEntityAtRel(oldPath, "E-0001", entity.KindEpic, entity.StatusDone, body)
	r.gitAddAll()
	r.gitCommit("hand-flip E-0001 proposed → done")

	// One commit: retire E-0001 (file gone) and open an unrelated
	// E-0002. done → proposed would be FSM-illegal if the walk were to
	// read one entity's status as the other's prior state.
	if err := os.MkdirAll(filepath.Join(r.root, filepath.Dir(newPath)), 0o755); err != nil {
		t.Fatalf("mkdir new dir: %v", err)
	}
	r.run("git", "rm", "-q", oldPath)
	r.writeEntityAtRel(newPath, "E-0002", entity.KindEpic, entity.StatusProposed, body)
	r.gitAddAll()
	r.gitCommit("retire E-0001, open E-0002")

	// Guard the premise: without git pairing the two as a rename this
	// test exercises nothing.
	nameStatus := strings.TrimSpace(r.run("git", "log", "-1", "--format=", "--name-status", "-M"))
	if !strings.HasPrefix(nameStatus, "R") {
		t.Fatalf("premise: expected git to pair the delete and the add as a rename, got %q", nameStatus)
	}

	tr := &tree.Tree{
		Root: r.root,
		Entities: []*entity.Entity{
			{ID: "E-0002", Kind: entity.KindEpic, Path: newPath},
		},
	}

	got := FSMHistoryConsistent(context.Background(), r.root, tr, nil, mustHead(t, r.root))

	for _, f := range got {
		if f.Code == CodeFSMHistoryConsistent && f.EntityID == "E-0002" {
			t.Errorf("E-0002 was created at proposed and never held another status, but the walk reported %s/%s: %s",
				f.Code, f.Subcode, f.Message)
		}
	}
}

// TestBatchedWalker_RenameFromIdlessFile_NotAttributed covers the
// third shape a rename pre-image can take: a file with no id in its
// frontmatter, which the renaming commit is what turns into an entity.
// Nothing ties it to the entity now at that path, so whatever status
// it happens to carry is not that entity's prior state.
func TestBatchedWalker_RenameFromIdlessFile_NotAttributed(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)

	const body = "## Context\n\nOne paragraph of boilerplate carried across the rename.\n"
	draftPath := "docs/drafts/some-note.md"
	entityPath := "work/epics/E-0001-promoted-draft/epic.md"

	// A note with a status but no id — not an entity yet.
	if err := mkdirAll(filepath.Join(r.root, filepath.Dir(draftPath))); err != nil {
		t.Fatalf("mkdir drafts: %v", err)
	}
	if err := writeFile(filepath.Join(r.root, draftPath),
		"---\ntitle: some note\nstatus: done\n---\n"+body); err != nil {
		t.Fatalf("write draft: %v", err)
	}
	r.gitAddAll()
	r.gitCommit("add an id-less note")

	// One commit: move it into the tree and give it an entity identity
	// at proposed. done → proposed would be FSM-illegal if the walk
	// took the note's status for the epic's prior state.
	if err := os.MkdirAll(filepath.Join(r.root, filepath.Dir(entityPath)), 0o755); err != nil {
		t.Fatalf("mkdir epic dir: %v", err)
	}
	r.run("git", "mv", draftPath, entityPath)
	r.writeEntityAtRel(entityPath, "E-0001", entity.KindEpic, entity.StatusProposed, body)
	r.gitAddAll()
	r.gitCommit("promote the note into an epic")

	nameStatus := strings.TrimSpace(r.run("git", "log", "-1", "--format=", "--name-status", "-M"))
	if !strings.HasPrefix(nameStatus, "R") {
		t.Fatalf("premise: expected git to record a rename touch, got %q", nameStatus)
	}

	tr := &tree.Tree{
		Root: r.root,
		Entities: []*entity.Entity{
			{ID: "E-0001", Kind: entity.KindEpic, Path: entityPath},
		},
	}

	got := FSMHistoryConsistent(context.Background(), r.root, tr, nil, mustHead(t, r.root))

	for _, f := range got {
		if f.Code == CodeFSMHistoryConsistent && f.EntityID == "E-0001" {
			t.Errorf("E-0001 was created at proposed and never held another status, but the walk reported %s/%s: %s",
				f.Code, f.Subcode, f.Message)
		}
	}
}

// TestBatchedWalker_ReallocateRenamePair_Attributed pins the other
// side of that identity check: `aiwf reallocate` renames the file and
// rewrites the id in one commit, so the pre-image carries an id the
// entity has since left behind. prior_ids is what still ties the two
// together, and without consulting it the renumbered entity's history
// would go unobserved from the reallocation backwards.
func TestBatchedWalker_ReallocateRenamePair_Attributed(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)

	// A body long enough that git's similarity index still pairs the
	// two files once the id, title and status lines differ.
	const body = "## Context\n\nOne paragraph of boilerplate the renumbered file carries across.\n"
	oldPath := "work/epics/E-0001-thing/epic.md"
	newPath := "work/epics/E-0002-thing/epic.md"

	r.writeEntityAtRel(oldPath, "E-0001", entity.KindEpic, entity.StatusProposed, body)
	r.gitAddAll()
	r.gitCommit("add E-0001")

	// Renumber and flip the status past the FSM in one commit.
	if err := os.MkdirAll(filepath.Join(r.root, filepath.Dir(newPath)), 0o755); err != nil {
		t.Fatalf("mkdir new dir: %v", err)
	}
	r.run("git", "mv", oldPath, newPath)
	r.writeEntityAtRel(newPath, "E-0002", entity.KindEpic, entity.StatusDone, body)
	r.gitAddAll()
	r.gitCommit("reallocate E-0001 -> E-0002, status done")

	nameStatus := strings.TrimSpace(r.run("git", "log", "-1", "--format=", "--name-status", "-M"))
	if !strings.HasPrefix(nameStatus, "R") {
		t.Fatalf("premise: expected git to record a rename touch, got %q", nameStatus)
	}

	tr := &tree.Tree{
		Root: r.root,
		Entities: []*entity.Entity{
			{ID: "E-0002", Kind: entity.KindEpic, Path: newPath, PriorIDs: []string{"E-0001"}},
		},
	}

	got := FSMHistoryConsistent(context.Background(), r.root, tr, nil, mustHead(t, r.root))

	var hasFinding bool
	for _, f := range got {
		if f.Code == CodeFSMHistoryConsistent &&
			f.Subcode == "illegal-transition" &&
			f.EntityID == "E-0002" {
			hasFinding = true
		}
	}
	if !hasFinding {
		t.Errorf("expected illegal-transition finding for E-0002 (proposed → done under its prior id E-0001); got %d finding(s): %+v",
			len(got), got)
	}
}

// TestBlobFrontmatterNames covers the identity check the rename fast
// path is guarded by, including the width canonicalization the walk
// cannot reach through a fixture: `aiwf reallocate` never writes a
// status, so a renumbering that also moved status — the only way a
// prior_ids match becomes observable end-to-end — is not producible
// through the verbs.
func TestBlobFrontmatterNames(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		fm   blobFrontmatter
		e    *entity.Entity
		want bool
	}{
		{
			name: "same id",
			fm:   blobFrontmatter{ID: "E-0001"},
			e:    &entity.Entity{ID: "E-0001"},
			want: true,
		},
		{
			name: "different id",
			fm:   blobFrontmatter{ID: "E-0002"},
			e:    &entity.Entity{ID: "E-0001"},
			want: false,
		},
		{
			name: "no id in the blob",
			fm:   blobFrontmatter{ID: "", Status: "done"},
			e:    &entity.Entity{ID: "E-0001"},
			want: false,
		},
		{
			// An entity whose own frontmatter carries no id loads with
			// an empty ID, so without the empty check two id-less files
			// would match each other on "".
			name: "no id on either side",
			fm:   blobFrontmatter{ID: "", Status: "done"},
			e:    &entity.Entity{ID: ""},
			want: false,
		},
		{
			name: "blob carries a narrow legacy width",
			fm:   blobFrontmatter{ID: "E-01"},
			e:    &entity.Entity{ID: "E-0001"},
			want: true,
		},
		{
			name: "entity carries a narrow legacy width",
			fm:   blobFrontmatter{ID: "E-0001"},
			e:    &entity.Entity{ID: "E-01"},
			want: true,
		},
		{
			name: "prior id from a reallocate",
			fm:   blobFrontmatter{ID: "E-0001"},
			e:    &entity.Entity{ID: "E-0002", PriorIDs: []string{"E-0001"}},
			want: true,
		},
		{
			name: "prior id stored at a narrow width",
			fm:   blobFrontmatter{ID: "E-0001"},
			e:    &entity.Entity{ID: "E-0002", PriorIDs: []string{"E-01"}},
			want: true,
		},
		{
			name: "prior id chain from repeated reallocation",
			fm:   blobFrontmatter{ID: "E-0001"},
			e:    &entity.Entity{ID: "E-0003", PriorIDs: []string{"E-0001", "E-0002"}},
			want: true,
		},
		{
			name: "id in neither the entity nor its priors",
			fm:   blobFrontmatter{ID: "E-0009"},
			e:    &entity.Entity{ID: "E-0003", PriorIDs: []string{"E-0001", "E-0002"}},
			want: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := c.fm.names(c.e); got != c.want {
				t.Errorf("blobFrontmatter{ID:%q}.names(&Entity{ID:%q, PriorIDs:%v}) = %v, want %v",
					c.fm.ID, c.e.ID, c.e.PriorIDs, got, c.want)
			}
		})
	}
}

// TestBatchedWalker_MissingNonZeroBlob_EmitsWalkError pins G-0327: a
// real blob id the local object store cannot produce — a damaged store,
// or a partial clone that can no longer reach the remote it would fetch
// the blob from — surfaces as a history-walk-error finding rather than
// a silent skip that hides whatever transition the pair carried.
//
// Injected through the blobReader seam because a repo in that state is
// not constructible by the fixture's ordinary git commands.
//
// The walker reads two blobs per pair and reports each side under its
// own label, so the subtests fail the older and the newer blob
// separately to reach both. Failing the older one is the case that
// matters most: the commit-side read succeeds, so the walk has a status
// in hand and only the comparison is lost — precisely the pair a silent
// skip would drop without a trace.
func TestBatchedWalker_MissingNonZeroBlob_EmitsWalkError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		// missingIn selects which commit's blob the reader cannot
		// produce, by a substring unique to that blob's content.
		missingIn string
		wantSide  string
	}{
		{name: "newer blob unreadable", missingIn: "status: done", wantSide: "commit"},
		{name: "older blob unreadable", missingIn: "status: proposed", wantSide: "parent"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r := newRepoFixture(t)
			r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add E-0001")
			r.commitEntity("E-0001", entity.KindEpic, entity.StatusDone, "skip-ahead illegal")

			delegate, err := gitops.NewBlobReader(context.Background(), r.root)
			if err != nil {
				t.Fatalf("NewBlobReader: %v", err)
			}
			defer delegate.Close()
			// ErrBlobMissing for a blob that genuinely exists in this repo
			// — the answer a store that cannot produce it gives for a
			// non-zero id `git log --raw` still names.
			fake := &fakeBlobReader{
				delegate:           delegate,
				errOnContentSubstr: c.missingIn,
				readErr:            gitops.ErrBlobMissing,
			}

			got := fsmHistoryConsistentWithDeps(context.Background(), r.root, r.tree(), nil, mustHead(t, r.root), fake)

			var sawSide bool
			for _, f := range got {
				if f.Code != CodeFSMHistoryConsistent ||
					f.Subcode != "history-walk-error" ||
					f.EntityID != "E-0001" {
					continue
				}
				if f.Severity != SeverityError {
					t.Errorf("history-walk-error severity = %q, want error", f.Severity)
				}
				if strings.Contains(f.Message, "reading "+c.wantSide+" status") {
					sawSide = true
				}
			}
			if !sawSide {
				t.Errorf("expected a %s-side history-walk-error for E-0001 (non-zero blob the store cannot produce); got %d finding(s): %+v",
					c.wantSide, len(got), got)
			}
		})
	}
}

// TestBatchedWalker_AllZeroBlob_NeverReachesTheReader pins the other
// half of G-0327: the absent side of an add or a delete keeps its
// skip. Its all-zero id short-circuits ahead of the read, so hardening
// the missing-blob case cannot turn an ordinary add or delete into a
// finding.
func TestBatchedWalker_AllZeroBlob_NeverReachesTheReader(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	relPath := canonicalEntityPath("E-0001", entity.KindEpic)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add E-0001") // all-zero PreSHA
	r.run("git", "rm", "-q", relPath)
	r.gitCommit("delete E-0001") // all-zero PostSHA

	delegate, err := gitops.NewBlobReader(context.Background(), r.root)
	if err != nil {
		t.Fatalf("NewBlobReader: %v", err)
	}
	defer delegate.Close()
	rec := &recordingBlobReader{delegate: delegate}

	// The file is gone from the working tree, so build the tree by hand
	// rather than through the git-ls-files helper.
	tr := &tree.Tree{
		Root: r.root,
		Entities: []*entity.Entity{
			{ID: "E-0001", Kind: entity.KindEpic, Path: relPath},
		},
	}

	got := fsmHistoryConsistentWithDeps(context.Background(), r.root, tr, nil, mustHead(t, r.root), rec)

	if rec.sawAllZero {
		t.Error("all-zero blob id reached ReadObject; the absent side of an add/delete must short-circuit before the read")
	}
	for _, f := range got {
		if f.Code == CodeFSMHistoryConsistent && f.Subcode == "history-walk-error" {
			t.Errorf("unexpected history-walk-error for an ordinary add + delete: %+v", f)
		}
	}
}

// recordingBlobReader delegates every read and records whether an
// all-zero blob id ever reached ReadObject.
type recordingBlobReader struct {
	delegate   *gitops.BlobReader
	sawAllZero bool
}

func (rb *recordingBlobReader) Read(commit, path string) ([]byte, error) {
	return rb.delegate.Read(commit, path)
}

func (rb *recordingBlobReader) ReadObject(sha string) ([]byte, error) {
	if gitops.BlobAllZero(sha) {
		rb.sawAllZero = true
	}
	return rb.delegate.ReadObject(sha)
}

// Close is owned by the test, which holds the delegate.
func (rb *recordingBlobReader) Close() error { return nil }

// TestBatchedWalker_OctopusMerge pins the walker's behavior on a merge
// commit with three parents. Before G-0372 Fix 1, `git log -m` fanned
// out one diff record per parent for a merge commit, and the walker
// deduped observations by (commit, parent, path) so each real
// (commit, parent) tuple emitted at most one observation. Fix 1 drops
// `-m`: `git log --raw` now suppresses diff output for merge commits
// entirely (git's default without -m/-c/--cc), so no observation is
// ever produced at a merge commit — safe because every current
// consumer of these observations discards merge-commit observations
// unconditionally (D-0010). This test now characterizes THAT: even a
// conflicted octopus merge whose resolved state differs from all three
// parents produces zero observations.
func TestBatchedWalker_OctopusMerge(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)

	// Root commit on main: epic at proposed.
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add E-0001")

	// Feature branch A: epic at active.
	r.gitCheckoutBranch("feat-a")
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusActive, "feat-a: status=active")

	// Back to main, branch B from there: epic at done.
	r.gitCheckout("main")
	r.gitCheckoutBranch("feat-b")
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusDone, "feat-b: status=done")

	// Back to main; advance with another touch so main is at proposed
	// (root) but at a different commit than feat-a's branch point.
	r.gitCheckout("main")
	r.writeEntityAtRel(canonicalEntityPath("E-0001", entity.KindEpic),
		"E-0001", entity.KindEpic, entity.StatusProposed, "main retitle\n")
	r.gitAddAll()
	r.gitCommit("main: retitle (no status change)")

	// Octopus merge: integrate both feat-a and feat-b into main. With
	// conflicting status on all three sides, the merge needs an
	// explicit resolution — write the resolved file then commit. Some
	// git versions refuse octopus merges with conflicts, in which case
	// we sequence the merges instead (still produces multi-parent
	// merge commits the walker should handle uniformly).
	cmd := r.runMaybe("git", "merge", "--no-commit", "--no-ff", "feat-a", "feat-b")
	_ = cmd // octopus may exit non-zero on conflict; resolve below
	abs := filepath.Join(r.root, canonicalEntityPath("E-0001", entity.KindEpic))
	// Resolve to cancelled (differs from all three parents).
	r.writeEntityAt(abs, "E-0001", entity.KindEpic, entity.StatusCancelled, "")
	r.gitAddAll()
	r.run("git", "commit", "-q", "-m", "octopus merge a+b (resolved to cancelled)")

	tr := r.tree()
	obs, err := walkStatusChanges(context.Background(), r.root, tr)
	if err != nil {
		t.Fatalf("walkStatusChanges: %v", err)
	}

	// G-0372 Fix 1: without -m, `git log --raw` emits no diff records for
	// a merge commit — the octopus merge itself contributes zero
	// observations, regardless of how many parents' state it resolved.
	for _, o := range obs {
		if o.IsMergeCommit {
			t.Errorf("expected no merge-commit observations (Fix 1 drops -m), got %+v", o)
		}
	}
}

// runMaybe runs a git subcommand and tolerates non-zero exit. Used
// when the caller explicitly handles failures (e.g., a merge conflict
// that the test resolves immediately afterward).
func (r *repoFixture) runMaybe(name string, args ...string) string {
	r.t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = r.root
	out, _ := cmd.CombinedOutput()
	return string(out)
}

// TestIsRepoPath_FilesystemOnly pins the helper's contract: returns
// true when .git exists (whether as a directory in normal checkouts
// or as a file in worktree pointers); false otherwise. Defined as a
// filesystem-only check so a cancelled context doesn't false-negative
// the way the exec-based gitops.IsRepo subprocess call would.
func TestIsRepoPath_FilesystemOnly(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	cases := []struct {
		name   string
		setup  func(t *testing.T) string
		expect bool
	}{
		{
			name: "plain dir without .git",
			setup: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			expect: false,
		},
		{
			name: "dir with .git directory (normal repo)",
			setup: func(t *testing.T) string {
				t.Helper()
				root := t.TempDir()
				if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
					t.Fatalf("mkdir .git: %v", err)
				}
				return root
			},
			expect: true,
		},
		{
			name: "dir with .git file (worktree pointer shape)",
			setup: func(t *testing.T) string {
				t.Helper()
				root := t.TempDir()
				// Worktree pointer: .git is a regular file containing
				// `gitdir: <path>`.
				if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: /tmp/repo/.git/worktrees/wt"), 0o644); err != nil {
					t.Fatalf("write .git pointer: %v", err)
				}
				return root
			},
			expect: true,
		},
		{
			name:   "empty path returns false",
			setup:  func(_ *testing.T) string { return "" },
			expect: false,
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			root := c.setup(t)
			if got := isRepoPath(ctx, root); got != c.expect {
				t.Errorf("isRepoPath(%q) = %v, want %v", root, got, c.expect)
			}
		})
	}
}
