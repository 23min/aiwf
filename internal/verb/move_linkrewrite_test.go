package verb_test

// move_linkrewrite_test.go — M-0314 real-tree integration tests for
// wiring the shared link-destination rewrite primitive into
// `aiwf move`, the one OpMove-emitting verb that rewrote nothing.

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// markdownLink captures the destination of every inline markdown link.
var markdownLink = regexp.MustCompile(`\]\(([^)]+)\)`)

// resolveDestination turns a markdown link destination found in a file
// at linkingPath into a repo-relative path, mirroring the two flavors
// RewriteLinkDestinations documents: a destination rooted at a known
// entity directory is root-relative; anything else resolves against the
// linking file's own directory.
//
// The prefix set matches entityRootPrefixes exactly. A broader set (say
// `work/`) would classify a destination under a non-entity directory as
// root-relative where the primitive resolves it against the linking
// file, and the sweep below would then check a path the primitive never
// produced.
func resolveDestination(dest, linkingPath string) string {
	for _, prefix := range []string{"work/epics/", "work/gaps/", "work/decisions/", "work/contracts/", "docs/adr/"} {
		if strings.HasPrefix(dest, prefix) {
			return path.Clean(dest)
		}
	}
	return path.Clean(path.Join(path.Dir(linkingPath), dest))
}

// moveLinkFixture builds the tree both tests in this file share: two
// epics, a milestone under the first, an unrelated gap, and a gap whose
// body links to the milestone twice — once root-relative, once
// `../`-relative, the two flavors resolveLinkDestination discriminates —
// mentions it bare in prose, and links to the unrelated gap.
//
// The milestone's own body carries a link to its own pre-move path. That
// is what makes move's exclude set load-bearing: without it the shared
// helper sees a body whose link resolves into the move set, and emits a
// competing write for the moved file — serialized from the *tree's*
// entity, which still carries the pre-move `parent:`. A milestone whose
// body links to nothing that moves cannot distinguish the two
// implementations at all.
//
// Returns the milestone's pre-move path, the unrelated gap's path, and
// the linking gap's path.
func moveLinkFixture(t *testing.T, r *runner) (milestonePath, otherPath, linkingPath string) {
	t.Helper()
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Infrastructure", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache layer", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Other gap", testActor, verb.AddOptions{
		BodyOverride: bornCompleteFixtureBody(entity.KindGap),
	}))

	milestone := r.tree().ByID("M-0001")
	other := r.tree().ByID("G-0001")
	if milestone == nil || other == nil {
		t.Fatal("fixture entities missing")
	}
	milestonePath, otherPath = milestone.Path, other.Path

	milestoneFull := filepath.Join(r.root, filepath.FromSlash(milestonePath))
	milestoneRaw, err := os.ReadFile(milestoneFull)
	if err != nil {
		t.Fatal(err)
	}
	selfLinked := string(milestoneRaw) + "\nSpec lives at [this file](" + milestonePath + ").\n"
	if writeErr := os.WriteFile(milestoneFull, []byte(selfLinked), 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}
	commitFixture(t, r.root, "fixture: milestone body linking to its own path")

	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Linking gap", testActor, verb.AddOptions{
		BodyOverride: []byte(
			"## What's missing\n\nSee [the cache milestone](" + milestonePath + ") and a bare mention of M-0001 in prose, " +
				"plus [an untouched gap](" + otherPath + ").\nAlso [the same milestone, relatively](../epics/E-0001-platform/M-0001-cache-layer.md)." +
				"\nAnd [an external reference](https://example.invalid/work/epics/E-0001-platform/M-0001-cache-layer.md)." +
				"\n\n## Why it matters\n\nFixture.\n"),
	}))
	linking := r.tree().ByID("G-0002")
	if linking == nil {
		t.Fatal("G-0002 missing")
	}
	return milestonePath, otherPath, linking.Path
}

