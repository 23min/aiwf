package main

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The visibility regression these tests pin (M-0087 AC-5/6/7 reopen):
//
// The stylesheet at internal/htmlrender/embedded/style.css contains:
//
//	section[data-tab] { display: none; padding: 0.5rem 0; }
//	section[data-tab="overview"] { display: block; }
//	section[data-tab]:target { display: block; }
//
// Any <section data-tab="X"> in the rendered output is hidden by
// default unless X is "overview" or the section becomes the
// fragment `:target` (which requires a tab-nav anchor click).
//
// The initial M-0087 implementation wrapped the home page's
// "Browse by kind" nav block in <section data-tab="kind-index">
// and every per-kind index page's main listing in
// <section data-tab="kind-listing">. Neither block had a tab-nav
// control, so the rendered pages displayed as empty in a browser.
// The prior structural tests asserted "the section exists" and
// "the table is inside it" but never asked "is this section
// visible per the page's own CSS?" — exactly the failure mode
// CLAUDE.md "Substring assertions are not structural assertions"
// and "Render output must be human-verified before the iteration
// closes" exist to prevent.
//
// These tests close that loophole. They assert, structurally,
// that the load-bearing content on index.html and per-kind index
// pages is NOT wrapped in any <section data-tab="..."> element
// other than "overview" — so the CSS hide-by-default rule cannot
// hide it. If a future change re-introduces a hidden wrapper, the
// test fails with the offending data-tab value named.

// sectionDataTabOpenRE matches a <section ...> opener carrying a
// data-tab attribute, capturing the attribute's value.
var sectionDataTabOpenRE = regexp.MustCompile(`<section\b[^>]*\bdata-tab="([^"]*)"[^>]*>`)

// sectionOpenRE matches any <section ...> opener (with or without
// data-tab). Used for ancestor-depth tracking.
var sectionOpenRE = regexp.MustCompile(`<section\b[^>]*>`)

// enclosingHiddenDataTabs returns the data-tab values of every
// <section data-tab="X"> element that encloses the byte offset
// `pos` inside `html`, excluding sections where X is the
// always-visible "overview" value per the embedded stylesheet.
//
// Algorithm: scan the prefix html[:pos], maintain a stack of open
// <section> elements (their data-tab value if any, otherwise the
// empty string). For every <section ...> opener encountered in the
// prefix, push; for every </section>, pop. The remaining stack
// after the scan is the chain of enclosing sections at `pos`.
//
// Returns only data-tab values that the CSS hide-by-default rule
// would mark `display: none` — i.e. any value other than
// "overview". An empty result means the content at `pos` is
// reachable per the page's own CSS.
func enclosingHiddenDataTabs(html string, pos int) []string {
	if pos < 0 || pos > len(html) {
		return nil
	}
	prefix := html[:pos]

	// Build a unified event stream of every <section ...> opener
	// and every </section> closer in the prefix, in source order.
	type event struct {
		offset  int
		isOpen  bool
		dataTab string // populated on opens; "" if the opener carried no data-tab
	}
	var events []event

	for _, m := range sectionOpenRE.FindAllStringIndex(prefix, -1) {
		dataTab := ""
		if dm := sectionDataTabOpenRE.FindStringSubmatch(prefix[m[0]:m[1]]); dm != nil {
			dataTab = dm[1]
		}
		events = append(events, event{offset: m[0], isOpen: true, dataTab: dataTab})
	}
	for _, idx := range allIndexes(prefix, "</section>") {
		events = append(events, event{offset: idx, isOpen: false})
	}

	// Sort by source offset. Open and close events never share an
	// offset in well-formed HTML; sort.SliceStable preserves the
	// add order on ties as a defensive fallback.
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].offset < events[j].offset
	})

	var stack []string
	for _, e := range events {
		if e.isOpen {
			stack = append(stack, e.dataTab)
			continue
		}
		if len(stack) > 0 {
			stack = stack[:len(stack)-1]
		}
	}

	var hidden []string
	for _, dt := range stack {
		if dt == "" {
			continue // <section> with no data-tab — visible
		}
		if dt == "overview" {
			continue // whitelisted by the embedded stylesheet
		}
		hidden = append(hidden, dt)
	}
	return hidden
}

// allIndexes returns every starting offset of `needle` in `s`.
func allIndexes(s, needle string) []int {
	var out []int
	start := 0
	for {
		i := strings.Index(s[start:], needle)
		if i < 0 {
			return out
		}
		out = append(out, start+i)
		start += i + len(needle)
	}
}

