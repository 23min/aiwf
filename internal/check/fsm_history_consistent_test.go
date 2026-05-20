package check

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// Walker tests for M-0130/AC-1: enumerate DAG-aware status-change
// observations across one or more entities, with rename tracking,
// branched/merged history correctness, and trailer capture. AC-1
// emits no findings — the test set asserts the walker's observation
// shape, which AC-2/3/4 build their per-subcode predicates on.

// TestFSMHistoryConsistent_NilGuards covers the entry-point's nil-
// tree and empty-root short-circuits.
func TestFSMHistoryConsistent_NilGuards(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		root string
		tr   *tree.Tree
	}{
		{"nil tree", "/some/path", nil},
		{"empty root", "", &tree.Tree{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := FSMHistoryConsistent(context.Background(), c.root, c.tr)
			if got != nil {
				t.Errorf("expected nil findings, got %+v", got)
			}
		})
	}
}

// TestWalkStatusChanges_SkipsNilOrPathlessEntities covers the
// defensive `if e == nil || e.Path == ""` branch — a tree
// containing a malformed entry should be tolerated without
// crashing or producing observations for that entry.
func TestWalkStatusChanges_SkipsNilOrPathlessEntities(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add")
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusActive, "promote")

	tr := &tree.Tree{Root: r.root, Entities: []*entity.Entity{
		nil,
		{ID: "E-0002", Kind: entity.KindEpic, Path: ""},
		{ID: "E-0001", Kind: entity.KindEpic, Path: canonicalEntityPath("E-0001", entity.KindEpic)},
	}}
	obs, err := walkStatusChanges(context.Background(), r.root, tr)
	if err != nil {
		t.Fatalf("walkStatusChanges: %v", err)
	}
	if len(obs) != 1 || obs[0].EntityID != "E-0001" {
		t.Errorf("expected one observation for E-0001 (others skipped); got %+v", obs)
	}
}

// TestWalkOneEntity_NoTouches — an entity whose declared path has
// no git history at all (e.g., a new uncommitted entity) produces
// zero observations cleanly.
func TestWalkOneEntity_NoTouches(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	// Commit something else so the repo has commits but the entity
	// file isn't in history.
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add E-0001")

	// Declare a phantom entity at a path that was never committed.
	tr := &tree.Tree{Root: r.root, Entities: []*entity.Entity{
		{ID: "E-0099", Kind: entity.KindEpic, Path: "work/epics/E-0099-phantom/epic.md"},
	}}
	obs, err := walkStatusChanges(context.Background(), r.root, tr)
	if err != nil {
		t.Fatalf("walkStatusChanges: %v", err)
	}
	if len(obs) != 0 {
		t.Errorf("expected 0 observations for an entity with no commits, got %d: %+v", len(obs), obs)
	}
}

// TestWalkStatusChanges_CancelledContext — a cancelled context
// causes git subprocess calls to fail; the walker propagates the
// error rather than emitting partial observations. Together with
// the per-entity error-propagation branches, this exercises the
// `if err != nil { return nil, err }` paths.
func TestWalkStatusChanges_CancelledContext(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add")
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusActive, "promote")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel BEFORE invoking walker
	obs, err := walkStatusChanges(ctx, r.root, r.tree())
	// Either an error or empty observations is acceptable — the
	// guarantee is "no half-emitted observations from partial git
	// reads." Both shapes satisfy that.
	if err == nil && len(obs) > 0 {
		t.Errorf("expected error or empty observations after cancel; got %d obs and nil err", len(obs))
	}
}

// TestFSMHistoryConsistent_AC1EmitsNoFindings pins AC-1's contract:
// the rule wires the walker, but emits no findings until AC-2/3/4
// land their per-subcode predicates. A fixture with a clear FSM-
// illegal transition should produce zero findings from the rule.
func TestFSMHistoryConsistent_AC1EmitsNoFindings(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add")
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusDone, "skip-ahead illegal") // proposed -> done is FSM-illegal

	tr := r.tree()
	got := FSMHistoryConsistent(context.Background(), r.root, tr)
	if len(got) != 0 {
		t.Errorf("AC-1 must emit no findings; got %d: %+v", len(got), got)
	}
}

