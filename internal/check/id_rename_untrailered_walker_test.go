package check

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
)

// id_rename_untrailered_walker_test.go — M-0160/AC-4 REFACTOR:
// unit-level coverage of WalkUntrailedIDRenames at the branch
// level (reviewer N-1 from GREEN review).
//
// The integration tests at
// internal/cli/integration/id_rename_untrailered_scenarios_test.go
// exercise the walker via subprocess (invisible to coverage
// tooling). The tests below drive WalkUntrailedIDRenames directly
// against in-process git fixtures so every branch lands in
// coverage:
//
//   - empty merge-base (returns nil)
//   - empty ref (returns nil)
//   - non-existent ref (returns nil)
//   - commit with rename-class trailer (skip path — emits nothing)
//   - commit without trailer + id-bearing rename (emit path —
//     produces the load-bearing record)
//   - commit without trailer + non-entity rename (skip path —
//     non-entity files are out of scope)
//   - partial-id-match (one side parseable, the other not — the
//     rule still emits, falling back to whichever side carries
//     an id)
//
// Plus entityIDFromPath direct branch coverage.

// walkerFixture is a minimal git fixture for the walker tests.
// Mirrors repoFixture's shape but scoped to what the walker needs
// (a trunk-side ref + a feature branch we can rename files on).
type walkerFixture struct {
	t    *testing.T
	root string
}

func newWalkerFixture(t *testing.T) *walkerFixture {
	t.Helper()
	root := t.TempDir()
	f := &walkerFixture{t: t, root: root}
	f.run("git", "init", "-q", "-b", "main")
	f.run("git", "config", "user.email", "test@example.com")
	f.run("git", "config", "user.name", "aiwf-test")
	// Seed an initial empty commit so HEAD has at least one commit.
	f.run("git", "commit", "-q", "--allow-empty", "-m", "seed")
	return f
}

