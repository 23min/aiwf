package verb_test

// outbound_linkrewrite_test.go — M-0315/AC-3. ADR-0046 extends
// ADR-0033's path-link commitment to the links a moved entity carries in
// its own body: a relative destination resolves against the directory the
// file sits in, so relocating the file changes what its unchanged text
// means.

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestArchive_MovedEntityKeepsItsOwnRelativeLinksResolving pins
// M-0315/AC-3 in the shape the 2026-08-19 observation recorded: a file
// swept into an `archive/` subdirectory keeps every outbound link it
// carries resolving.
//
// The assertions resolve each destination on disk and stat it, rather
// than matching the rewritten text: a destination rewritten to a path
// nothing occupies satisfies a pattern match and fails a stat, and it is
// the resolving that AC-3 asks about.
//
// Root-relative destinations are included as a control. They name a path
// from the repo root, so moving the linking file must NOT change them —
// only the relative forms need recomputation.
func TestArchive_MovedEntityKeepsItsOwnRelativeLinksResolving(t *testing.T) {
	t.Parallel()
	r := newRunner(t)

	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Sibling target", testActor, verb.AddOptions{
		BodyOverride: bornCompleteFixtureBody(entity.KindGap),
	}))
	sibling := r.tree().ByID("G-0001")
	if sibling == nil {
		t.Fatal("G-0001 missing")
	}
	siblingPath := sibling.Path // work/gaps/G-0001-sibling-target.md
	siblingFile := path.Base(siblingPath)

	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Travelling gap", testActor, verb.AddOptions{
		BodyOverride: []byte(
			"## What's missing\n\n" +
				"A bare-filename sibling link: [sibling](" + siblingFile + ").\n" +
				"A dot-slash sibling link: [same sibling](./" + siblingFile + ").\n" +
				"A root-relative link that must not move: [rooted](" + siblingPath + ").\n" +
				"A non-canonical root-relative link: [rooted oddly](work/gaps/./" + siblingFile + ").\n" +
				"An anchor naming no file: [section](#why-it-matters).\n" +
				"A scheme naming nothing in the repo: [mail](mailto:nobody@example.invalid).\n" +
				"A protocol-relative URL: [cdn](//cdn.example.invalid/x.png).\n" +
				"A site-absolute path: [rooted](/README.md).\n" +
				"An angle-bracket destination: [angled](<../gaps/" + siblingFile + ">).\n" +
				"\n## Why it matters\n\nFixture.\n"),
	}))
	traveller := r.tree().ByID("G-0002")
	if traveller == nil {
		t.Fatal("G-0002 missing")
	}

	// Terminal status is what makes the sweep pick it up (ADR-0004).
	r.must(verb.Cancel(r.ctx, r.tree(), "G-0002", testActor, "", false))
	r.must(verb.Archive(r.ctx, r.root, testActor, ""))

	moved := r.tree().ByID("G-0002")
	if moved == nil {
		t.Fatal("G-0002 missing after the sweep")
	}
	if !strings.Contains(filepath.ToSlash(moved.Path), "/archive/") {
		t.Fatalf("G-0002 is at %s, expected it swept under an archive/ subdirectory", moved.Path)
	}

	body, err := os.ReadFile(filepath.Join(r.root, filepath.FromSlash(moved.Path)))
	if err != nil {
		t.Fatal(err)
	}
	got := string(body)

	dests := markdownLink.FindAllStringSubmatch(got, -1)
	if len(dests) != 9 {
		t.Fatalf("found %d link destinations in the moved body, want the fixture's 9:\n%s", len(dests), got)
	}
	var pathDests int
	for _, m := range dests {
		dest := m[1]
		// Only a repo-path destination is resolved here. An anchor names a
		// section of this file; a scheme, a protocol-relative host and a
		// site-absolute path each name something no repo-relative
		// resolution applies to; an angle-bracket destination carries
		// delimiters that are not part of the path.
		if strings.HasPrefix(dest, "#") || strings.Contains(dest, ":") ||
			strings.HasPrefix(dest, "/") || strings.HasPrefix(dest, "<") {
			continue
		}
		resolved := resolveDestination(dest, filepath.ToSlash(moved.Path))
		if _, statErr := os.Stat(filepath.Join(r.root, filepath.FromSlash(resolved))); statErr != nil {
			t.Errorf("outbound link %q in the moved entity resolves to %s, which does not exist", dest, resolved)
		}
		pathDests++
	}
	if pathDests != 4 {
		t.Errorf("resolved %d path destinations, want the fixture's 4", pathDests)
	}
	// A root-relative destination names a path from the repo root, so
	// moving the linking file cannot change what it means — including when
	// its spelling is non-canonical, where re-rendering would silently
	// canonicalize it and count as a content change.
	if !strings.Contains(got, "(work/gaps/./"+siblingFile+")") {
		t.Errorf("non-canonical root-relative destination was canonicalized by the move:\n%s", got)
	}

	// The root-relative form names a path from the repo root, so the move
	// must leave it byte-identical.
	if !strings.Contains(got, "("+siblingPath+")") {
		t.Errorf("root-relative destination %s was rewritten; moving the linking file must not change it:\n%s", siblingPath, got)
	}
	// No non-path destination may be recomputed. Each would be corrupted a
	// different way: resolving `#anchor` against a directory yields the
	// directory; a `mailto:` address, a `//host` and a `/absolute` path
	// would each be mangled into a relative path; and rewriting inside
	// angle brackets strips the closing one.
	for _, untouched := range []string{
		"(#why-it-matters)",
		"(mailto:nobody@example.invalid)",
		"(//cdn.example.invalid/x.png)",
		"(/README.md)",
		"(<../gaps/" + siblingFile + ">)",
	} {
		if !strings.Contains(got, untouched) {
			t.Errorf("destination %s names nothing a move can invalidate and was rewritten anyway:\n%s", untouched, got)
		}
	}
}