// TestMove_RoutesInboundLinkRewriteThroughSharedPrimitive pins
// M-0314/AC-1: `aiwf move` plans its inbound link repair through the
// same primitive its four sibling movers use. The plan must carry a
// rewrite write for the linking gap — absent if the call is removed —
// and exactly one write for the moved milestone itself, since move
// already writes that file to update `parent:` and a second competing
// write would be the double-write the exclude set exists to prevent.
//
// The prose/unrelated-link assertions after Apply pin the primitive's
// established discrimination reaching move too: a reimplementation
// beside it would have to re-derive all of it.
func TestMove_RoutesInboundLinkRewriteThroughSharedPrimitive(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	milestonePath, otherPath, linkingPath := moveLinkFixture(t, r)

	res, err := verb.Move(r.ctx, r.tree(), "M-0001", "E-0002", testActor)
	if err != nil {
		t.Fatal(err)
	}
	if res.Plan == nil {
		t.Fatal("expected plan")
	}

	var moveOps, writeOps []verb.FileOp
	for _, op := range res.Plan.Ops {
		switch op.Type {
		case verb.OpMove:
			moveOps = append(moveOps, op)
		case verb.OpWrite:
			writeOps = append(writeOps, op)
		}
	}
	if len(moveOps) != 1 || moveOps[0].Path != milestonePath {
		t.Fatalf("moveOps = %+v, want exactly one move of %s", moveOps, milestonePath)
	}
	movedPath := moveOps[0].NewPath

	writesAt := map[string][]verb.FileOp{}
	for _, op := range writeOps {
		writesAt[op.Path] = append(writesAt[op.Path], op)
	}
	// Exactly one write for the moved file. The fixture's self-link makes
	// a missing exclude produce a second one, serialized from the tree's
	// pre-move entity — so this count is decided by the exclude, not by a
	// body that could never have matched.
	if len(writesAt[movedPath]) != 1 {
		t.Fatalf("writes at the moved milestone's own path = %d, want exactly 1 (its `parent:` update; a second is the competing double-write the exclude set prevents); writeOps = %+v", len(writesAt[movedPath]), writeOps)
	}
	// The surviving write must be move's own, carrying the post-move
	// parent. The helper's competing write would serialize the tree's
	// entity and still say E-0001.
	if survivor := string(writesAt[movedPath][0].Content); !strings.Contains(survivor, "parent: E-0002") {
		t.Errorf("the write at %s does not carry the post-move parent — move's own write must be the one that lands:\n%s", movedPath, survivor)
	}
	if len(writesAt[linkingPath]) != 1 {
		t.Errorf("writes at the linking gap %s = %d, want exactly 1 — move must plan the inbound link repair through the shared primitive; writeOps = %+v", linkingPath, len(writesAt[linkingPath]), writeOps)
	}
	if len(writesAt[otherPath]) != 0 {
		t.Errorf("gap %s links to nothing that moved and must not be rewritten; writeOps = %+v", otherPath, writeOps)
	}

	if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatal(applyErr)
	}

	body, err := os.ReadFile(filepath.Join(r.root, filepath.FromSlash(linkingPath)))
	if err != nil {
		t.Fatal(err)
	}
	got := string(body)
	if !strings.Contains(got, "("+movedPath+")") {
		t.Errorf("root-relative link to the moved milestone not rewritten to %s:\n%s", movedPath, got)
	}
	// Rewritten, not merely accompanied: an implementation that appended
	// the new destination while leaving the old one would satisfy the
	// assertion above on its own.
	if strings.Contains(got, "("+milestonePath+")") {
		t.Errorf("the pre-move destination %s survives in the body — the link was added beside the old one, not rewritten:\n%s", milestonePath, got)
	}
	// The `../`-relative flavor resolves and re-renders through a
	// different arm than the root-relative one.
	if !strings.Contains(got, "(../epics/E-0002-infrastructure/M-0001-cache-layer.md)") {
		t.Errorf("`../`-relative link to the moved milestone not rewritten in its own flavor:\n%s", got)
	}
	if !strings.Contains(got, "bare mention of M-0001 in prose") {
		t.Errorf("bare-id prose mention of M-0001 must be left untouched:\n%s", got)
	}
	// The URL embeds the milestone's pre-move path, so a primitive that
	// rewrote URL-shaped destinations would visibly alter it.
	if !strings.Contains(got, "(https://example.invalid/work/epics/E-0001-platform/M-0001-cache-layer.md)") {
		t.Errorf("URL-shaped destination must be left untouched even though it embeds the moved path:\n%s", got)
	}
	if !strings.Contains(got, "("+otherPath+")") {
		t.Errorf("link to the non-moved gap must remain unchanged (%s):\n%s", otherPath, got)
	}
}

// TestMove_LeavesNoInboundLinkBroken pins M-0314/AC-2 end to end: after
// a milestone moves between epics, every inbound markdown link in the
// tree still resolves to a file that exists, and the link that named the
// milestone resolves to the milestone itself. The assertions read the
// rewritten bytes off disk and stat the destination — a string match
// against the plan would not distinguish a rewritten link from one
// rewritten to a path nothing occupies.
//
// The moved file is excluded from the sweep because its own outbound
// self-link is not this milestone's subject: ADR-0033 commits to links
// that point *at* a moved entity, and M-0315 is where the moved file's
// own links start resolving.
func TestMove_LeavesNoInboundLinkBroken(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	_, _, linkingPath := moveLinkFixture(t, r)

	r.must(verb.Move(r.ctx, r.tree(), "M-0001", "E-0002", testActor))

	tr := r.tree()
	moved := tr.ByID("M-0001")
	if moved == nil {
		t.Fatal("M-0001 missing after the move")
	}

	var swept int
	for _, e := range tr.Entities {
		if e.Path == moved.Path {
			continue
		}
		body, err := os.ReadFile(filepath.Join(r.root, filepath.FromSlash(e.Path)))
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range markdownLink.FindAllStringSubmatch(string(body), -1) {
			dest := m[1]
			if strings.Contains(dest, "://") {
				continue
			}
			resolved := resolveDestination(dest, e.Path)
			if _, statErr := os.Stat(filepath.Join(r.root, filepath.FromSlash(resolved))); statErr != nil {
				t.Errorf("broken inbound link in %s: destination %q resolves to %s, which does not exist", e.Path, dest, resolved)
			}
			swept++
		}
	}
	// A sweep that examined nothing would pass vacuously.
	if swept < 3 {
		t.Fatalf("swept %d link destinations, want at least the fixture's 3 — the sweep is not reaching the bodies it is meant to check", swept)
	}

	// Resolving is necessary but not sufficient: the link must land on
	// the moved milestone, not merely on some file that happens to exist.
	linkingBody, err := os.ReadFile(filepath.Join(r.root, filepath.FromSlash(linkingPath)))
	if err != nil {
		t.Fatal(err)
	}
	var landedOnMilestone int
	for _, m := range markdownLink.FindAllStringSubmatch(string(linkingBody), -1) {
		if resolveDestination(m[1], linkingPath) == moved.Path {
			landedOnMilestone++
		}
	}
	if landedOnMilestone != 2 {
		t.Errorf("links in %s resolving to the moved milestone at %s = %d, want 2 (the root-relative and the `../`-relative one):\n%s", linkingPath, moved.Path, landedOnMilestone, linkingBody)
	}
}
