package verb_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// M-0284/AC-5. The commit-side guard's input is the record, not git's
// report of what the operator changed.
//
// Both vectors below are invisible to a dirty-set guard by construction:
// git has been told to stop reporting the path, so no amount of
// intersecting the plan's paths with `git status` reaches them. What the
// commit carries is decided by HEAD's blobs and the disk, which is what
// the guard has to compare.

// hideFromGitReporting sets an index bit that stops git reporting a
// tracked path's working-copy state, without changing what HEAD records
// or what the commit would carry. `assume-unchanged` and `skip-worktree`
// are the two operators reach for; a sparse checkout sets the second.
func hideFromGitReporting(t *testing.T, root, bit, path string) {
	t.Helper()
	cmd := exec.Command("git", "update-index", bit, "--", path)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git update-index %s %s: %v\n%s", bit, path, err, out)
	}
	status := exec.Command("git", "status", "--short")
	status.Dir = root
	out, err := status.Output()
	if err != nil {
		t.Fatalf("git status: %v", err)
	}
	if strings.Contains(string(out), path) {
		t.Fatalf("fixture is not staging the vector — git still reports %s:\n%s", path, out)
	}
}

// nestedEpicRunner builds an epic directory carrying one milestone, the
// smallest fixture where a move's carried set is larger than the entity
// the verb was handed.
func nestedEpicRunner(t *testing.T) *runner {
	t.Helper()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Alpha epic", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First milestone", testActor, verb.AddOptions{
		EpicID: "E-0001",
		TDD:    "none",
	}))
	return r
}

// TestApply_AssumeUnchangedNestedEntity_Refused is the vector that
// survived M-0283's ignored-path fix. The milestone is tracked, edited,
// and carried into the parent epic's rename commit — and `assume-unchanged`
// keeps every dirty-set query from naming it, so the guard that reads
// those queries has nothing to refuse on.
//
// Measured before the fix: the rename commits `tdd: required` under
// `aiwf-verb: rename` / `aiwf-entity: E-0001`, a policy change to the
// milestone attributed to a different entity's rename, at exit 0.
func TestApply_AssumeUnchangedNestedEntity_Refused(t *testing.T) {
	t.Parallel()
	r := nestedEpicRunner(t)
	path := dirtyEntity(t, r, "M-0001", "tdd: none", "tdd: required")
	hideFromGitReporting(t, r.root, "--assume-unchanged", path)

	res, err := verb.Rename(r.ctx, r.tree(), "E-0001", "renamed-epic-slug", testActor, 0)
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	assertRefusedAndUncommitted(t, r, res.Plan, path)
}

// TestApply_HEADPathAbsentFromDiskUnderMove_Refused is the other side of
// the same comparison, and the call AC-5 reserved for itself: a path the
// record carries and the working tree does not.
//
// gatherCommitOps builds the move's writes by walking disk, so a path
// missing there is never re-written at the destination — while HEAD's
// copy at the old path is never removed either. The commit lands a split
// directory: the epic and its other milestone at the new path, this one
// stranded at the old. Measured, `aiwf check` reports zero errors on the
// result, so refusing here is the only place it is caught.
func TestApply_HEADPathAbsentFromDiskUnderMove_Refused(t *testing.T) {
	t.Parallel()
	r := nestedEpicRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Second milestone", testActor, verb.AddOptions{
		EpicID: "E-0001",
		TDD:    "none",
	}))
	second := r.tree().ByID("M-0002")
	path := filepath.ToSlash(second.Path)

	hideFromGitReporting(t, r.root, "--skip-worktree", path)
	if err := os.Remove(filepath.Join(r.root, second.Path)); err != nil {
		t.Fatalf("removing %s from the working tree: %v", path, err)
	}

	res, err := verb.Rename(r.ctx, r.tree(), "E-0001", "renamed-epic-slug", testActor, 0)
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	assertRefusedAndUncommitted(t, r, res.Plan, path)
}

// TestApply_MissingPathIsClassifiedApart pins the three-bucket split
// itself, not merely that a blocking path is named.
//
// The buckets exist because their remedies differ and offering the wrong
// one is worse than offering none: `git restore` discards work on a
// tracked path, errors on one git never recorded, and is the entire fix
// for one recorded but absent from the working tree. Asserting only that
// the path appears in the message is satisfied by any bucket, so the
// classification would be free to drift.
func TestApply_MissingPathIsClassifiedApart(t *testing.T) {
	t.Parallel()
	r := nestedEpicRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Second milestone", testActor, verb.AddOptions{
		EpicID: "E-0001",
		TDD:    "none",
	}))
	second := filepath.ToSlash(r.tree().ByID("M-0002").Path)
	hideFromGitReporting(t, r.root, "--skip-worktree", second)
	if err := os.Remove(filepath.Join(r.root, filepath.FromSlash(second))); err != nil {
		t.Fatalf("removing %s: %v", second, err)
	}

	res, err := verb.Rename(r.ctx, r.tree(), "E-0001", "renamed-epic-slug", testActor, 0)
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	_, applyErr := verb.Apply(r.ctx, r.root, res.Plan)
	var conflictErr *verb.UncommittedConflictError
	if !errors.As(applyErr, &conflictErr) {
		t.Fatalf("error is not a *verb.UncommittedConflictError: %v", applyErr)
	}
	if !slices.Contains(conflictErr.Missing, second) {
		t.Errorf("%s is not in Missing; Tracked=%v Untracked=%v Missing=%v",
			second, conflictErr.Tracked, conflictErr.Untracked, conflictErr.Missing)
	}
	if slices.Contains(conflictErr.Tracked, second) || slices.Contains(conflictErr.Untracked, second) {
		t.Errorf("%s was also bucketed as tracked or untracked, so its remedy is ambiguous", second)
	}
}

