package check

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// entity_body_section_dropped_test.go — M-0331/AC-1 branch coverage for the
// push-seam membership gate. The seam test in internal/cli/check drives the
// real reader over a real range; these drive WalkDroppedBodySections directly
// so each rule it applies lands in coverage with its own case.
//
// One test per rule the walker decides, not per input that could reach it.

const (
	droppedSpecPath = "work/epics/E-0001-seed/M-0001-seed.md"

	droppedSpecWhole = `---
id: M-0001
title: Seed
status: in_progress
parent: E-0001
---
## Goal

Ship it.

## Acceptance criteria

### AC-1 — It ships

It ships.
`

	droppedSpecPartial = `---
id: M-0001
title: Seed
status: in_progress
parent: E-0001
---
## Goal

Ship it.
`
)

// commitAt writes content to relPath and commits it, returning the new SHA and
// the SHA it replaced, so a caller can hand WalkDroppedBodySections a commit
// record without going through the reader under test elsewhere.
func commitAt(f *walkerFixture, relPath, content, msg string) (sha, parent string) {
	f.t.Helper()
	parent = strings.TrimSpace(f.run("git", "rev-parse", "HEAD"))
	f.writeFile(relPath, content)
	return f.commit(msg), parent
}

func TestWalkDroppedBodySections(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// A section the body carried and the commit removed is the whole subject
	// of the rule; every case below states an exception to it.
	t.Run("reports a section the commit removed", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		commitAt(f, droppedSpecPath, droppedSpecWhole, "seed the spec")
		sha, parent := commitAt(f, droppedSpecPath, droppedSpecPartial, "wrap")
		got := WalkDroppedBodySections(ctx, f.root, []UntrailedCommit{
			{SHA: sha, ParentSHAs: []string{parent}, Paths: []string{droppedSpecPath}},
		})
		want := []DroppedBodySection{{
			SHA: sha, Path: droppedSpecPath, EntityID: "M-0001", Section: "Acceptance criteria",
		}}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("(-want +got):\n%s", diff)
		}
	})

	// An ordinary `--no-ff` merge republishes content another branch already
	// pushed, and that branch's own push judged it.
	t.Run("skips a merge that only absorbed the change", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		commitAt(f, droppedSpecPath, droppedSpecWhole, "seed the spec")
		sha, parent := commitAt(f, droppedSpecPath, droppedSpecPartial, "Merge branch 'other'")
		got := WalkDroppedBodySections(ctx, f.root, []UntrailedCommit{
			{SHA: sha, ParentSHAs: []string{parent, "deadbeef"}, Subject: "Merge branch 'other'", Paths: []string{droppedSpecPath}},
		})
		if len(got) != 0 {
			t.Errorf("merge commit judged: %+v", got)
		}
	})

	// A squash carries no trailers and is the integration branch's only record
	// of the collapsed work, so it is judged despite having two parents.
	t.Run("judges a squash merge", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		commitAt(f, droppedSpecPath, droppedSpecWhole, "seed the spec")
		sha, parent := commitAt(f, droppedSpecPath, droppedSpecPartial, "wrap the spec (#42)")
		got := WalkDroppedBodySections(ctx, f.root, []UntrailedCommit{
			{SHA: sha, ParentSHAs: []string{parent, "deadbeef"}, Subject: "wrap the spec (#42)", Paths: []string{droppedSpecPath}},
		})
		if len(got) != 1 {
			t.Errorf("squash merge not judged; got %+v", got)
		}
	})

	// With no parent there is no baseline to regress from.
	t.Run("skips a commit with no recorded parent", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		commitAt(f, droppedSpecPath, droppedSpecWhole, "seed the spec")
		sha, _ := commitAt(f, droppedSpecPath, droppedSpecPartial, "wrap")
		got := WalkDroppedBodySections(ctx, f.root, []UntrailedCommit{
			{SHA: sha, Paths: []string{droppedSpecPath}},
		})
		if len(got) != 0 {
			t.Errorf("parentless commit judged: %+v", got)
		}
	})

	// `aiwf add` holds a create to completeness and its --force override stays
	// in force afterwards, so re-judging the same body here would revoke it.
	t.Run("skips an entity the commit created", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		sha, parent := commitAt(f, droppedSpecPath, droppedSpecPartial, "create incomplete")
		got := WalkDroppedBodySections(ctx, f.root, []UntrailedCommit{
			{SHA: sha, ParentSHAs: []string{parent}, Paths: []string{droppedSpecPath}},
		})
		if len(got) != 0 {
			t.Errorf("create judged: %+v", got)
		}
	})

	// A path gone at the commit has no body to judge.
	t.Run("skips an entity the commit deleted", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		commitAt(f, droppedSpecPath, droppedSpecWhole, "seed the spec")
		parent := strings.TrimSpace(f.run("git", "rev-parse", "HEAD"))
		f.run("git", "rm", "-q", droppedSpecPath)
		sha := f.commit("delete the spec")
		got := WalkDroppedBodySections(ctx, f.root, []UntrailedCommit{
			{SHA: sha, ParentSHAs: []string{parent}, Paths: []string{droppedSpecPath}},
		})
		if len(got) != 0 {
			t.Errorf("delete judged: %+v", got)
		}
	})

	// The rule judges what the push publishes; refusing over an intermediate
	// state would leave the author rewriting history to satisfy it.
	t.Run("skips a drop a later commit in the range restored", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		commitAt(f, droppedSpecPath, droppedSpecWhole, "seed the spec")
		dropSHA, dropParent := commitAt(f, droppedSpecPath, droppedSpecPartial, "drop it")
		restoreSHA, restoreParent := commitAt(f, droppedSpecPath, droppedSpecWhole, "put it back")
		got := WalkDroppedBodySections(ctx, f.root, []UntrailedCommit{
			{SHA: dropSHA, ParentSHAs: []string{dropParent}, Paths: []string{droppedSpecPath}},
			{SHA: restoreSHA, ParentSHAs: []string{restoreParent}, Paths: []string{droppedSpecPath}},
		})
		if len(got) != 0 {
			t.Errorf("self-corrected drop reported: %+v", got)
		}
	})

	// A delete and a recreate compose to a create, which `aiwf add` holds.
	// Without the delete arm the removing commit reads as dropping every
	// section at once, and whichever of them the recreated body still omits
	// survives the HEAD confirmation and is reported against the delete.
	t.Run("skips a delete whose entity a later commit recreated incomplete", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		commitAt(f, droppedSpecPath, droppedSpecWhole, "seed the spec")
		delParent := strings.TrimSpace(f.run("git", "rev-parse", "HEAD"))
		f.run("git", "rm", "-q", droppedSpecPath)
		delSHA := f.commit("delete the spec")
		addSHA, addParent := commitAt(f, droppedSpecPath, droppedSpecPartial, "recreate it")
		got := WalkDroppedBodySections(ctx, f.root, []UntrailedCommit{
			{SHA: delSHA, ParentSHAs: []string{delParent}, Paths: []string{droppedSpecPath}},
			{SHA: addSHA, ParentSHAs: []string{addParent}, Paths: []string{droppedSpecPath}},
		})
		if len(got) != 0 {
			t.Errorf("delete+recreate judged as a drop: %+v", got)
		}
	})

	// A file under work/ that carries no frontmatter is not an entity; the
	// unexpected-tree-file rule is what reports it.
	t.Run("skips a path carrying no entity frontmatter", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		commitAt(f, droppedSpecPath, droppedSpecWhole, "seed the spec")
		sha, parent := commitAt(f, droppedSpecPath, "## Goal\n\nno frontmatter\n", "strip frontmatter")
		got := WalkDroppedBodySections(ctx, f.root, []UntrailedCommit{
			{SHA: sha, ParentSHAs: []string{parent}, Paths: []string{droppedSpecPath}},
		})
		if len(got) != 0 {
			t.Errorf("frontmatter-less file judged: %+v", got)
		}
	})

	// Only entity paths are in scope.
	t.Run("skips a path that is not an entity file", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		sha, parent := commitAt(f, "README.md", "# readme\n", "touch the readme")
		got := WalkDroppedBodySections(ctx, f.root, []UntrailedCommit{
			{SHA: sha, ParentSHAs: []string{parent}, Paths: []string{"README.md"}},
		})
		if len(got) != 0 {
			t.Errorf("non-entity path judged: %+v", got)
		}
	})

	// An epic directory carrying no id in its name is epic-shaped to PathKind
	// and gives IDFromPath nothing to extract, so the finding would have no
	// entity to name.
	t.Run("skips an entity path whose id does not parse", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		const noID = "work/epics/no-id-here/epic.md"
		const whole = "---\nid: E-0001\n---\n## Goal\n\ng\n\n## Scope\n\ns\n\n## Out of scope\n\no\n"
		const partial = "---\nid: E-0001\n---\n## Goal\n\ng\n\n## Scope\n\ns\n"
		commitAt(f, noID, whole, "seed")
		sha, parent := commitAt(f, noID, partial, "drop a section")
		got := WalkDroppedBodySections(ctx, f.root, []UntrailedCommit{
			{SHA: sha, ParentSHAs: []string{parent}, Paths: []string{noID}},
		})
		if len(got) != 0 {
			t.Errorf("unparseable-id path judged: %+v", got)
		}
	})
}