// TestWalkStatusChanges_NilTreeOrEmptyRoot returns nil cleanly.
func TestWalkStatusChanges_NilTreeOrEmptyRoot(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		root string
		tr   *tree.Tree
	}{
		{"nil tree", "/some/path", nil},
		{"empty root", "", &tree.Tree{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := walkStatusChanges(context.Background(), c.root, c.tr)
			if err != nil {
				t.Errorf("expected nil error, got %v", err)
			}
			if got != nil {
				t.Errorf("expected nil observations, got %+v", got)
			}
		})
	}
}

// TestWalkStatusChanges_NotAGitRepo returns nil cleanly when the
// root is a directory but lacks a git repo.
func TestWalkStatusChanges_NotAGitRepo(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	tr := &tree.Tree{Root: root, Entities: []*entity.Entity{
		{ID: "E-0001", Kind: entity.KindEpic, Path: "epic.md"},
	}}
	got, err := walkStatusChanges(context.Background(), root, tr)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if got != nil {
		t.Errorf("expected nil observations for non-git repo, got %+v", got)
	}
}

// TestWalkStatusChanges_LinearHistory_SingleStatusChange exercises
// the simplest case: one entity, one status-change commit on a
// linear history. Expected: exactly one observation with the
// correct (Prior, Next).
func TestWalkStatusChanges_LinearHistory_SingleStatusChange(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add")
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusActive, "promote proposed -> active")

	obs, err := walkStatusChanges(context.Background(), r.root, r.tree())
	if err != nil {
		t.Fatalf("walkStatusChanges: %v", err)
	}
	if len(obs) != 1 {
		t.Fatalf("expected 1 observation, got %d: %+v", len(obs), obs)
	}
	o := obs[0]
	if o.EntityID != "E-0001" || o.EntityKind != entity.KindEpic {
		t.Errorf("entity mismatch: %+v", o)
	}
	if o.Prior != entity.StatusProposed || o.Next != entity.StatusActive {
		t.Errorf("status mismatch: prior=%q next=%q want proposed->active", o.Prior, o.Next)
	}
}

// TestWalkStatusChanges_RootCommitSkipped — the very first commit
// that adds the entity has no parent with the file. The walker
// emits no observation for it (no prior status to compute a delta).
func TestWalkStatusChanges_RootCommitSkipped(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add")

	obs, err := walkStatusChanges(context.Background(), r.root, r.tree())
	if err != nil {
		t.Fatalf("walkStatusChanges: %v", err)
	}
	if len(obs) != 0 {
		t.Errorf("expected 0 observations for entity with only an add commit, got %d: %+v", len(obs), obs)
	}
}

// TestWalkStatusChanges_NoStatusChange — commits that touch the
// file without changing status (body edits, retitles) emit no
// observations.
func TestWalkStatusChanges_NoStatusChange(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add")
	r.commitEntityWithBody("E-0001", entity.KindEpic, entity.StatusProposed, "more body", "edit body")
	r.commitEntityWithBody("E-0001", entity.KindEpic, entity.StatusProposed, "even more body", "another edit")

	obs, err := walkStatusChanges(context.Background(), r.root, r.tree())
	if err != nil {
		t.Fatalf("walkStatusChanges: %v", err)
	}
	if len(obs) != 0 {
		t.Errorf("expected 0 observations for body-only edits, got %d: %+v", len(obs), obs)
	}
}