func (f *walkerFixture) run(args ...string) string {
	f.t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = f.root
	out, err := cmd.CombinedOutput()
	if err != nil {
		f.t.Fatalf("running %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func (f *walkerFixture) writeFile(relPath, content string) {
	f.t.Helper()
	abs := filepath.Join(f.root, relPath)
	if err := mkdirAll(filepath.Dir(abs)); err != nil {
		f.t.Fatalf("mkdir: %v", err)
	}
	if err := writeFile(abs, content); err != nil {
		f.t.Fatalf("write %s: %v", relPath, err)
	}
}

func (f *walkerFixture) commit(msg string, trailers ...string) string {
	f.t.Helper()
	f.run("git", "add", "-A")
	body := msg
	if len(trailers) > 0 {
		body += "\n\n" + strings.Join(trailers, "\n")
	}
	f.run("git", "commit", "-q", "-m", body)
	return strings.TrimSpace(f.run("git", "rev-parse", "HEAD"))
}

// TestWalkUntrailedIDRenames_EmptyRef pins the early-return arms.
func TestWalkUntrailedIDRenames_EmptyRef(t *testing.T) {
	t.Parallel()
	f := newWalkerFixture(t)
	if got := WalkUntrailedIDRenames(context.Background(), f.root, ""); got != nil {
		t.Errorf("empty ref: got %d records; want nil", len(got))
	}
	if got := WalkUntrailedIDRenames(context.Background(), "", "refs/heads/main"); got != nil {
		t.Errorf("empty root: got %d records; want nil", len(got))
	}
}

// TestWalkUntrailedIDRenames_NonExistentRef pins the "ref does
// not resolve" path. git merge-base fails → walker returns nil.
func TestWalkUntrailedIDRenames_NonExistentRef(t *testing.T) {
	t.Parallel()
	f := newWalkerFixture(t)
	got := WalkUntrailedIDRenames(context.Background(), f.root, "refs/heads/nonexistent")
	if got != nil {
		t.Errorf("non-existent ref: got %d records; want nil", len(got))
	}
}

// TestWalkUntrailedIDRenames_TrailerExempts pins the canonical-
// path branch: a commit carrying an `aiwf-verb` trailer in the
// rename-class closed set (here: retitle) is skipped by the
// walker, so its rename produces no records — even if the rename
// is of an id-bearing entity file.
func TestWalkUntrailedIDRenames_TrailerExempts(t *testing.T) {
	t.Parallel()
	f := newWalkerFixture(t)

	// Trunk-side: seed an entity file at the original slug.
	f.writeFile("work/gaps/G-0001-original.md",
		"---\nid: G-0001\nkind: gap\ntitle: original\nstatus: open\n---\n")
	f.commit("seed: add G-0001")
	f.run("git", "branch", "trunk-ref")

	// Rename via a commit carrying `aiwf-verb: retitle`. The walker
	// recognizes the trailer and skips the commit.
	f.run("git", "mv",
		"work/gaps/G-0001-original.md",
		"work/gaps/G-0001-renamed.md")
	f.commit("aiwf retitle G-0001",
		"aiwf-verb: retitle",
		"aiwf-entity: G-0001",
		"aiwf-actor: human/test")

	got := WalkUntrailedIDRenames(context.Background(), f.root, "trunk-ref")
	if len(got) != 0 {
		t.Errorf("trailer-exempted rename: got %d records; want 0\n%+v", len(got), got)
	}
}

// TestWalkUntrailedIDRenames_EmitsOnUntrailedRename pins the
// load-bearing primary path: a commit that renames an id-bearing
// entity file with NO rename-class trailer produces exactly one
// record.
func TestWalkUntrailedIDRenames_EmitsOnUntrailedRename(t *testing.T) {
	t.Parallel()
	f := newWalkerFixture(t)

	f.writeFile("work/gaps/G-0001-original.md",
		"---\nid: G-0001\nkind: gap\ntitle: original\nstatus: open\n---\n")
	f.commit("seed: add G-0001")
	f.run("git", "branch", "trunk-ref")

	f.run("git", "mv",
		"work/gaps/G-0001-original.md",
		"work/gaps/G-0001-via-inline-mv.md")
	sha := f.commit("chore: rename G-0001 slug")

	got := WalkUntrailedIDRenames(context.Background(), f.root, "trunk-ref")
	if len(got) != 1 {
		t.Fatalf("got %d records; want 1\n%+v", len(got), got)
	}
	r := got[0]
	if r.SHA != sha {
		t.Errorf("SHA = %q; want %q", r.SHA, sha)
	}
	if r.OldPath != "work/gaps/G-0001-original.md" {
		t.Errorf("OldPath = %q; want work/gaps/G-0001-original.md", r.OldPath)
	}
	if r.NewPath != "work/gaps/G-0001-via-inline-mv.md" {
		t.Errorf("NewPath = %q; want work/gaps/G-0001-via-inline-mv.md", r.NewPath)
	}
	if r.OldID != "G-0001" || r.NewID != "G-0001" {
		t.Errorf("OldID/NewID = %q/%q; want G-0001/G-0001", r.OldID, r.NewID)
	}
}

// TestWalkUntrailedIDRenames_NonEntityRenameIgnored pins the
// non-entity-file skip branch: a non-id-bearing file rename
// (README.md → DOCS.md) produces no records even without a
// trailer.
func TestWalkUntrailedIDRenames_NonEntityRenameIgnored(t *testing.T) {
	t.Parallel()
	f := newWalkerFixture(t)

	f.writeFile("README.md", "# README\n")
	f.commit("seed: add README")
	f.run("git", "branch", "trunk-ref")

	f.run("git", "mv", "README.md", "DOCS.md")
	f.commit("chore: rename README to DOCS")

	got := WalkUntrailedIDRenames(context.Background(), f.root, "trunk-ref")
	if len(got) != 0 {
		t.Errorf("non-entity rename: got %d records; want 0\n%+v", len(got), got)
	}
}

// TestWalkUntrailedIDRenames_NoCommitsInRange pins the empty-range
// arm: HEAD == trunk-ref, so mergeBase..HEAD is empty and the
// walker emits nothing.
func TestWalkUntrailedIDRenames_NoCommitsInRange(t *testing.T) {
	t.Parallel()
	f := newWalkerFixture(t)

	f.writeFile("work/gaps/G-0001-foo.md",
		"---\nid: G-0001\nkind: gap\ntitle: foo\nstatus: open\n---\n")
	f.commit("seed: G-0001")
	f.run("git", "branch", "trunk-ref")
	// HEAD == trunk-ref: no commits in mergeBase..HEAD.

	got := WalkUntrailedIDRenames(context.Background(), f.root, "trunk-ref")
	if len(got) != 0 {
		t.Errorf("empty range: got %d records; want 0\n%+v", len(got), got)
	}
}

// TestEntityIDFromPath pins the entityIDFromPath helper across
// every kind + the non-entity skip branch. Each row exercises a
// distinct PathKind/IDFromPath arm.
func TestEntityIDFromPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		relPath string
		wantOK  bool
		wantID  string
	}{
		{
			name:    "gap (active)",
			relPath: "work/gaps/G-0001-some-slug.md",
			wantOK:  true,
			wantID:  "G-0001",
		},
		{
			name:    "gap (archived)",
			relPath: "work/gaps/archive/G-0050-old.md",
			wantOK:  true,
			wantID:  "G-0050",
		},
		{
			name:    "milestone inside epic dir",
			relPath: "work/epics/E-0001-slug/M-0001-slug.md",
			wantOK:  true,
			wantID:  "M-0001",
		},
		{
			name:    "epic (epic.md)",
			relPath: "work/epics/E-0001-slug/epic.md",
			wantOK:  true,
			wantID:  "E-0001",
		},
		{
			name:    "decision (active)",
			relPath: "work/decisions/D-0001-some.md",
			wantOK:  true,
			wantID:  "D-0001",
		},
		{
			name:    "ADR (active)",
			relPath: "docs/adr/ADR-0001-some.md",
			wantOK:  true,
			wantID:  "ADR-0001",
		},
		{
			name:    "non-entity file at repo root",
			relPath: "README.md",
			wantOK:  false,
		},
		{
			name:    "non-entity file under work/",
			relPath: "work/notes/scratch.md",
			wantOK:  false,
		},
		{
			name:    "non-entity file under docs/",
			relPath: "docs/index.md",
			wantOK:  false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			id, ok := entityIDFromPath(tc.relPath)
			if ok != tc.wantOK {
				t.Errorf("entityIDFromPath(%q): ok = %v; want %v", tc.relPath, ok, tc.wantOK)
			}
			if id != tc.wantID {
				t.Errorf("entityIDFromPath(%q): id = %q; want %q", tc.relPath, id, tc.wantID)
			}
		})
	}
}