// TestWalkDroppedBodySections_WhatCountsAsAViolation — M-0331/AC-2 states the
// gate's scope; this states what inside that scope is a violation. Exactly one
// thing: a section the kind requires, not present as a top-level `## ` heading.
// Sections beyond the declared set are legal, order carries no meaning, and a
// required heading nested below top level is absent — `ParseBodySections`, the
// parser `aiwf show` and the body rules already share, reads `## ` alone, so a
// nested one yields no key on any read path.
func TestWalkDroppedBodySections_WhatCountsAsAViolation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	cases := []struct {
		name  string
		after string
		want  []string // sections reported, in the kind's canonical order
	}{
		{
			name: "a section beyond the declared set is legal",
			after: droppedSpecWhole + `
## Notes

An author's own heading, which no kind declares.
`,
		},
		{
			name: "order is not enforced",
			after: `---
id: M-0001
title: Seed
status: in_progress
parent: E-0001
---
## Acceptance criteria

### AC-1 — It ships

It ships.

## Goal

Ship it.
`,
		},
		{
			name: "a required heading nested below top level is absent",
			after: `---
id: M-0001
title: Seed
status: in_progress
parent: E-0001
---
## Goal

Ship it.

### Acceptance criteria

### AC-1 — It ships

It ships.
`,
			want: []string{"Acceptance criteria"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newWalkerFixture(t)
			commitAt(f, droppedSpecPath, droppedSpecWhole, "seed the spec")
			sha, parent := commitAt(f, droppedSpecPath, tc.after, "rework the body")
			var got []string
			for _, d := range WalkDroppedBodySections(ctx, f.root, []UntrailedCommit{
				{SHA: sha, ParentSHAs: []string{parent}, Paths: []string{droppedSpecPath}},
			}) {
				got = append(got, d.Section)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("reported sections (-want +got):\n%s", diff)
			}
		})
	}
}

// TestRunEntityBodySectionDropped_AcknowledgedSHAIsExempt pins the only escape
// this rule has. No verb on the path carries --force, so a deliberate removal
// reaches the push as a finding and `aiwf acknowledge illegal` is what clears
// it; without the exemption the removal could never be pushed at all.
func TestRunEntityBodySectionDropped_AcknowledgedSHAIsExempt(t *testing.T) {
	t.Parallel()
	dropped := []DroppedBodySection{
		{SHA: "aaaa111", Path: "work/gaps/G-0001-x.md", EntityID: "G-0001", Section: "Why it matters"},
		{SHA: "bbbb222", Path: "work/gaps/G-0002-y.md", EntityID: "G-0002", Section: "Why it matters"},
	}
	got := RunEntityBodySectionDropped(dropped, map[string]bool{"aaaa111": true})
	if len(got) != 1 {
		t.Fatalf("want 1 finding after the exemption, got %d: %+v", len(got), got)
	}
	if got[0].EntityID != "G-0002" {
		t.Errorf("the wrong record was exempted; surviving finding names %q", got[0].EntityID)
	}
}
