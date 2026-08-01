package verb_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/verb"
)

// canonicalWidthViolations reports every on-disk deviation from
// canonical id width, checking both surfaces an id occupies: the
// frontmatter `id:` the loader reads back, and the path segment that
// names the file or directory. The two can disagree — a verb that
// canonicalizes one and not the other passes a single-surface check —
// so both are asserted independently.
//
// It reads the width the loader actually returns rather than a
// re-derived value: `tree.Load` preserves an entity's stored width, so
// a narrow id on disk surfaces here as a narrow `e.ID`.
func canonicalWidthViolations(t *testing.T, tr *tree.Tree) []string {
	t.Helper()
	var out []string
	for _, e := range tr.Entities {
		if canon := entity.Canonicalize(e.ID); canon != e.ID {
			out = append(out, fmt.Sprintf("frontmatter id %q (%s) is narrower than canonical %q", e.ID, e.Path, canon))
		}
		pathID, ok := entity.IDFromPath(e.Path, e.Kind)
		if !ok {
			out = append(out, fmt.Sprintf("path %q yields no id for kind %s", e.Path, e.Kind))
			continue
		}
		if canon := entity.Canonicalize(pathID); canon != pathID {
			out = append(out, fmt.Sprintf("path id %q (%s) is narrower than canonical %q", pathID, e.Path, canon))
		}
	}
	return out
}

// nestingViolations reports every milestone whose file does not sit in
// its parent epic's directory.
//
// Width and containment are independent failures, and a width-only
// helper is structurally blind to the second: a milestone filed under a
// canonically-named directory that holds no epic still yields a
// canonical path id. The tree loader resolves such a milestone by
// frontmatter and `aiwf check` reports nothing, so an assertion that
// walks entities one at a time cannot see the split — only one that
// compares a child's directory against its parent's.
func nestingViolations(t *testing.T, tr *tree.Tree) []string {
	t.Helper()
	var out []string
	for _, e := range tr.Entities {
		if e.Kind != entity.KindMilestone {
			continue
		}
		parent := tr.ByID(e.Parent)
		if parent == nil {
			out = append(out, fmt.Sprintf("milestone %s: parent %q resolves to no entity", e.ID, e.Parent))
			continue
		}
		gotDir, wantDir := filepath.Dir(e.Path), filepath.Dir(parent.Path)
		if gotDir != wantDir {
			out = append(out, fmt.Sprintf("milestone %s is in %q, but its parent %s owns %q", e.ID, gotDir, parent.ID, wantDir))
		}
	}
	return out
}

