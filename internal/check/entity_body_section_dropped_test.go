package check

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// entity_body_section_dropped_test.go — M-0331. WalkDroppedBodySections judges
// each entity by id, at the start of the pushed range against HEAD, and credits
// every required section missing at HEAD that the entity carried at its starting
// point to the commit that left it out.
//
// The walker's answers turn on history shape — merges, renames, creates,
// reallocations — so every test drives real commits through git. One test per
// rule the walker applies.

const (
	gapPath = "work/gaps/G-0001-fixture.md"
	// gapRenamed sorts before gapPath, so git lists a rename's added path ahead
	// of its deleted one and a delete line is the last word on the entity.
	gapRenamed   = "work/gaps/G-0001-a-renamed.md"
	whatsMissing = "What's missing"
	whyItMatters = "Why it matters"
)

// wrapTrailers is the trailer set the wrap-milestone ritual stamps on the plain
// `git commit` that writes a spec without passing a body-supplying verb.
var wrapTrailers = []string{"aiwf-verb: wrap-milestone", "aiwf-entity: G-0001", "aiwf-actor: human/test"}

// gapFile renders a gap carrying the named top-level sections, each with prose,
// under frontmatter for id plus any extra frontmatter lines.
func gapFile(id, extraFrontmatter string, sections ...string) string {
	var b strings.Builder
	b.WriteString("---\nid: " + id + "\ntitle: Fixture\nstatus: open\n" + extraFrontmatter + "---\n")
	for _, s := range sections {
		b.WriteString("## " + s + "\n\nProse.\n\n")
	}
	return b.String()
}

func (f *walkerFixture) head() string {
	f.t.Helper()
	return strings.TrimSpace(f.run("git", "rev-parse", "HEAD"))
}

// put writes content at rel and commits it with msg and trailers.
func (f *walkerFixture) put(rel, content, msg string, trailers ...string) string {
	f.t.Helper()
	f.writeFile(rel, content)
	return f.commit(msg, trailers...)
}

func walkFrom(t *testing.T, f *walkerFixture, base string) []DroppedBodySection {
	t.Helper()
	return WalkDroppedBodySections(context.Background(), f.root, base, "")
}

// walkWithTrunk runs the walker with a trunk ref whose omissions are exempt.
func walkWithTrunk(t *testing.T, f *walkerFixture, base, trunk string) []DroppedBodySection {
	t.Helper()
	return WalkDroppedBodySections(context.Background(), f.root, base, trunk)
}

func assertDropped(t *testing.T, want, got []DroppedBodySection) {
	t.Helper()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("dropped sections (-want +got):\n%s", diff)
	}
}