// assertVisible fails the test if any content matching `marker`
// in `html` is enclosed in a hidden <section data-tab="..."> per
// the page's embedded stylesheet. `pageName` and `markerName`
// label the failure for the triage reader.
func assertVisible(t *testing.T, html, marker, pageName, markerName string) {
	t.Helper()
	idx := strings.Index(html, marker)
	if idx < 0 {
		t.Fatalf("%s: %s (%q) not found in rendered HTML", pageName, markerName, marker)
	}
	hidden := enclosingHiddenDataTabs(html, idx)
	if len(hidden) > 0 {
		t.Errorf("%s: %s is enclosed in hidden section(s) data-tab=%v — these are display:none by default per style.css. The wrapper must use a tag/attribute the stylesheet does not hide (e.g. <nav>, <section> without data-tab, or data-tab=\"overview\").",
			pageName, markerName, hidden)
	}
}

// TestRender_IndexKindIndexNavIsVisible — M-0087/AC-6 visibility
// pin: the home page's "Browse by kind" nav links must be
// reachable per the page's own CSS. The prior implementation
// wrapped the block in <section data-tab="kind-index">, which the
// stylesheet hides by default. This test asserts the per-kind
// link list is NOT inside any hidden data-tab section.
//
// Failure mode: a future change re-introduces a hidden wrapper
// around the kind-index nav. The test names the offending
// data-tab value so the regression is obvious.
func TestRender_IndexKindIndexNavIsVisible(t *testing.T) {
	root := setupArchiveRenderFixture(t)
	out := filepath.Join(t.TempDir(), "site")
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out)

	indexHTML := readFileT(t, filepath.Join(out, "index.html"))
	// The <ul class="kind-index"> element is the substantive
	// content — that's what the broken wrapper was hiding. Use
	// it as the marker; if it ends up inside a hidden section,
	// the page renders empty.
	assertVisible(t, indexHTML, `<ul class="kind-index">`, "index.html", "kind-index nav list")
}

// TestRender_PerKindIndexListingIsVisible — M-0087/AC-7 visibility
// pin: the per-kind index page's main content (the table of
// entities) must be reachable per the page's own CSS. The prior
// implementation wrapped the entire <main> body in <section
// data-tab="kind-listing">, hiding the whole listing. This test
// asserts the listing table is NOT inside any hidden data-tab
// section, on both the active-default and all-set pages.
func TestRender_PerKindIndexListingIsVisible(t *testing.T) {
	root := setupArchiveRenderFixture(t)
	out := filepath.Join(t.TempDir(), "site")
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out)

	cases := []struct {
		file   string
		marker string
	}{
		{"gaps.html", `<table class="kind-index">`},
		{"gaps-all.html", `<table class="kind-index">`},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			html := readFileT(t, filepath.Join(out, c.file))
			assertVisible(t, html, c.marker, c.file, "kind-listing table")
		})
	}
}

// TestEnclosingHiddenDataTabs_DetectsHiddenWrapper — unit test for
// the helper. Pins that a content marker inside <section
// data-tab="kind-listing"> is flagged hidden, that "overview" is
// whitelisted, and that a plain <nav> wrapper is not flagged.
func TestEnclosingHiddenDataTabs_DetectsHiddenWrapper(t *testing.T) {
	cases := []struct {
		name string
		html string
		// marker is the substring whose enclosing chain we inspect
		marker     string
		wantHidden []string
	}{
		{
			name:       "hidden kind-listing wrapper",
			html:       `<main><section data-tab="kind-listing"><table class="kind-index">data</table></section></main>`,
			marker:     `<table class="kind-index">`,
			wantHidden: []string{"kind-listing"},
		},
		{
			name:       "overview is whitelisted",
			html:       `<main><section data-tab="overview"><table>data</table></section></main>`,
			marker:     `<table>data</table>`,
			wantHidden: nil,
		},
		{
			name:       "plain nav wrapper is visible",
			html:       `<main><nav class="kind-index"><ul class="kind-index"><li>x</li></ul></nav></main>`,
			marker:     `<ul class="kind-index">`,
			wantHidden: nil,
		},
		{
			name:       "plain section without data-tab is visible",
			html:       `<main><section><table class="kind-index">data</table></section></main>`,
			marker:     `<table class="kind-index">`,
			wantHidden: nil,
		},
		{
			name:       "balanced open/close before marker leaves stack empty",
			html:       `<main><section data-tab="manifest"><p>x</p></section><table class="kind-index">data</table></main>`,
			marker:     `<table class="kind-index">`,
			wantHidden: nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			idx := strings.Index(c.html, c.marker)
			if idx < 0 {
				t.Fatalf("marker %q not found in fixture", c.marker)
			}
			got := enclosingHiddenDataTabs(c.html, idx)
			if len(got) != len(c.wantHidden) {
				t.Fatalf("enclosingHiddenDataTabs = %v, want %v", got, c.wantHidden)
			}
			for i, want := range c.wantHidden {
				if got[i] != want {
					t.Errorf("enclosingHiddenDataTabs[%d] = %q, want %q", i, got[i], want)
				}
			}
		})
	}
}