// TestApply_CleanNestedMove_StillCommits is the negative control both
// refusals rest on. The guard now enumerates every path a move carries
// from two sides — the disk walk and HEAD's tree — and compares each. A
// guard that refused an ordinary directory move would satisfy the two
// tests above while making every epic rename unreachable.
func TestApply_CleanNestedMove_StillCommits(t *testing.T) {
	t.Parallel()
	r := nestedEpicRunner(t)
	before := headSHA(t, r.root)

	res, err := verb.Rename(r.ctx, r.tree(), "E-0001", "renamed-epic-slug", testActor, 0)
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("Apply refused a clean directory move: %v", applyErr)
	}
	if after := headSHA(t, r.root); after == before {
		t.Error("HEAD did not advance; the rename committed nothing")
	}
	moved := filepath.Join(r.root, "work", "epics", "E-0001-renamed-epic-slug", "M-0001-first-milestone.md")
	if _, statErr := os.Stat(moved); statErr != nil {
		t.Errorf("the nested milestone did not travel with its epic: %v", statErr)
	}
}

// TestApply_UntrackedFileAtTheMoveDestination_Refused covers the other
// end of a move. os.Rename onto an existing file replaces it silently,
// so a plan whose destination is already occupied would destroy content
// no verb named and nobody was warned about.
//
// The path is untracked, which is what makes it worth pinning
// separately: there is no committed version, so the loss is total.
func TestApply_UntrackedFileAtTheMoveDestination_Refused(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)
	res, err := verb.Rename(r.ctx, r.tree(), "G-0001", "a-new-slug", testActor, 0)
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	var dest string
	for _, op := range res.Plan.Ops {
		if op.Type == verb.OpMove {
			dest = op.NewPath
		}
	}
	if dest == "" {
		t.Fatal("the rename plan carries no move")
	}
	if wErr := os.WriteFile(filepath.Join(r.root, filepath.FromSlash(dest)),
		[]byte("someone else's untracked work\n"), 0o600); wErr != nil {
		t.Fatalf("writing the occupying file: %v", wErr)
	}

	assertRefusedAndUncommitted(t, r, res.Plan, dest)
	raw, readErr := os.ReadFile(filepath.Join(r.root, filepath.FromSlash(dest)))
	if readErr != nil {
		t.Fatalf("the occupying file did not survive the refusal: %v", readErr)
	}
	if !strings.Contains(string(raw), "someone else's untracked work") {
		t.Errorf("the occupying file's content was replaced:\n%s", raw)
	}
}

// TestApply_UnreadableCarriedFile_FailsLoud pins the fail-loud
// direction at this seam. A committed file the guard cannot read is a
// comparison it cannot make, and reading that as "nothing diverges"
// would leave the guard inert exactly where the record is least
// trustworthy.
//
// The file is nested under the move rather than the renamed entity
// itself, which is what makes it reach Apply: the claim-side guard is
// scoped to the epic's own file and has nothing to say about it.
func TestApply_UnreadableCarriedFile_FailsLoud(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	r := nestedEpicRunner(t)
	nested := filepath.Join(r.root, r.tree().ByID("M-0001").Path)
	before := headSHA(t, r.root)

	// The plan is computed while the file is still readable; the guard's
	// read is the one that has to fail, so the mode change lands between
	// planning and Apply.
	res, err := verb.Rename(r.ctx, r.tree(), "E-0001", "renamed-epic-slug", testActor, 0)
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if chmodErr := os.Chmod(nested, 0o000); chmodErr != nil {
		t.Fatalf("chmod: %v", chmodErr)
	}
	t.Cleanup(func() { _ = os.Chmod(nested, 0o644) })

	_, applyErr := verb.Apply(r.ctx, r.root, res.Plan)
	if applyErr == nil {
		t.Fatal("Apply succeeded over a file it could not compare against the record")
	}
	if !strings.Contains(applyErr.Error(), "M-0001") {
		t.Errorf("error does not name the unreadable path:\n%v", applyErr)
	}
	if after := headSHA(t, r.root); after != before {
		t.Errorf("HEAD advanced to %s despite an unmakeable comparison", after)
	}
}