// TestCommitHasRenameClassVerb pins the helper's per-trailer
// scan + closed-set check. Five positive (one per rename-class
// verb) plus four negative cases (non-rename verb, empty
// trailer set, non-aiwf-verb trailer key with rename-class
// value, aiwf-verb with non-rename value).
func TestCommitHasRenameClassVerb(t *testing.T) {
	t.Parallel()
	mk := func(key, value string) []gitops.Trailer {
		return []gitops.Trailer{{Key: key, Value: value}}
	}
	for _, verb := range []string{"retitle", "rename", "reallocate", "archive", "move"} {
		t.Run("rename-class verb: "+verb, func(t *testing.T) {
			t.Parallel()
			if !commitHasRenameClassVerb(mk("aiwf-verb", verb)) {
				t.Errorf("commitHasRenameClassVerb(aiwf-verb: %q) = false; want true", verb)
			}
		})
	}
	for _, verb := range []string{"promote", "add", "check", ""} {
		t.Run("non-rename verb: "+verb, func(t *testing.T) {
			t.Parallel()
			if commitHasRenameClassVerb(mk("aiwf-verb", verb)) {
				t.Errorf("commitHasRenameClassVerb(aiwf-verb: %q) = true; want false", verb)
			}
		})
	}
	t.Run("non-aiwf-verb trailer with rename-class value", func(t *testing.T) {
		t.Parallel()
		// A trailer with a rename-class value but the wrong key
		// must NOT match — the closed-set check is keyed on
		// `aiwf-verb:` specifically.
		if commitHasRenameClassVerb(mk("Other-Key", "retitle")) {
			t.Error("commitHasRenameClassVerb on Other-Key trailer = true; want false")
		}
	})
	t.Run("empty trailer set", func(t *testing.T) {
		t.Parallel()
		if commitHasRenameClassVerb(nil) {
			t.Error("commitHasRenameClassVerb(nil) = true; want false")
		}
		if commitHasRenameClassVerb([]gitops.Trailer{}) {
			t.Error("commitHasRenameClassVerb([]) = true; want false")
		}
	})
}