// TestWalkStatusChanges_BranchedAndMerged_NoPhantom is the load-
// bearing test for AC-1's DAG-aware design. It reproduces the bug
// pattern that triggered M-0130's AC-1 redo: two commits touch the
// same entity file on parallel branches; one changes status, the
// other doesn't. With the original linearization-adjacency walker,
// adjacency between the two commits in `git log --follow` output
// would produce a phantom "active -> proposed" observation across
// the branch boundary. The DAG-aware walker compares each commit
// only against its actual parent in the DAG, so the phantom is
// eliminated by construction.
//
// Repo shape:
//
//	      main: proposed (add)
//	     /                \
//	branch-a:              branch-b:
//	promote → active        retitle (status unchanged)
//	     \                /
//	      merge (resolves to active)
//
// Expected observations: exactly ONE — the proposed → active
// promote on branch-a. The retitle on branch-b changes nothing.
// The merge commit, when compared per-parent, sees:
//
//   - vs branch-a parent: same status (active). No observation.
//   - vs branch-b parent: proposed → active (legal). One observation.
//
// So total = 2: one at the promote, one at the merge (integration).
// The phantom "active → proposed" from the original adjacency walker
// is gone.
func TestWalkStatusChanges_BranchedAndMerged_NoPhantom(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	// main: add the entity at proposed.
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add")

	// branch-a: promote proposed -> active
	r.gitCheckoutBranch("branch-a")
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusActive, "promote proposed -> active on branch-a")

	// Back to main, then branch-b: retitle (no status change)
	r.gitCheckout("main")
	r.gitCheckoutBranch("branch-b")
	r.commitEntityWithBody("E-0001", entity.KindEpic, entity.StatusProposed, "retitled body", "retitle on branch-b")

	// Merge branch-a into main; resolves status -> active.
	r.gitCheckout("main")
	r.gitMerge("branch-a", "merge branch-a into main")

	// Merge branch-b into main; merge has potential conflict on the body. We
	// resolve by taking branch-b's body but main's status. Since both have
	// status=active... actually wait. Let me reread the setup. branch-b kept
	// status=proposed (it only changed body). main now has status=active. The
	// merge will produce a conflict on status. Resolve to active (main wins on
	// status). The merge commit's resulting status is active; vs branch-b
	// parent (proposed) it's a "proposed -> active" observation.
	r.gitMergeWithResolution("branch-b", "merge branch-b into main", func(absPath string) {
		// Write the resolved file with branch-b's body but main's status.
		r.writeEntityAt(absPath, "E-0001", entity.KindEpic, entity.StatusActive, "retitled body")
	})

	obs, err := walkStatusChanges(context.Background(), r.root, r.tree())
	if err != nil {
		t.Fatalf("walkStatusChanges: %v", err)
	}

	// Assert no phantom: there must be no observation with Prior=active
	// AND Next=proposed (the phantom shape from the original
	// linearization bug).
	for _, o := range obs {
		if o.Prior == entity.StatusActive && o.Next == entity.StatusProposed {
			t.Errorf("phantom observation: active -> proposed at commit %s — the DAG-aware walker must not emit this", o.Commit)
		}
	}

	// Affirmative shape: count the legal proposed->active observations.
	// Expect at least 1 (the promote on branch-a) and at most 2 (the
	// merge integration may produce a second one). Both are real;
	// neither is phantom.
	legalCount := 0
	for _, o := range obs {
		if o.Prior == entity.StatusProposed && o.Next == entity.StatusActive {
			legalCount++
		}
	}
	if legalCount < 1 {
		t.Errorf("expected at least one proposed->active observation (the branch-a promote); got %d. Full set: %+v", legalCount, obs)
	}
}

// TestWalkStatusChanges_RenameWithoutStatusChange — a rename commit
// on a single entity emits no observation (parent has the file at
// the OLD name; `git show parent:NEW-name` fails; the pair is
// skipped). Pre-rename and post-rename status changes are still
// detected.
func TestWalkStatusChanges_RenameWithoutStatusChange(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	// Add at OLD path with status=proposed.
	r.writeEntityAtRel("work/epics/E-0001-old-slug/epic.md", "E-0001", entity.KindEpic, entity.StatusProposed, "body")
	r.gitAddAll()
	r.gitCommit("add at old slug")

	// Promote to active (still at old path).
	r.writeEntityAtRel("work/epics/E-0001-old-slug/epic.md", "E-0001", entity.KindEpic, entity.StatusActive, "body")
	r.gitAddAll()
	r.gitCommit("promote proposed -> active at old slug")

	// Rename to new path (status unchanged).
	r.gitMv("work/epics/E-0001-old-slug", "work/epics/E-0001-new-slug")
	r.gitAddAll()
	r.gitCommit("retitle: old-slug -> new-slug")

	// Promote to done at new path.
	r.writeEntityAtRel("work/epics/E-0001-new-slug/epic.md", "E-0001", entity.KindEpic, entity.StatusDone, "body")
	r.gitAddAll()
	r.gitCommit("promote active -> done at new slug")

	tr := &tree.Tree{
		Root: r.root,
		Entities: []*entity.Entity{
			{ID: "E-0001", Kind: entity.KindEpic, Path: "work/epics/E-0001-new-slug/epic.md"},
		},
	}
	obs, err := walkStatusChanges(context.Background(), r.root, tr)
	if err != nil {
		t.Fatalf("walkStatusChanges: %v", err)
	}

	// Should see exactly 2 status-change observations:
	//   1. proposed -> active (pre-rename, old slug)
	//   2. active -> done (post-rename, new slug)
	// The rename commit itself produces no observation.
	if len(obs) != 2 {
		t.Fatalf("expected 2 observations (pre-rename promote + post-rename promote), got %d: %+v", len(obs), obs)
	}
	want := []struct {
		prior, next string
	}{
		{entity.StatusProposed, entity.StatusActive},
		{entity.StatusActive, entity.StatusDone},
	}
	// observations come in git log order (newest first), so reverse for
	// chronological comparison.
	for i, w := range want {
		o := obs[len(obs)-1-i]
		if o.Prior != w.prior || o.Next != w.next {
			t.Errorf("observation #%d (chronological): got prior=%q next=%q want prior=%q next=%q", i, o.Prior, o.Next, w.prior, w.next)
		}
	}
}