func TestWalkDroppedBodySections(t *testing.T) {
	t.Parallel()

	full := gapFile("G-0001", "", whatsMissing, whyItMatters)
	partial := gapFile("G-0001", "", whatsMissing)

	// The subject of the rule: a section the entity carried when the push
	// started, missing at HEAD, credited to the commit that removed it.
	t.Run("reports a section a commit removed", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		f.put(gapPath, full, "seed")
		base := f.head()
		drop := f.put(gapPath, partial, "wrap", wrapTrailers...)
		assertDropped(t, []DroppedBodySection{{SHA: drop, Path: gapPath, EntityID: "G-0001", Section: whyItMatters}}, walkFrom(t, f, base))
	})

	// An author is never held to an omission the push did not introduce.
	t.Run("does not report an omission present when the push started", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		f.put(gapPath, partial, "seed")
		base := f.head()
		f.put(gapPath, strings.Replace(partial, "Prose.", "Edited prose.", 1), "edit", wrapTrailers...)
		assertDropped(t, nil, walkFrom(t, f, base))
	})

	// The wrap ritual edits a spec on its milestone branch and merges it with
	// --no-ff; the drop is still published by the push that carries the merge.
	t.Run("reports a drop made on a branch merged with --no-ff", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		f.put(gapPath, full, "seed")
		base := f.head()
		f.run("git", "checkout", "-q", "-b", "side")
		drop := f.put(gapPath, partial, "wrap", wrapTrailers...)
		f.run("git", "checkout", "-q", "main")
		f.run("git", "merge", "-q", "--no-ff", "--no-edit", "side")
		assertDropped(t, []DroppedBodySection{{SHA: drop, Path: gapPath, EntityID: "G-0001", Section: whyItMatters}}, walkFrom(t, f, base))
	})

	// A merge that writes content of its own — a conflict resolution, or an
	// edit folded into the merge — is the commit that removed the section.
	t.Run("credits a drop written by a merge to that merge", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		f.put(gapPath, full, "seed")
		base := f.head()
		f.run("git", "checkout", "-q", "-b", "side")
		f.put("work/gaps/G-0002-other.md", gapFile("G-0002", "", whatsMissing, whyItMatters), "unrelated")
		f.run("git", "checkout", "-q", "main")
		f.run("git", "merge", "-q", "--no-ff", "--no-commit", "side")
		merge := f.put(gapPath, partial, "merge side and drop a section")
		assertDropped(t, []DroppedBodySection{{SHA: merge, Path: gapPath, EntityID: "G-0001", Section: whyItMatters}}, walkFrom(t, f, base))
	})

	// Identity is the id, not the path: a later retitle or archive in the same
	// push moves the file and does not carry the drop away with it.
	t.Run("reports a drop that a later commit renamed", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		f.put(gapPath, full, "seed")
		base := f.head()
		drop := f.put(gapPath, partial, "drop")
		f.run("git", "mv", gapPath, gapRenamed)
		f.commit("retitle")
		assertDropped(t, []DroppedBodySection{{SHA: drop, Path: gapRenamed, EntityID: "G-0001", Section: whyItMatters}}, walkFrom(t, f, base))
	})

	t.Run("reports a drop made in the commit that renamed the file", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		f.put(gapPath, full, "seed")
		base := f.head()
		f.run("git", "mv", gapPath, gapRenamed)
		f.writeFile(gapRenamed, partial)
		moved := f.commit("move and drop")
		assertDropped(t, []DroppedBodySection{{SHA: moved, Path: gapRenamed, EntityID: "G-0001", Section: whyItMatters}}, walkFrom(t, f, base))
	})

	// Only the two ends of the push count, so a section that came and went
	// inside it leaves the entity no worse than it started.
	t.Run("does not report a section added and removed within the push", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		f.put(gapPath, partial, "seed")
		base := f.head()
		f.put(gapPath, full, "add it")
		f.put(gapPath, partial, "take it out again")
		assertDropped(t, nil, walkFrom(t, f, base))
	})

	// A section dropped, restored and dropped again is one regression, owed by
	// the commit whose removal HEAD still carries.
	t.Run("credits a repeated drop once, to the last removal", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		f.put(gapPath, full, "seed")
		base := f.head()
		f.put(gapPath, partial, "drop")
		f.put(gapPath, full, "restore")
		last := f.put(gapPath, partial, "drop again")
		assertDropped(t, []DroppedBodySection{{SHA: last, Path: gapPath, EntityID: "G-0001", Section: whyItMatters}}, walkFrom(t, f, base))
	})

	// A merge can adopt one parent's copy of a file wholesale — `-s ours`, or a
	// resolution taking a stale version — and git lists no write for it. The
	// entity still differs between the two ends, and the merge that published
	// the older copy is the commit that left the section out.
	t.Run("credits a merge that adopted an older copy lacking the section", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		f.put(gapPath, partial, "an old version, before the section existed")
		f.run("git", "branch", "stale")
		f.put(gapPath, full, "add the section")
		base := f.head()
		f.run("git", "checkout", "-q", "stale")
		f.put("work/gaps/G-0002-other.md", gapFile("G-0002", "", whatsMissing, whyItMatters), "unrelated work")
		f.run("git", "merge", "-q", "-s", "ours", "--no-edit", "main")
		adopted := f.head()
		f.put("work/gaps/G-0003-later.md", gapFile("G-0003", "", whatsMissing, whyItMatters), "later work, so the merge is not HEAD")
		assertDropped(t, []DroppedBodySection{{SHA: adopted, Path: gapPath, EntityID: "G-0001", Section: whyItMatters}}, walkFrom(t, f, base))
	})

	// A clean merge of edits from both sides writes the file too, but the
	// section went on the side that removed it, not at the merge.
	t.Run("credits the removal, not a clean merge that combined both sides", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		f.put(gapPath, gapFile("G-0001", "", whatsMissing, "Notes", whyItMatters), "seed")
		base := f.head()
		f.run("git", "checkout", "-q", "-b", "side")
		f.put(gapPath, strings.Replace(gapFile("G-0001", "", whatsMissing, "Notes", whyItMatters), "## What's missing\n\nProse.", "## What's missing\n\nSharper prose.", 1), "edit the other end")
		f.run("git", "checkout", "-q", "main")
		drop := f.put(gapPath, gapFile("G-0001", "", whatsMissing, "Notes"), "drop the section")
		f.run("git", "merge", "-q", "--no-ff", "--no-edit", "side")
		assertDropped(t, []DroppedBodySection{{SHA: drop, Path: gapPath, EntityID: "G-0001", Section: whyItMatters}}, walkFrom(t, f, base))
	})

	// A reallocation inside the push keeps the entity's starting point: a forced
	// create stays exempt under its new id.
	t.Run("keeps a forced create's starting point through a reallocation in the push", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		base := f.head()
		const created, reallocated = "work/gaps/G-0002-fixture.md", "work/gaps/G-0003-fixture.md"
		f.put(created, gapFile("G-0002", "", whatsMissing), "aiwf add gap G-0002",
			"aiwf-verb: add", "aiwf-entity: G-0002", "aiwf-actor: human/test", "aiwf-force: needed now")
		f.run("git", "mv", created, reallocated)
		f.writeFile(reallocated, gapFile("G-0003", "prior_ids:\n    - G-0002\n", whatsMissing))
		f.commit("aiwf reallocate G-0002 -> G-0003", "aiwf-verb: reallocate", "aiwf-entity: G-0003", "aiwf-actor: human/test")
		assertDropped(t, nil, walkFrom(t, f, base))
	})

	// A drop followed by a reallocation is owed by the drop, not by the commit
	// that renumbered the entity.
	t.Run("credits a drop, not a later reallocation that renumbered the entity", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		f.put(gapPath, full, "seed")
		base := f.head()
		drop := f.put(gapPath, partial, "drop", wrapTrailers...)
		const reallocated = "work/gaps/G-0002-fixture.md"
		f.run("git", "mv", gapPath, reallocated)
		f.writeFile(reallocated, gapFile("G-0002", "prior_ids:\n    - G-0001\n", whatsMissing))
		f.commit("aiwf reallocate G-0001 -> G-0002", "aiwf-verb: reallocate", "aiwf-entity: G-0002", "aiwf-actor: human/test")
		assertDropped(t, []DroppedBodySection{{SHA: drop, Path: reallocated, EntityID: "G-0002", Section: whyItMatters}}, walkFrom(t, f, base))
	})

	// A section trunk already lacks is not the pushing author's debt, even when
	// merging trunk is what brings the removal into their branch.
	t.Run("does not report a section trunk already lacks", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		f.put(gapPath, full, "seed")
		f.run("git", "branch", "feature")
		f.put(gapPath, partial, "someone else drops it on trunk")
		f.run("git", "checkout", "-q", "feature")
		base := f.head()
		f.run("git", "merge", "-q", "--no-ff", "--no-edit", "main")
		assertDropped(t, nil, walkWithTrunk(t, f, base, "main"))
	})

	// A create that passed no body-supplying verb starts from nothing, so every
	// required section it leaves out is reported.
	t.Run("reports a section missing from an entity created without a verb", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		base := f.head()
		created := f.put(gapPath, partial, "hand-written gap", wrapTrailers...)
		f.put("work/gaps/G-0002-later.md", gapFile("G-0002", "", whatsMissing, whyItMatters), "later work, so the create is not HEAD")
		assertDropped(t, []DroppedBodySection{{SHA: created, Path: gapPath, EntityID: "G-0001", Section: whyItMatters}}, walkFrom(t, f, base))
	})

	// A create made by `aiwf add` starts from the body that commit wrote, so a
	// section removed by hand later in the same push is still a regression.
	t.Run("reports a section removed after an aiwf add create in the same push", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		base := f.head()
		f.put(gapPath, full, "aiwf add gap G-0001", "aiwf-verb: add", "aiwf-entity: G-0001", "aiwf-actor: human/test")
		drop := f.put(gapPath, partial, "trim by hand")
		assertDropped(t, []DroppedBodySection{{SHA: drop, Path: gapPath, EntityID: "G-0001", Section: whyItMatters}}, walkFrom(t, f, base))
	})

	// An unforced `aiwf add` refuses a body missing a section, so an incomplete
	// create that carries its trailer without `aiwf-force` passed no verb.
	t.Run("reports an incomplete create whose add trailer is not forced", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		base := f.head()
		created := f.put(gapPath, partial, "hand-written, stamped as add", "aiwf-verb: add", "aiwf-entity: G-0001", "aiwf-actor: human/test")
		assertDropped(t, []DroppedBodySection{{SHA: created, Path: gapPath, EntityID: "G-0001", Section: whyItMatters}}, walkFrom(t, f, base))
	})

	// A forced `aiwf add` wrote an incomplete body on purpose, and that body is
	// its starting point — the sovereign override stays in force afterwards.
	t.Run("does not report what a forced aiwf add create left out", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		base := f.head()
		f.put(gapPath, partial, "aiwf add gap G-0001",
			"aiwf-verb: add", "aiwf-entity: G-0001", "aiwf-actor: human/test", "aiwf-force: needed now")
		assertDropped(t, nil, walkFrom(t, f, base))
	})

	// `aiwf import` is excluded from the write seams rather than gated, and a
	// body it wrote is likewise its own starting point.
	t.Run("does not report what an aiwf import create left out", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		base := f.head()
		f.put(gapPath, partial, "aiwf import", "aiwf-verb: import", "aiwf-entity: G-0001", "aiwf-actor: human/test")
		assertDropped(t, nil, walkFrom(t, f, base))
	})

	// `aiwf reallocate` gives an entity a new id and records the old one in
	// prior_ids; it is the same entity, so an omission it already carried is
	// not a create's.
	t.Run("matches a reallocated entity to its prior id", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		f.put(gapPath, partial, "seed")
		base := f.head()
		const reallocated = "work/gaps/G-0002-fixture.md"
		f.run("git", "mv", gapPath, reallocated)
		f.writeFile(reallocated, gapFile("G-0002", "prior_ids:\n    - G-0001\n", whatsMissing))
		f.commit("aiwf reallocate G-0001 -> G-0002", "aiwf-verb: reallocate", "aiwf-entity: G-0002", "aiwf-actor: human/test")
		assertDropped(t, nil, walkFrom(t, f, base))
	})

	// The push is judged from where the branch left its base, so an entity the
	// base's line deleted afterwards is compared against the body the branch
	// forked from, and an omission it already had is not the pusher's.
	t.Run("does not report an entity the base's line deleted after the fork", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		f.put(gapPath, partial, "an entity already missing a section")
		f.run("git", "branch", "keep")
		f.run("git", "rm", "-q", gapPath)
		f.commit("delete it on the base's line")
		base := f.head()
		f.run("git", "checkout", "-q", "keep")
		f.put("work/gaps/G-0002-other.md", gapFile("G-0002", "", whatsMissing, whyItMatters), "unrelated work")
		assertDropped(t, nil, walkFrom(t, f, base))
	})

	// A branch tracking a moving upstream is judged from where it forked, not from
	// wherever the upstream has since reached: a section filled in there afterwards
	// was never this branch's to lose.
	t.Run("does not report a section the upstream gained after the fork", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		f.put(gapPath, partial, "an entity missing a section")
		f.run("git", "branch", "feature")
		upstream := f.put(gapPath, full, "someone fills the section in on the upstream")
		f.run("git", "checkout", "-q", "feature")
		f.put("work/gaps/G-0002-other.md", gapFile("G-0002", "", whatsMissing, whyItMatters), "unrelated work")
		assertDropped(t, nil, walkFrom(t, f, upstream))
	})

	// Two histories with no commit in common give the push no starting point.
	t.Run("returns nothing when the base shares no history with HEAD", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		f.put(gapPath, full, "seed")
		base := f.head()
		f.run("git", "checkout", "-q", "--orphan", "unrelated")
		f.run("git", "rm", "-rqf", ".")
		f.put(gapPath, partial, "an unrelated history")
		assertDropped(t, nil, walkFrom(t, f, base))
	})

	// Trunk exempts only what the entity itself lacks there. An unrelated entity
	// that kept the id through a collision is not this one.
	t.Run("does not borrow omissions from another entity holding its prior id on trunk", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		base := f.head()
		f.run("git", "branch", "trunk")
		f.put(gapPath, full, "the branch adds its own G-0001", "aiwf-verb: add", "aiwf-entity: G-0001", "aiwf-actor: human/test")
		f.run("git", "checkout", "-q", "trunk")
		f.put("work/gaps/G-0001-other.md", gapFile("G-0001", "", whatsMissing),
			"trunk force-adds an unrelated G-0001", "aiwf-verb: add", "aiwf-entity: G-0001", "aiwf-actor: human/test", "aiwf-force: x")
		f.run("git", "checkout", "-q", "main")
		f.run("git", "merge", "-q", "--no-ff", "--no-edit", "trunk")
		const reallocated = "work/gaps/G-0002-fixture.md"
		f.run("git", "mv", gapPath, reallocated)
		f.writeFile(reallocated, gapFile("G-0002", "prior_ids:\n    - G-0001\n", whatsMissing, whyItMatters))
		f.commit("aiwf reallocate G-0001 -> G-0002", "aiwf-verb: reallocate", "aiwf-entity: G-0002", "aiwf-actor: human/test")
		f.put(reallocated, gapFile("G-0002", "prior_ids:\n    - G-0001\n", whatsMissing), "hand drop")
		got := walkWithTrunk(t, f, base, "trunk")
		// Which commit is credited is not pinned here: two entities hold the id
		// across this history, so reading the prior id back through the range can
		// reach the other one's commits. The report itself is what matters, and
		// the commit it names is one an acknowledgment can reach.
		if len(got) != 1 || got[0].EntityID != "G-0002" || got[0].Section != whyItMatters || got[0].SHA == "" {
			t.Errorf("want one G-0002 %q finding naming some commit, got %+v", whyItMatters, got)
		}
	})

	// A file at an entity path on trunk that carries no frontmatter is not an
	// entity there, and exempts nothing.
	t.Run("ignores a trunk file that carries no entity frontmatter", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		f.put(gapPath, "## What's missing\n\nnot an entity\n", "a file that is not an entity")
		f.run("git", "branch", "trunk")
		f.put(gapPath, full, "make it an entity", "aiwf-verb: add", "aiwf-entity: G-0001", "aiwf-actor: human/test")
		base := f.head()
		drop := f.put(gapPath, partial, "drop a section", wrapTrailers...)
		assertDropped(t, []DroppedBodySection{{SHA: drop, Path: gapPath, EntityID: "G-0001", Section: whyItMatters}}, walkWithTrunk(t, f, base, "trunk"))
	})

	// A merge that writes a new entity in its own resolution lists no commit that
	// added it, so the push has no verb-written body to start from and the entity
	// is held to the whole set.
	t.Run("holds an entity a merge itself created to the whole set", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		f.put("work/gaps/G-0002-other.md", gapFile("G-0002", "", whatsMissing, whyItMatters), "seed")
		base := f.head()
		f.run("git", "checkout", "-q", "-b", "side")
		f.put("work/gaps/G-0003-side.md", gapFile("G-0003", "", whatsMissing, whyItMatters), "side work")
		f.run("git", "checkout", "-q", "main")
		f.run("git", "merge", "-q", "--no-ff", "--no-commit", "side")
		merge := f.put(gapPath, partial, "merge, and write a new gap in the same commit")
		assertDropped(t, []DroppedBodySection{{SHA: merge, Path: gapPath, EntityID: "G-0001", Section: whyItMatters}}, walkFrom(t, f, base))
	})

	// An entity gone at HEAD has no body the push publishes.
	t.Run("does not report an entity the push deleted", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		f.put(gapPath, full, "seed")
		base := f.head()
		f.run("git", "rm", "-q", gapPath)
		f.commit("delete")
		assertDropped(t, nil, walkFrom(t, f, base))
	})

	// git quotes a path carrying non-ASCII bytes unless told otherwise, and a
	// quoted path matches no entity shape.
	t.Run("judges an entity whose path carries non-ASCII characters", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		const accented = "work/gaps/G-0001-café crème.md"
		f.put(accented, full, "seed")
		base := f.head()
		drop := f.put(accented, partial, "drop")
		assertDropped(t, []DroppedBodySection{{SHA: drop, Path: accented, EntityID: "G-0001", Section: whyItMatters}}, walkFrom(t, f, base))
	})

	// A file under an entity path that carries no frontmatter is not an entity;
	// unexpected-tree-file and load-error report it.
	t.Run("does not judge a file carrying no entity frontmatter", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		f.put(gapPath, full, "seed")
		base := f.head()
		f.put(gapPath, "## What's missing\n\nno frontmatter\n", "strip frontmatter")
		assertDropped(t, nil, walkFrom(t, f, base))
	})

	// A file that was not yet an entity when the push started has no body to
	// start from, so the commit that made it one is held to the whole set.
	t.Run("holds a file that became an entity during the push to the whole set", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		f.put(gapPath, "## What's missing\n\nnot an entity yet\n", "seed")
		base := f.head()
		became := f.put(gapPath, partial, "give it frontmatter", wrapTrailers...)
		assertDropped(t, []DroppedBodySection{{SHA: became, Path: gapPath, EntityID: "G-0001", Section: whyItMatters}}, walkFrom(t, f, base))
	})

	// An epic directory carrying no id in its name is epic-shaped and names no
	// entity, so there is nothing to credit a finding to.
	t.Run("does not judge an entity path whose id does not parse", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		const noID = "work/epics/no-id-here/epic.md"
		f.put(noID, "---\nid: E-0001\n---\n## Goal\n\ng\n\n## Scope\n\ns\n\n## Out of scope\n\no\n", "seed")
		base := f.head()
		f.put(noID, "---\nid: E-0001\n---\n## Goal\n\ng\n\n## Scope\n\ns\n", "drop a section")
		assertDropped(t, nil, walkFrom(t, f, base))
	})

	// A base that resolves to no commit gives the push no starting point; the
	// provenance audit reports an unresolvable --since itself.
	t.Run("returns nothing for a base that resolves to no commit", func(t *testing.T) {
		t.Parallel()
		f := newWalkerFixture(t)
		f.put(gapPath, partial, "seed")
		assertDropped(t, nil, walkFrom(t, f, "no-such-ref"))
	})
}

