package verb_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/verb"
)

// editbody_same_state_noop_test.go covers M-0281/AC-8: `edit-body --body-file`
// converges to a NoOp when the content it would write is already both committed
// and on disk, instead of landing an empty-diff commit on every repeat.
//
// Convergence is judged on the SERIALIZED entity, not the body bytes; against
// BOTH HEAD and the working copy; and never when there is no committed version
// at all. The tests below pin each of those four choices with a case that fails
// if the comparison is narrowed.

// commitAllForEditBody stages every change in root and commits it, so a test can
// establish a HEAD version that differs from what the verb would serialize.
func commitAllForEditBody(t *testing.T, root, subject string) {
	t.Helper()
	if err := gitops.Add(t.Context(), root, "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := gitops.Commit(t.Context(), root, subject, "", nil); err != nil {
		t.Fatalf("git commit: %v", err)
	}
}

// writeLooseEpicOnly plants an epic on disk without committing it, bypassing the
// verb layer the way a hand-authored or imported tree does. The loader keys on
// the frontmatter id rather than the filename, so the entity resolves — and
// `aiwf check` reports no error on the result — while having no version in HEAD
// at all. That combination is what makes the arms these fixtures drive reachable
// from a supported state rather than merely defensive.
func writeLooseEpicOnly(t *testing.T, root, dir, id, title string) {
	t.Helper()
	full := filepath.Join(root, "work", "epics", dir)
	if err := os.MkdirAll(full, 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	body := "---\nid: " + id + "\ntitle: " + title + "\nstatus: proposed\n---\n## Goal\n\nFixture prose for test setup; not the subject under test.\n\n## Scope\n\nFixture prose for test setup; not the subject under test.\n"
	if err := os.WriteFile(filepath.Join(full, "epic.md"), []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", dir, err)
	}
}

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

// TestEditBody_ExplicitIdenticalBodyOverNonCanonicalFrontmatter_StillCommits
// pins the third of the comparison's three choices: it is made on the
// SERIALIZED entity, not on the body bytes.
//
// A committed file whose frontmatter keys sit in a non-canonical order carries
// the same body but not the same serialization, so re-canonicalizing it is real
// work. Comparing body bytes instead converges here and strands that work —
// invisibly, because the body genuinely does match.
func TestEditBody_ExplicitIdenticalBodyOverNonCanonicalFrontmatter_StillCommits(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	path, body := epicBodyOnDisk(t, r.root)
	canonical, err := os.ReadFile(path) //nolint:gosec // fixture path inside the test's own temp root
	if err != nil {
		t.Fatalf("reading the canonical epic: %v", err)
	}

	// Commit a version whose frontmatter is valid but not canonically ordered,
	// then restore the canonical bytes to disk uncommitted. HEAD and the working
	// copy now carry the same BODY but different serializations, which is the
	// only arrangement that distinguishes a serialized comparison from a
	// body-bytes one: comparing bodies finds both sides equal and converges,
	// stranding the re-canonicalization.
	noncanon := []byte("---\nstatus: proposed\ntitle: Foundations\nid: E-0001\n---\n" + string(body))
	if err = os.WriteFile(path, noncanon, 0o600); err != nil {
		t.Fatalf("writing non-canonical frontmatter: %v", err)
	}
	commitAllForEditBody(t, r.root, "hand-ordered frontmatter")
	if err = os.WriteFile(path, canonical, 0o600); err != nil {
		t.Fatalf("restoring the canonical epic: %v", err)
	}

	res, err := verb.EditBody(r.ctx, r.tree(), "E-0001", body, testActor, "")
	if err != nil {
		t.Fatalf("edit-body over a non-canonically-serialized HEAD: %v", err)
	}
	if res.NoOp {
		t.Fatal("res.NoOp = true, want false — the frontmatter still needs re-canonicalizing")
	}
	if res.Plan == nil {
		t.Fatal("res.Plan = nil, want a plan re-canonicalizing the entity")
	}
}

// TestEditBody_ExplicitOnNeverCommittedEntity_StillCommits pins the
// no-committed-version arm. Explicit mode is the sanctioned route for an entity
// that exists only in the working tree — bless mode refuses exactly that case
// and redirects here — so it must never converge for want of a HEAD version.
//
// Treating a missing HEAD as "already carries this body" would report success
// while the entity stayed uncommitted, which is the false-NoOp shape this AC's
// own design note calls out.
func TestEditBody_ExplicitOnNeverCommittedEntity_StillCommits(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	// A second epic planted on disk only: never committed, so it has no HEAD
	// version, while the loader resolves it by its frontmatter id.
	writeLooseEpicOnly(t, r.root, "E-0002-uncommitted", "E-0002", "Uncommitted epic")

	e := r.tree().ByID("E-0002")
	if e == nil {
		t.Fatal("E-0002 missing from the fixture tree")
	}
	// Hand it the body already on disk. That is the case a missing-HEAD guard
	// gets wrong: the disk comparison matches, so treating a nil HEAD as settled
	// converges and the entity is never committed at all.
	raw, err := os.ReadFile(filepath.Join(r.root, e.Path)) //nolint:gosec // fixture path inside the test's own temp root
	if err != nil {
		t.Fatalf("reading the loose epic: %v", err)
	}
	_, body, ok := entity.Split(raw)
	if !ok {
		t.Fatalf("loose epic has no frontmatter:\n%s", raw)
	}

	res, err := verb.EditBody(r.ctx, r.tree(), "E-0002", body, testActor, "")
	if err != nil {
		t.Fatalf("edit-body on a never-committed entity: %v", err)
	}
	if res.NoOp {
		t.Fatal("res.NoOp = true, want false — nothing is committed, so nothing can already match")
	}
	if res.Plan == nil {
		t.Fatal("res.Plan = nil, want a plan committing the body")
	}
}