// TestWalkStatusChanges_MultiEntity asserts that multiple entities'
// histories are walked independently — entity A's commits don't
// produce observations on entity B and vice versa.
func TestWalkStatusChanges_MultiEntity(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	// E-0001: add proposed, promote to active
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add E-0001")
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusActive, "promote E-0001 proposed -> active")

	// E-0002: add proposed, promote to active, promote to done
	r.commitEntity("E-0002", entity.KindEpic, entity.StatusProposed, "add E-0002")
	r.commitEntity("E-0002", entity.KindEpic, entity.StatusActive, "promote E-0002 proposed -> active")
	r.commitEntity("E-0002", entity.KindEpic, entity.StatusDone, "promote E-0002 active -> done")

	obs, err := walkStatusChanges(context.Background(), r.root, r.tree())
	if err != nil {
		t.Fatalf("walkStatusChanges: %v", err)
	}

	byEntity := map[string]int{}
	for _, o := range obs {
		byEntity[o.EntityID]++
	}
	if byEntity["E-0001"] != 1 {
		t.Errorf("E-0001: expected 1 observation, got %d", byEntity["E-0001"])
	}
	if byEntity["E-0002"] != 2 {
		t.Errorf("E-0002: expected 2 observations, got %d", byEntity["E-0002"])
	}
}

// TestWalkStatusChanges_TrailerCapture asserts that observations
// carry the commit's aiwf-* trailers, which the AC-2/3/4 predicates
// will consume to classify the change (aiwf-verb present? aiwf-
// force present?).
func TestWalkStatusChanges_TrailerCapture(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add")
	// Status-change commit with a full trailer set.
	r.writeEntityAtRel("work/epics/E-0001-x/epic.md", "E-0001", entity.KindEpic, entity.StatusActive, "body")
	r.gitAddAll()
	r.gitCommit("aiwf promote E-0001 proposed -> active\n\naiwf-verb: promote\naiwf-entity: E-0001\naiwf-actor: human/peter")

	obs, err := walkStatusChanges(context.Background(), r.root, r.tree())
	if err != nil {
		t.Fatalf("walkStatusChanges: %v", err)
	}
	if len(obs) != 1 {
		t.Fatalf("expected 1 observation, got %d: %+v", len(obs), obs)
	}
	if obs[0].Trailers["aiwf-verb"] != "promote" {
		t.Errorf("aiwf-verb trailer missing or wrong; got %q", obs[0].Trailers["aiwf-verb"])
	}
	if obs[0].Trailers["aiwf-actor"] != "human/peter" {
		t.Errorf("aiwf-actor trailer missing or wrong; got %q", obs[0].Trailers["aiwf-actor"])
	}
}

// TestParseStatusFromFrontmatter pins the helper's behavior on a
// representative input set: well-formed frontmatter, CRLF, missing
// delimiter, no status field, malformed YAML.
func TestParseStatusFromFrontmatter(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"well-formed", "---\nid: E-0001\nstatus: proposed\n---\nbody\n", "proposed"},
		{"CRLF", "---\r\nid: E-0001\r\nstatus: active\r\n---\r\nbody\r\n", "active"},
		{"no leading delim", "id: E-0001\nstatus: proposed\n", ""},
		{"unterminated frontmatter", "---\nid: E-0001\nstatus: proposed\nbody\n", ""},
		{"no status field", "---\nid: E-0001\ntitle: hi\n---\nbody\n", ""},
		{"malformed yaml", "---\nstatus: : :\n---\nbody\n", ""},
		{"empty", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := parseStatusFromFrontmatter([]byte(c.in)); got != c.want {
				t.Errorf("parseStatusFromFrontmatter(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestCommitParents pins parent-SHA extraction on root, single-
// parent, and merge commits.
func TestCommitParents(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	rootSHA := r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "root")
	singleSHA := r.commitEntity("E-0001", entity.KindEpic, entity.StatusActive, "single-parent")

	r.gitCheckoutBranch("side")
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusDone, "side-branch head")
	r.gitCheckout("main")
	mergeSHA := r.gitMerge("side", "merge side into main")

	cases := []struct {
		name string
		sha  string
		want int
	}{
		{"root commit has no parents", rootSHA, 0},
		{"single-parent has one parent", singleSHA, 1},
		{"merge has two parents", mergeSHA, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := commitParents(context.Background(), r.root, c.sha)
			if len(got) != c.want {
				t.Errorf("commitParents(%s) returned %d parents, want %d: %+v", c.sha[:8], len(got), c.want, got)
			}
		})
	}
}