// TestMove_MovedMilestoneKeepsItsOwnRelativeLinksResolving covers the
// second geometry: the directory does not deepen, it changes. A
// milestone linking to a sibling that stays behind must end up naming it
// across the epic boundary.
//
// This is the seam M-0314 deliberately left open — it repaired only the
// links pointing AT a moved milestone, and asserted nothing about the
// ones the milestone carries.
func TestMove_MovedMilestoneKeepsItsOwnRelativeLinksResolving(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Infrastructure", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Staying put", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Travelling", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))

	stayer := r.tree().ByID("M-0001")
	traveller := r.tree().ByID("M-0002")
	if stayer == nil || traveller == nil {
		t.Fatal("fixture milestones missing")
	}
	travellerFull := filepath.Join(r.root, filepath.FromSlash(traveller.Path))
	raw, err := os.ReadFile(travellerFull)
	if err != nil {
		t.Fatal(err)
	}
	updated := string(raw) + "\nDepends on [the sibling](" + path.Base(stayer.Path) + ").\n"
	if writeErr := os.WriteFile(travellerFull, []byte(updated), 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}
	commitFixture(t, r.root, "fixture: milestone linking to a sibling by bare filename")

	r.must(verb.Move(r.ctx, r.tree(), "M-0002", "E-0002", testActor))

	moved := r.tree().ByID("M-0002")
	if moved == nil {
		t.Fatal("M-0002 missing after the move")
	}
	body, err := os.ReadFile(filepath.Join(r.root, filepath.FromSlash(moved.Path)))
	if err != nil {
		t.Fatal(err)
	}
	assertOutboundLinksResolve(t, r.root, string(body), filepath.ToSlash(moved.Path), 1)
}

