package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// residualOwningGaps are the gaps that own the residual ADR-0033 declines
// to cover mechanically — links from non-entity files into the entity
// tree. M-0317/AC-2 routes the delegation measurement to them.
var residualOwningGaps = []string{"G-0478", "G-0439"}

// TestM0317_AC2_ADR0033NamesTheGapsOwningItsResidual is the mechanical
// evidence for M-0317/AC-2.
//
// The measurement's consequence for ADR-0033 is that its second bullet
// named a delegate carrying no mechanical trigger, while the check that
// does cover the class went unmentioned. Correcting the prose is not what
// makes the finding survive: what does is that a reader of the decision
// can reach the gaps that own the residual, so the next person to ask
// "what covers non-entity narrative?" lands on the measurement instead of
// re-deriving it.
//
// A relationship check rather than a phrase assertion: it compares the
// ADR against the tree, so deleting either gap or dropping its citation
// turns it red, while rewording any of the three does not. It is also not
// redundant with `body-prose-id`, which fires on a *dangling* id and
// never on a *missing* one — the citation going absent is exactly the
// failure that rule cannot see.
//
// Retires when both gaps reach a terminal status: the residual then has
// an answer rather than an owner, and the citation becomes history.
func TestM0317_AC2_ADR0033NamesTheGapsOwningItsResidual(t *testing.T) {
	t.Parallel()

	root, tr := sharedRepoTree(t)

	adr := tr.ByID("ADR-0033")
	if adr == nil {
		t.Fatal("ADR-0033 does not resolve via tr.ByID")
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(adr.Path)))
	if err != nil {
		t.Fatalf("reading ADR-0033 at %s: %v", adr.Path, err)
	}
	refs := sectionBody(string(raw), "\n## References")
	if refs == "" {
		t.Fatal("ADR-0033 has no `## References` section to carry the citations")
	}

	for _, id := range residualOwningGaps {
		if tr.ByID(id) == nil {
			t.Errorf("%s does not resolve via tr.ByID — ADR-0033's residual has no owning gap to reach", id)
			continue
		}
		if !strings.Contains(refs, id) {
			t.Errorf("ADR-0033's `## References` does not name %s, so the gap owning its non-entity residual is not reachable from the decision:\n%s", id, refs)
		}
	}
}

// TestM0317_MarkdownLinkRegexShapes pins which CommonMark destination
// forms markdownLinkRegex reaches.
//
// The pattern's doc comment makes a behavioral claim about the titled
// form that nothing else checks — a mutant widening the capture to admit
// that form leaves the rest of the package green. The two unreached
// shapes are recorded in G-0624, which carries whether to widen; this
// test is what makes that gap's premise re-checkable rather than
// remembered.
//
// Retires with G-0624: whichever way that decision goes, it rewrites both
// the pattern's contract and this table.
func TestM0317_MarkdownLinkRegexShapes(t *testing.T) {
	t.Parallel()

	const dest = "work/gaps/G-0001-a.md"

	tests := []struct {
		name      string
		link      string
		wantMatch bool
		want      string // captured destination when wantMatch
	}{
		{name: "bare destination", link: "[x](" + dest + ")", wantMatch: true, want: dest},
		{name: "titled destination is not matched at all", link: `[x](` + dest + ` "t")`},
		{name: "pointy brackets are captured with the destination", link: "[x](<" + dest + ">)", wantMatch: true, want: "<" + dest + ">"},
		// wantMatch distinguishes "no match" from "matched with an empty
		// capture" — without it the case is satisfied either way, and a
		// pattern widened to accept an empty destination passes.
		{name: "empty destination is not matched", link: "[x]()"},
		{name: "empty link text still matches", link: "[](" + dest + ")", wantMatch: true, want: dest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := markdownLinkRegex.FindStringSubmatch(tc.link)
			if (m != nil) != tc.wantMatch {
				t.Fatalf("markdownLinkRegex on %q matched = %v, want %v", tc.link, m != nil, tc.wantMatch)
			}
			if tc.wantMatch && m[1] != tc.want {
				t.Errorf("markdownLinkRegex on %q captured %q, want %q", tc.link, m[1], tc.want)
			}
		})
	}
}