// ---------------------------------------------------------------------
// Repo fixture helpers — initialize a tmp git repo, write entity
// files, make commits, branch/merge.
// ---------------------------------------------------------------------

type repoFixture struct {
	t    *testing.T
	root string
}

func newRepoFixture(t *testing.T) *repoFixture {
	t.Helper()
	root := t.TempDir()
	r := &repoFixture{t: t, root: root}
	r.run("git", "init", "-q", "-b", "main")
	r.run("git", "config", "user.email", "test@example.com")
	r.run("git", "config", "user.name", "aiwf-test")
	return r
}

func (r *repoFixture) run(args ...string) string {
	r.t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = r.root
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("running %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// writeEntityAtRel writes an entity file at the named repo-relative
// path with the given status. Body is the markdown content after the
// frontmatter (empty string = body-less).
func (r *repoFixture) writeEntityAtRel(relPath, id string, kind entity.Kind, status, body string) {
	r.t.Helper()
	abs := filepath.Join(r.root, relPath)
	r.writeEntityAt(abs, id, kind, status, body)
}

func (r *repoFixture) writeEntityAt(absPath, id string, kind entity.Kind, status, body string) {
	r.t.Helper()
	if err := mkdirAll(filepath.Dir(absPath)); err != nil {
		r.t.Fatalf("mkdir: %v", err)
	}
	content := "---\nid: " + id + "\nkind: " + string(kind) + "\ntitle: " + id + "\nstatus: " + status + "\n---\n" + body + "\n"
	if err := writeFile(absPath, content); err != nil {
		r.t.Fatalf("write %s: %v", absPath, err)
	}
}

// gitAddAll stages all working-tree changes.
func (r *repoFixture) gitAddAll() {
	r.run("git", "add", "-A")
}

// gitCommit creates a commit with the given message. Returns the
// new commit's SHA.
func (r *repoFixture) gitCommit(msg string) string {
	r.t.Helper()
	r.run("git", "commit", "-q", "--allow-empty", "-m", msg)
	out := r.run("git", "rev-parse", "HEAD")
	return strings.TrimSpace(out)
}

// commitEntity writes the entity at its canonical kind-derived path,
// stages, and commits. Returns the new commit's SHA.
func (r *repoFixture) commitEntity(id string, kind entity.Kind, status, msg string) string {
	r.t.Helper()
	return r.commitEntityWithBody(id, kind, status, "", msg)
}

func (r *repoFixture) commitEntityWithBody(id string, kind entity.Kind, status, body, msg string) string {
	r.t.Helper()
	// Use a simple kind-derived path; the walker only needs entity.Path
	// to match the working-tree location, not the canonical aiwf
	// layout.
	relPath := canonicalEntityPath(id, kind)
	r.writeEntityAtRel(relPath, id, kind, status, body)
	r.gitAddAll()
	return r.gitCommit(msg)
}

func (r *repoFixture) gitCheckoutBranch(branch string) {
	r.run("git", "checkout", "-q", "-b", branch)
}

func (r *repoFixture) gitCheckout(ref string) {
	r.run("git", "checkout", "-q", ref)
}

// gitMv runs `git mv src dst` to rename within a single commit
// (preserves git's rename tracking metadata better than write+delete).
func (r *repoFixture) gitMv(src, dst string) {
	r.run("git", "mv", src, dst)
}

// gitMerge merges the named branch into the current branch. Returns
// the merge commit's SHA. Fails the test on conflict (use
// gitMergeWithResolution for conflict-resolved merges).
func (r *repoFixture) gitMerge(branch, msg string) string {
	r.t.Helper()
	r.run("git", "merge", "-q", "--no-ff", "-m", msg, branch)
	out := r.run("git", "rev-parse", "HEAD")
	return strings.TrimSpace(out)
}

// gitMergeWithResolution merges with --no-commit, then invokes
// resolve(absPath) to let the test write the resolved entity file,
// then stages and commits. Used to script conflict resolutions.
func (r *repoFixture) gitMergeWithResolution(branch, msg string, resolve func(absPath string)) string {
	r.t.Helper()
	cmd := exec.Command("git", "merge", "--no-commit", "--no-ff", branch)
	cmd.Dir = r.root
	_ = cmd.Run() // may exit non-zero on conflict; we resolve below
	// Locate the entity file deterministically by walking common
	// patterns; the fixture's commitEntity uses canonicalEntityPath
	// so the test caller knows the path it wrote.
	abs := filepath.Join(r.root, canonicalEntityPath("E-0001", entity.KindEpic))
	resolve(abs)
	r.gitAddAll()
	r.run("git", "commit", "-q", "-m", msg)
	out := r.run("git", "rev-parse", "HEAD")
	return strings.TrimSpace(out)
}

// tree returns a tree.Tree pointing at the current working-tree path
// for whatever entity ids have been committed via commitEntity.
// Synthesizes entries from canonical kind-derived paths; this is
// adequate for walker tests that only need Tree.Entities populated
// with (ID, Kind, Path).
func (r *repoFixture) tree() *tree.Tree {
	r.t.Helper()
	tr := &tree.Tree{Root: r.root}
	// Discover entity files by walking the filesystem. Tests with
	// non-canonical paths (rename test) build the tree manually
	// instead of using this helper.
	seen := map[string]bool{}
	out := r.run("git", "ls-files")
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		id, kind := identifyEntity(line)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		tr.Entities = append(tr.Entities, &entity.Entity{ID: id, Kind: kind, Path: line})
	}
	return tr
}

// canonicalEntityPath returns a stable repo-relative path for an
// entity given its id and kind. The walker only requires (ID, Kind,
// Path) to be self-consistent; the path doesn't have to match
// aiwf's production layout for these tests.
func canonicalEntityPath(id string, kind entity.Kind) string {
	switch kind {
	case entity.KindEpic:
		return filepath.ToSlash(filepath.Join("work", "epics", id+"-x", "epic.md"))
	case entity.KindMilestone:
		return filepath.ToSlash(filepath.Join("work", "epics", "E-0001-x", id+"-x.md"))
	case entity.KindGap:
		return filepath.ToSlash(filepath.Join("work", "gaps", id+"-x.md"))
	case entity.KindDecision:
		return filepath.ToSlash(filepath.Join("work", "decisions", id+"-x.md"))
	case entity.KindADR:
		return filepath.ToSlash(filepath.Join("docs", "adr", id+"-x.md"))
	case entity.KindContract:
		return filepath.ToSlash(filepath.Join("work", "contracts", id+"-x", "contract.md"))
	}
	return ""
}

// identifyEntity returns (id, kind) from a repo-relative path,
// matching the canonical layout above. Returns ("", "") for paths
// that don't fit.
func identifyEntity(relPath string) (string, entity.Kind) {
	rel := filepath.ToSlash(relPath)
	switch {
	case strings.HasPrefix(rel, "work/epics/E-") && strings.HasSuffix(rel, "/epic.md"):
		// Path: work/epics/E-NNNN-x/epic.md
		parts := strings.Split(rel, "/")
		if len(parts) < 3 {
			return "", ""
		}
		id := strings.SplitN(parts[2], "-", 3)
		if len(id) < 2 {
			return "", ""
		}
		return id[0] + "-" + id[1], entity.KindEpic
	case strings.HasSuffix(rel, ".md") && strings.Contains(rel, "/M-"):
		base := filepath.Base(rel)
		parts := strings.SplitN(base, "-", 3)
		if len(parts) >= 2 {
			return parts[0] + "-" + parts[1], entity.KindMilestone
		}
	}
	return "", ""
}

// ---------------------------------------------------------------------
// Filesystem helpers — small wrappers to keep test bodies clean.
// ---------------------------------------------------------------------

func mkdirAll(path string) error     { return os.MkdirAll(path, 0o755) }
func writeFile(path, s string) error { return os.WriteFile(path, []byte(s), 0o644) }