// TestRetitle_DirShapedKindLeavesRelativeLinksByteIdentical covers the
// third geometry, where the right answer is to change nothing: a
// dir-shaped retitle renames a directory in place, same parent and same
// depth. A relative destination never names its own directory, so no
// relative spelling can change — `../../gaps/G-NNNN-<slug>.md` resolves
// identically from either name.
//
// What this pins is therefore the absence of churn, not a repair. The
// outbound extension recomputes a destination only when the geometry
// makes the old text mean something new; a recompute that re-rendered
// every destination would canonicalize spellings here and rewrite bytes
// for no reason, widening the commit and the merge surface.
func TestRetitle_DirShapedKindLeavesRelativeLinksByteIdentical(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Outside target", testActor, verb.AddOptions{
		BodyOverride: bornCompleteFixtureBody(entity.KindGap),
	}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Original name", testActor, verb.AddOptions{}))

	gap := r.tree().ByID("G-0001")
	epic := r.tree().ByID("E-0001")
	if gap == nil || epic == nil {
		t.Fatal("fixture entities missing")
	}
	epicFull := filepath.Join(r.root, filepath.FromSlash(epic.Path))
	raw, err := os.ReadFile(epicFull)
	if err != nil {
		t.Fatal(err)
	}
	// A `../`-relative destination out of the epic's own directory.
	rel := "../../gaps/" + path.Base(gap.Path)
	updated := string(raw) + "\nSee [the outside gap](" + rel + ").\n"
	if writeErr := os.WriteFile(epicFull, []byte(updated), 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}
	commitFixture(t, r.root, "fixture: epic body linking out of its own directory")

	r.must(verb.Retitle(r.ctx, r.tree(), "E-0001", "Renamed epic", testActor, "", 0))

	renamed := r.tree().ByID("E-0001")
	if renamed == nil {
		t.Fatal("E-0001 missing after the retitle")
	}
	body, err := os.ReadFile(filepath.Join(r.root, filepath.FromSlash(renamed.Path)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "("+rel+")") {
		t.Errorf("relative destination %q was rewritten by a same-depth directory rename, which cannot change what it names:\n%s", rel, body)
	}
	// It resolves as well as being unchanged — an unchanged destination
	// that no longer resolves would mean the geometry argument is wrong.
	assertOutboundLinksResolve(t, r.root, string(body), filepath.ToSlash(renamed.Path), 1)
}

// assertOutboundLinksResolve stats every non-URL markdown destination in
// body, resolved from linkingPath, and requires at least wantAtLeast of
// them — a sweep that examined nothing would otherwise pass vacuously.
func assertOutboundLinksResolve(t *testing.T, root, body, linkingPath string, wantAtLeast int) {
	t.Helper()
	var checked int
	for _, m := range markdownLink.FindAllStringSubmatch(body, -1) {
		dest := m[1]
		if strings.Contains(dest, "://") {
			continue
		}
		resolved := resolveDestination(dest, linkingPath)
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(resolved))); err != nil {
			t.Errorf("outbound link %q in %s resolves to %s, which does not exist", dest, linkingPath, resolved)
		}
		checked++
	}
	if checked < wantAtLeast {
		t.Fatalf("checked %d destinations in %s, want at least %d", checked, linkingPath, wantAtLeast)
	}
}

// TestMove_LeavesNonMovingEntitiesUnwritten pins the bound on the
// outbound extension's blast radius: an entity that is not moving is not
// rewritten, even when its body carries a non-canonical relative
// destination that a re-render would tidy.
//
// The outbound recompute resolves a destination against the body's old
// directory and renders it against the new one. For a body that is not
// moving those are the same directory, so re-rendering would return the
// destination's canonical spelling — which differs from `./x.md` by the
// two characters that make it non-canonical. Emitting that write would
// pull every such entity into an unrelated verb's commit.
func TestMove_LeavesNonMovingEntitiesUnwritten(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Infrastructure", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Travelling", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Target", testActor, verb.AddOptions{
		BodyOverride: bornCompleteFixtureBody(entity.KindGap),
	}))
	target := r.tree().ByID("G-0001")
	if target == nil {
		t.Fatal("G-0001 missing")
	}
	// `./` makes it non-canonical: a re-render would drop those two
	// characters and count as a content change.
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Bystander", testActor, verb.AddOptions{
		BodyOverride: []byte("## What's missing\n\nSee [the target](./" + path.Base(target.Path) +
			").\n\n## Why it matters\n\nFixture.\n"),
	}))
	bystander := r.tree().ByID("G-0002")
	if bystander == nil {
		t.Fatal("G-0002 missing")
	}

	res, err := verb.Move(r.ctx, r.tree(), "M-0001", "E-0002", testActor)
	if err != nil {
		t.Fatal(err)
	}
	if res.Plan == nil {
		t.Fatal("expected plan")
	}
	for _, op := range res.Plan.Ops {
		if op.Type == verb.OpWrite && op.Path == bystander.Path {
			t.Errorf("move planned a write for %s, which neither moved nor links at anything that moved — the outbound recompute must not re-render a body whose directory is unchanged", bystander.Path)
		}
	}
}