// TestApply_HiddenEditOutsideThePlan_DoesNotBlock pins the scope. The
// guard reads the record for the paths the plan carries, so an operator
// with a hidden edit somewhere else is not blocked by a verb that would
// never commit it. Without this, "compare against HEAD" could be read as
// "compare the whole tree", which would refuse far more than it should.
func TestApply_HiddenEditOutsideThePlan_DoesNotBlock(t *testing.T) {
	t.Parallel()
	r := nestedEpicRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Unrelated gap", testActor, verb.AddOptions{
		BodyOverride: bornCompleteFixtureBody(entity.KindGap),
	}))
	elsewhere := dirtyEntity(t, r, "G-0001", "## Why it matters", "## Why it matters\n\nHIDDEN EDIT.\n")
	hideFromGitReporting(t, r.root, "--assume-unchanged", elsewhere)

	res, err := verb.Rename(r.ctx, r.tree(), "E-0001", "renamed-epic-slug", testActor, 0)
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("an edit outside the plan's paths blocked the verb: %v", applyErr)
	}
}

// TestApply_CarriedSymlink_RefusedRegardlessOfShape pins that a symbolic
// link the plan would carry is refused, whatever it points at and whether
// or not its target string still matches the record.
//
// Divergence is the wrong question for a link. `gatherCommitOps` reads
// each carried path with os.ReadFile, which follows the link, and
// CommitTree stores every write at mode 100644 — so a link whose target
// string is untouched, and which every git query calls clean, is still
// replaced in the record by a copy of whatever it points at. Measured
// before this guard: an epic rename turned a 120000 link into a 100644
// blob holding the linked file's body under `aiwf-verb: rename`, and a
// link pointing outside the repo carried that file's content into git
// history. Both left the working tree reporting a type change with no
// remedy that clears it.
func TestApply_CarriedSymlink_RefusedRegardlessOfShape(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		target func(t *testing.T, root string) string
	}{
		{"a link to a file in the tree", func(t *testing.T, root string) string {
			t.Helper()
			return "M-0001-first-milestone.md"
		}},
		{"a link pointing outside the repo", func(t *testing.T, root string) string {
			t.Helper()
			outside := filepath.Join(t.TempDir(), "creds.env")
			if err := os.WriteFile(outside, []byte("API_KEY=fixture\n"), 0o600); err != nil {
				t.Fatalf("writing the outside file: %v", err)
			}
			return outside
		}},
		{"a dangling link", func(t *testing.T, root string) string {
			t.Helper()
			return "no-such-file.md"
		}},
		{"a link to a directory", func(t *testing.T, root string) string {
			t.Helper()
			return "."
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := nestedEpicRunner(t)
			epicDir := filepath.Dir(r.tree().ByID("E-0001").Path)
			link := filepath.Join(r.root, epicDir, "latest.md")
			if err := os.Symlink(tc.target(t, r.root), link); err != nil {
				t.Skipf("symlinks unavailable on this platform: %v", err)
			}
			commitFixture(t, r.root, "fixture: a tracked symlink")
			before := headSHA(t, r.root)

			res, err := verb.Rename(r.ctx, r.tree(), "E-0001", "renamed-epic-slug", testActor, 0)
			if err != nil {
				t.Fatalf("Rename: %v", err)
			}
			_, applyErr := verb.Apply(r.ctx, r.root, res.Plan)
			var linkErr *verb.CarriedSymlinkError
			if !errors.As(applyErr, &linkErr) {
				t.Fatalf("error is not a *verb.CarriedSymlinkError: %v", applyErr)
			}
			rel := filepath.ToSlash(filepath.Join(epicDir, "latest.md"))
			if !slices.Contains(linkErr.Paths, rel) {
				t.Errorf("refusal does not name the link; names %v", linkErr.Paths)
			}
			// The message is the operator's whole diagnosis here: nothing
			// in `git status` explains why a clean tree was refused.
			msg := linkErr.Error()
			if !strings.Contains(msg, rel) {
				t.Errorf("message does not name the link:\n%s", msg)
			}
			for _, want := range []string{"symbolic link", "regular", "points at"} {
				if !strings.Contains(msg, want) {
					t.Errorf("message does not explain %q:\n%s", want, msg)
				}
			}
			if after := headSHA(t, r.root); after != before {
				t.Errorf("HEAD advanced to %s; the link was rewritten into the record", after)
			}
		})
	}
}

// TestApply_NoSymlink_StillCommits is the negative control: the refusal
// is about links specifically, not about directory moves.
func TestApply_NoSymlink_StillCommits(t *testing.T) {
	t.Parallel()
	r := nestedEpicRunner(t)
	before := headSHA(t, r.root)

	res, err := verb.Rename(r.ctx, r.tree(), "E-0001", "renamed-epic-slug", testActor, 0)
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("Apply refused a link-free directory move: %v", applyErr)
	}
	if after := headSHA(t, r.root); after == before {
		t.Error("HEAD did not advance")
	}
}
