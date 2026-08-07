package integration

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/entity"
)

// show_body_map_help_test.go — M-0305/AC-2. The root banner's `show`
// entry tells a --format=json consumer which keys the body map carries
// per kind. That list is a third statement of the per-kind section set,
// beside the owned table in internal/entity and the scaffold rendered
// from it — and prose is the copy no compiler checks.
//
// The banner is checked here rather than generated from the table: it
// is a sentence describing a projection, and a generated fragment
// spliced into hand-written prose reads worse than one a test keeps
// honest. What the test buys is that the description cannot go stale
// silently — a section added to or removed from the owned set fails
// here until the sentence follows.
//
// The assertion parses the named clause and compares the full per-kind
// list rather than grepping for slugs, per CLAUDE.md §"Substring
// assertions are not structural assertions": `goal` appearing somewhere
// in a 100-line banner proves nothing about the sentence a reader
// looking up the body map actually finds.
//
// Serial (no t.Parallel): captureHelpBanner swaps os.Stdout — see the
// skip-list in setup_test.go.
func TestShowBodyMapHelpNamesTheOwnedSectionSet(t *testing.T) {
	got := parseBodyMapClause(t, captureHelpBanner(t))

	want := map[entity.Kind][]string{}
	for _, k := range entity.AllKinds() {
		var slugs []string
		for _, section := range entity.RequiredSections(k) {
			slugs = append(slugs, entity.SectionSlug(section))
		}
		want[k] = slugs
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("the root banner's body-map clause disagrees with the owned section set (-want +got):\n%s\n\n"+
			"entity.RequiredSections in internal/entity/required_sections.go owns the per-kind set; the `show <id>` "+
			"line in internal/cli/root.go's printHelp() describes it. Update the banner clause to name each kind's "+
			"sections, slugified and in the owned order, as `<kind> <slug>/<slug>` segments separated by `; `.", diff)
	}
}

// parseBodyMapClause returns the `show <id>` banner entry's body-map
// description as kind → section slugs. The clause is the parenthesized
// group opened by the anchor below, and its contents are `<kind>
// <slug>/<slug>` segments separated by `;`.
//
// Every shape failure is fatal rather than an empty result: a clause
// that has been reworded or deleted must fail loudly, since a silently
// empty haystack would let the assertion above pass on nothing.
func parseBodyMapClause(t *testing.T, banner string) map[entity.Kind][]string {
	t.Helper()
	const anchor = "map of section-heading slug to prose:"

	_, rest, ok := strings.Cut(banner, anchor)
	if !ok {
		t.Fatalf("root help banner has no %q clause; printHelp() in internal/cli/root.go must describe\n"+
			"the `show --format=json` body map per kind.\nBanner was:\n%s", anchor, banner)
	}
	clause, _, ok := strings.Cut(rest, ")")
	if !ok {
		t.Fatalf("the body-map clause in the root help banner is never closed by `)`.\nClause was:\n%s", rest)
	}

	out := map[entity.Kind][]string{}
	for _, segment := range strings.Split(clause, ";") {
		kind, slugs, ok := strings.Cut(strings.TrimSpace(segment), " ")
		if !ok {
			t.Fatalf("body-map clause segment %q is not `<kind> <slug>/<slug>` shaped", segment)
		}
		out[entity.Kind(kind)] = strings.Split(slugs, "/")
	}
	return out
}