// TestCanonicalWidthViolations_DetectsNarrowOnDisk is the vacuity guard
// for the invariant below. The assertion is only worth running if it can
// fail, and the pre-existing import tests show how easy it is to write a
// width check that cannot: they declare narrow ids and assert through
// `Tree.ByID`, which canonicalizes on lookup, so they pass whatever width
// reaches disk.
//
// A hand-written narrow epic — bypassing every verb — must be reported on
// both surfaces.
func TestCanonicalWidthViolations_DetectsNarrowOnDisk(t *testing.T) {
	t.Parallel()
	r := newRunner(t)

	dir := filepath.Join(r.root, "work", "epics", "E-11-legacy-narrow")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := "---\nid: E-11\ntitle: Legacy narrow\nstatus: proposed\n---\n\n## Goal\n\nPredates the width migration.\n"
	if err := os.WriteFile(filepath.Join(dir, "epic.md"), []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Assert which two branches fired, not how many. A count alone does
	// not discriminate: a fixture whose directory carries no id at all
	// also yields two, via the frontmatter branch plus the no-id-in-path
	// branch — a different defect reported by the same total.
	got := canonicalWidthViolations(t, r.tree())
	var sawFrontmatter, sawPath bool
	for _, v := range got {
		switch {
		case strings.HasPrefix(v, "frontmatter id "):
			sawFrontmatter = true
		case strings.HasPrefix(v, "path id "):
			sawPath = true
		}
	}
	if !sawFrontmatter || !sawPath {
		t.Fatalf("want both a frontmatter-width and a path-width violation; got %+v", got)
	}
}

// TestIDCreatingVerbs_EmitCanonicalWidth pins the output property behind
// ADR-0008's canonical emission: whatever width an input happens to use,
// every route that puts an id on disk stores the canonical form.
//
// This is an output-property assertion rather than a source-shape one on
// purpose. There is no structural signature for "this string literal is
// an entity id", so a static rule over the verb layer would both miss
// verbatim writes and fire on innocent formatting; driving each route and
// measuring what lands on disk is what actually constrains the behavior.
//
// Width alone is not the whole property: an id also names a directory
// that has to contain the right entities, so each route is measured for
// containment as well.
func TestIDCreatingVerbs_EmitCanonicalWidth(t *testing.T) {
	t.Parallel()

	t.Run("add allocates canonical for every kind", func(t *testing.T) {
		t.Parallel()
		r := newRunner(t)
		// Drive every kind the kernel knows rather than a hand-listed
		// subset, so a kind added later cannot be skipped silently.
		// Milestone is the one whose path nests inside another entity's
		// directory, which makes it the kind this sweep most needs.
		r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
		epic := r.tree().Entities[0]
		for _, k := range entity.AllKinds() {
			if k == entity.KindEpic {
				continue
			}
			opts := verb.AddOptions{BodyOverride: bornCompleteFixtureBody(k)}
			if k == entity.KindMilestone {
				opts.EpicID, opts.TDD = epic.ID, "none"
			}
			r.must(verb.Add(r.ctx, r.tree(), k, "Something for "+string(k), testActor, opts))
		}
		tr := r.tree()
		if got := canonicalWidthViolations(t, tr); len(got) != 0 {
			t.Errorf("add emitted non-canonical ids: %+v", got)
		}
		if got := nestingViolations(t, tr); len(got) != 0 {
			t.Errorf("add misplaced a milestone: %+v", got)
		}
	})

	t.Run("import stores canonical for a narrow explicit id", func(t *testing.T) {
		t.Parallel()
		r := newRunner(t)
		// Every width here is legal per the kind's grammar (`E-\d{2,}`,
		// `M-\d{3,}`) and narrower than CanonicalPad. Widths below the
		// grammar minimum never reach this code — the manifest parser
		// rejects them via entity.ValidateID.
		src := `version: 1
entities:
  - kind: epic
    id: E-11
    frontmatter: {title: "Narrow epic", status: active}
  - kind: milestone
    id: M-001
    frontmatter: {title: "Narrow milestone", status: draft, parent: E-11}
`
		m := loadManifest(t, src)
		res, err := verb.Import(r.ctx, r.tree(), m, testActor, verb.ImportOptions{})
		if err != nil {
			t.Fatalf("Import: %v", err)
		}
		if check.HasErrors(res.Findings) {
			t.Fatalf("findings: %+v", res.Findings)
		}
		applyImport(t, r, res.Plans)

		tr := r.tree()
		if got := canonicalWidthViolations(t, tr); len(got) != 0 {
			t.Errorf("import stored non-canonical ids: %+v", got)
		}
		// The child's `parent:` is consumed as a path component, not only
		// as a lookup key, so the epic's resolved id has to drive both its
		// own directory and its milestone's. Containment is asserted
		// directly: a milestone filed beside its epic rather than inside it
		// still carries a canonical path id, so the width helper above
		// cannot see the split.
		if got := nestingViolations(t, tr); len(got) != 0 {
			t.Errorf("import split a milestone from its epic: %+v", got)
		}
		// A child's `parent:` names the id its epic actually carries, so
		// the guards that walk an epic's children — which compare the
		// field literally — still see it. Asserted through the guard
		// itself rather than through a canonicalizing lookup, because a
		// lookup would resolve either spelling and prove nothing.
		if _, err := verb.Cancel(r.ctx, r.tree(), "E-0011", testActor, "measuring the child guard", false); err == nil {
			t.Error("cancelling an epic that owns a draft milestone was allowed; the child guard did not see the milestone")
		}
		// Scope: the remaining reference fields — `depends_on`,
		// `superseded_by`, `discovered_in`, `addressed_by` — still carry
		// whatever spelling the manifest declared, since the frontmatter
		// map is copied with only `id` and `parent` resolved. Tracked as
		// G-0505, which also owns the decision to exclude `prior_ids`,
		// whose narrow entries are the record rather than a defect.
		if tr.ByID("M-0001") == nil {
			t.Fatalf("milestone M-0001 absent; tree has %+v", idsOf(tr))
		}
	})

	t.Run("import auto-allocates canonical alongside a narrow reservation", func(t *testing.T) {
		t.Parallel()
		r := newRunner(t)
		src := `version: 1
entities:
  - kind: epic
    id: E-11
    frontmatter: {title: "Reserved narrow", status: active}
  - kind: epic
    id: auto
    frontmatter: {title: "Auto", status: proposed}
`
		m := loadManifest(t, src)
		res, err := verb.Import(r.ctx, r.tree(), m, testActor, verb.ImportOptions{})
		if err != nil {
			t.Fatalf("Import: %v", err)
		}
		if check.HasErrors(res.Findings) {
			t.Fatalf("findings: %+v", res.Findings)
		}
		applyImport(t, r, res.Plans)

		if got := canonicalWidthViolations(t, r.tree()); len(got) != 0 {
			t.Errorf("import auto path stored non-canonical ids: %+v", got)
		}
	})

	t.Run("parent epic resolution refuses every unusable parent", func(t *testing.T) {
		t.Parallel()
		// The four refusal arms of the parent lookup. Each is reachable
		// from an ordinary manifest, so each is driven rather than
		// annotated; together they cover both sides of the
		// resident-versus-manifest-declared split.
		milestone := func(parent string) string {
			return "version: 1\nentities:\n" +
				"  - kind: milestone\n    id: M-001\n" +
				"    frontmatter: {title: \"Child\", status: draft, parent: " + parent + ", tdd: none}\n"
		}
		epicWithTitle := func(title string) string {
			return "  - kind: epic\n    id: E-0011\n    frontmatter: {title: \"" + title + "\", status: active}\n"
		}
		tests := []struct {
			name     string
			resident bool // seed a non-epic entity in the tree first
			src      string
			opts     verb.ImportOptions
			want     string
		}{
			{
				name:     "resident parent is not an epic",
				resident: true,
				src:      milestone("G-0001"),
				want:     "is not an epic",
			},
			{
				name: "parent resolves to nothing",
				src:  milestone("E-0099"),
				want: "does not exist in tree or manifest",
			},
			{
				name: "manifest-declared parent is not an epic",
				src: "version: 1\nentities:\n" +
					"  - kind: gap\n    id: G-0007\n    frontmatter: {title: \"Not an epic\", status: open}\n" +
					"    body: \"## What's missing\\n\\nA thing.\\n\\n## Why it matters\\n\\nIt bites.\\n\"\n" +
					"  - kind: milestone\n    id: M-001\n" +
					"    frontmatter: {title: \"Child\", status: draft, parent: G-0007, tdd: none}\n",
				want: "is not an epic",
			},
			// The last two reach the parent lookup only as forward
			// references. Declared epic-first, the epic's own build
			// validates its title and fails before any child consults it;
			// declared child-first, the lookup is where the bad title
			// surfaces.
			{
				name: "forward-referenced parent epic title exceeds the cap",
				src:  milestone("E-0011") + epicWithTitle("This parent title is comfortably past the cap set below"),
				opts: verb.ImportOptions{TitleMaxLength: 10},
				want: "parent epic",
			},
			{
				name: "forward-referenced parent epic title slugifies to empty",
				src:  milestone("E-0011") + epicWithTitle("!!!"),
				want: "cannot derive directory",
			},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				r := newRunner(t)
				if tc.resident {
					r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Not an epic", testActor, verb.AddOptions{
						BodyOverride: bornCompleteFixtureBody(entity.KindGap),
					}))
				}
				_, err := verb.Import(r.ctx, r.tree(), loadManifest(t, tc.src), testActor, tc.opts)
				if err == nil {
					t.Fatalf("Import succeeded; want error containing %q", tc.want)
				}
				if !strings.Contains(err.Error(), tc.want) {
					t.Errorf("error = %q, want it to contain %q", err, tc.want)
				}
			})
		}
	})

	t.Run("reallocate emits canonical", func(t *testing.T) {
		t.Parallel()
		r := newRunner(t)
		r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Needs a new number", testActor, verb.AddOptions{
			BodyOverride: bornCompleteFixtureBody(entity.KindGap),
		}))
		tr := r.tree()
		if len(tr.Entities) != 1 {
			t.Fatalf("setup: want 1 entity, got %d", len(tr.Entities))
		}
		r.must(verb.Reallocate(r.ctx, r.tree(), tr.Entities[0].ID, testActor))
		if got := canonicalWidthViolations(t, r.tree()); len(got) != 0 {
			t.Errorf("reallocate emitted non-canonical ids: %+v", got)
		}
	})
}
