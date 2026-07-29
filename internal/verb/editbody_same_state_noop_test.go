package verb_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// editbody_same_state_noop_test.go covers M-0281/AC-8: `edit-body --body-file`
// converges to a NoOp when the content it would write is already both committed
// and on disk, instead of landing an empty-diff commit on every repeat.
//
// Convergence is judged on the SERIALIZED entity, not the body bytes, and
// against BOTH HEAD and the working copy. The tests below pin each of those
// three choices with a case that fails if the comparison is narrowed.

// epicBodyOnDisk returns the current body of E-0001 and its file path.
func epicBodyOnDisk(t *testing.T, root string) (path string, body []byte) {
	t.Helper()
	path = filepath.Join(root, "work", "epics", "E-0001-foundations", "epic.md")
	raw, err := os.ReadFile(path) //nolint:gosec // fixture path inside the test's own temp root
	if err != nil {
		t.Fatalf("reading the epic: %v", err)
	}
	_, body, ok := entity.Split(raw)
	if !ok {
		t.Fatalf("epic file has no frontmatter:\n%s", raw)
	}
	return path, body
}

// TestEditBody_ExplicitIdenticalContent_ReturnsNoOp is AC-8's primary case: a
// clean tree handed the body it already carries has nothing to write, so the
// verb converges rather than committing an empty diff. The commit count is
// asserted, because an empty-diff commit is exactly what this exists to stop —
// a Result that merely looks right would not catch a regression.
func TestEditBody_ExplicitIdenticalContent_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	_, body := epicBodyOnDisk(t, r.root)
	before := countCommits(t, r.root)

	res, err := verb.EditBody(r.ctx, r.tree(), "E-0001", body, testActor, "")
	if err != nil {
		t.Fatalf("edit-body with byte-identical content returned a Go error, want a NoOp: %v", err)
	}
	if !res.NoOp {
		t.Errorf("res.NoOp = false, want true — HEAD and the working copy both already carry this body")
	}
	if res.Plan != nil {
		t.Errorf("res.Plan = %+v, want nil — applying it would land a commit with an empty diff", res.Plan)
	}
	if got := countCommits(t, r.root); got != before {
		t.Errorf("commit count = %s, want %s (the NoOp must append no commit)", got, before)
	}
}

// TestEditBody_ExplicitUncommittedMatchingContent_StillCommits pins why the
// comparison cannot be made against the working copy alone. Here the working
// copy already holds the requested body but HEAD does not — the shape produced
// by writing a file and then routing it through the verb, which the project's
// own guidance encourages. Converging would silently never land the commit that
// was the entire point of the call.
func TestEditBody_ExplicitUncommittedMatchingContent_StillCommits(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	path, _ := epicBodyOnDisk(t, r.root)

	// Put the new body on disk without committing it.
	raw, err := os.ReadFile(path) //nolint:gosec // fixture path inside the test's own temp root
	if err != nil {
		t.Fatalf("reading the epic: %v", err)
	}
	fm, _, ok := entity.Split(raw)
	if !ok {
		t.Fatalf("epic file has no frontmatter:\n%s", raw)
	}
	const wanted = "## Goal\n\nUncommitted edit already on disk.\n"
	staged := append(append([]byte("---\n"), fm...), append([]byte("---\n"), wanted...)...)
	if writeErr := os.WriteFile(path, staged, 0o600); writeErr != nil {
		t.Fatalf("writing the uncommitted body: %v", writeErr)
	}

	res, err := verb.EditBody(r.ctx, r.tree(), "E-0001", []byte(wanted), testActor, "")
	if err != nil {
		t.Fatalf("edit-body over an uncommitted matching body: %v", err)
	}
	if res.NoOp {
		t.Errorf("res.NoOp = true, want false — the body is on disk but not committed, so there is a commit to land")
	}
	if res.Plan == nil {
		t.Fatal("res.Plan = nil, want a plan that commits the on-disk body")
	}
}

// TestEditBody_ExplicitMatchingBodyOverDirtyWorkingCopy_StillCommits pins why
// the comparison cannot be made against HEAD alone. The working copy carries an
// unwanted uncommitted edit and the operator asks for the committed content
// back — a revert, expressed declaratively. Judging on HEAD alone would report
// "already carries this body" while leaving the dirty file untouched: a NoOp
// that is true of git and false of the file the operator is looking at.
func TestEditBody_ExplicitMatchingBodyOverDirtyWorkingCopy_StillCommits(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	path, committed := epicBodyOnDisk(t, r.root)

	raw, err := os.ReadFile(path) //nolint:gosec // fixture path inside the test's own temp root
	if err != nil {
		t.Fatalf("reading the epic: %v", err)
	}
	fm, _, ok := entity.Split(raw)
	if !ok {
		t.Fatalf("epic file has no frontmatter:\n%s", raw)
	}
	dirty := append(append([]byte("---\n"), fm...), []byte("---\n## Goal\n\nUnwanted local edit.\n")...)
	if writeErr := os.WriteFile(path, dirty, 0o600); writeErr != nil {
		t.Fatalf("dirtying the working copy: %v", writeErr)
	}

	res, err := verb.EditBody(r.ctx, r.tree(), "E-0001", committed, testActor, "")
	if err != nil {
		t.Fatalf("edit-body reverting a dirty working copy: %v", err)
	}
	if res.NoOp {
		t.Errorf("res.NoOp = true, want false — the working copy differs, so claiming the body already matches would be false")
	}
	if res.Plan == nil {
		t.Fatal("res.Plan = nil, want a plan that restores the committed body")
	}
}

// TestEditBody_BlessOnACleanTree_StillRefuses pins the other mode's outcome
// against this change. Bless mode is handed no target — its input is the
// current state — so it cannot tell "I meant to change nothing" from "my editor
// did not save", and a refusal is the only honest answer. It must not drift
// into a NoOp alongside explicit mode.
func TestEditBody_BlessOnACleanTree_StillRefuses(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))

	res, err := verb.EditBody(r.ctx, r.tree(), "E-0001", nil, testActor, "")
	if err == nil {
		t.Fatalf("bless mode on a clean tree returned res=%+v, want a refusal", res)
	}
	if !strings.Contains(err.Error(), "no changes to commit") {
		t.Errorf("err = %q, want the bless-mode refusal naming that there is no edit to commit", err)
	}
}