// TestWalkDroppedBodySections_WhatCountsAsAViolation states what, inside the
// gate's scope, is a violation: a section the kind requires, not present as a
// top-level `## ` heading. Sections beyond the declared set are legal, order
// carries no meaning, and a required heading nested below top level is absent —
// ParseBodySections, the parser `aiwf show` and the body rules share, reads
// `## ` alone.
func TestWalkDroppedBodySections_WhatCountsAsAViolation(t *testing.T) {
	t.Parallel()
	const specPath = "work/epics/E-0001-seed/M-0001-seed.md"
	const frontmatter = "---\nid: M-0001\ntitle: Seed\nstatus: in_progress\nparent: E-0001\n---\n"
	const whole = frontmatter + "## Goal\n\nShip it.\n\n## Acceptance criteria\n\n### AC-1 — It ships\n\nIt ships.\n"

	cases := []struct {
		name  string
		after string
		want  []string
	}{
		{
			name:  "a section beyond the declared set is legal",
			after: whole + "\n## Notes\n\nAn author's own heading.\n",
		},
		{
			name:  "order is not enforced",
			after: frontmatter + "## Acceptance criteria\n\n### AC-1 — It ships\n\nIt ships.\n\n## Goal\n\nShip it.\n",
		},
		{
			name:  "a required heading nested below top level is absent",
			after: frontmatter + "## Goal\n\nShip it.\n\n### Acceptance criteria\n\n### AC-1 — It ships\n\nIt ships.\n",
			want:  []string{"Acceptance criteria"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newWalkerFixture(t)
			f.put(specPath, whole, "seed the spec")
			base := f.head()
			f.put(specPath, tc.after, "rework the body")
			var got []string
			for _, d := range walkFrom(t, f, base) {
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
