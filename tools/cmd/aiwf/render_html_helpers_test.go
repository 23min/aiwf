package main

import (
	"regexp"
	"strings"
	"testing"
)

// htmlSection extracts the contents of the <section data-tab="<name>">
// ... </section> block from an aiwf-rendered milestone page. Returns
// the full element including the opening and closing tags. The
// helper is intentionally narrow: it pairs the opener at the
// nearest matching `</section>` at the same nesting depth.
//
// This is a structural assertion helper for the I3 milestone
// templates. Tests scope substring checks into the right tab via
// htmlSection(html, "tab-build") instead of grepping the whole
// document — without it, an AC anchor in the wrong tab would still
// satisfy the test.
//
// Returns "" when the section is not found.
func htmlSection(html, dataTab string) string {
	openRE := regexp.MustCompile(`<section[^>]*\bdata-tab="` + regexp.QuoteMeta(dataTab) + `"[^>]*>`)
	loc := openRE.FindStringIndex(html)
	if loc == nil {
		return ""
	}
	// Walk from after the opener, tracking nested <section> opens
	// and closes until depth returns to zero.
	rest := html[loc[1]:]
	depth := 1
	cursor := 0
	for depth > 0 {
		nextOpen := strings.Index(rest[cursor:], "<section")
		nextClose := strings.Index(rest[cursor:], "</section>")
		if nextClose < 0 {
			return ""
		}
		if nextOpen >= 0 && nextOpen < nextClose {
			depth++
			cursor += nextOpen + len("<section")
			continue
		}
		depth--
		cursor += nextClose + len("</section>")
	}
	return html[loc[0] : loc[1]+cursor]
}

// assertContainsIn fails the test when needle is not present inside
// section. The error message names the section so a triage reader
// knows which tab contained the wrong content.
func assertContainsIn(t *testing.T, html, dataTab, needle, label string) {
	t.Helper()
	section := htmlSection(html, dataTab)
	if section == "" {
		t.Errorf("%s: section [data-tab=%q] not found", label, dataTab)
		return
	}
	if !strings.Contains(section, needle) {
		t.Errorf("%s: section [data-tab=%q] missing %q\nsection content:\n%s", label, dataTab, needle, section)
	}
}

// assertNotContainsIn fails when needle appears inside section. The
// inverse of assertContainsIn — used to pin "this content does NOT
// leak into that tab" properties.
func assertNotContainsIn(t *testing.T, html, dataTab, needle, label string) {
	t.Helper()
	section := htmlSection(html, dataTab)
	if section == "" {
		t.Errorf("%s: section [data-tab=%q] not found", label, dataTab)
		return
	}
	if strings.Contains(section, needle) {
		t.Errorf("%s: section [data-tab=%q] unexpectedly contains %q\nsection content:\n%s", label, dataTab, needle, section)
	}
}

// TestHTMLSection_NestedSectionsHandled pins the load-bearing case:
// <section data-tab="manifest"> contains nested <section class="ac">
// blocks per AC. The depth-tracker must stop at the first OUTER
// </section>, not the inner one.
func TestHTMLSection_NestedSectionsHandled(t *testing.T) {
	html := `<main>
<section data-tab="manifest" id="tab-manifest">
<h2>Manifest</h2>
<section class="ac" id="ac-1">
<h3>AC-1</h3>
</section>
<section class="ac" id="ac-2">
<h3>AC-2</h3>
</section>
</section>
<section data-tab="build" id="tab-build">
<h2>Build</h2>
</section>
</main>`
	got := htmlSection(html, "manifest")
	for _, want := range []string{`id="ac-1"`, `id="ac-2"`, `<h2>Manifest</h2>`} {
		if !strings.Contains(got, want) {
			t.Errorf("manifest section missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "<h2>Build</h2>") {
		t.Errorf("manifest section leaked into build:\n%s", got)
	}
	build := htmlSection(html, "build")
	if !strings.Contains(build, "<h2>Build</h2>") {
		t.Errorf("build section missing build header in:\n%s", build)
	}
	if strings.Contains(build, `id="ac-1"`) {
		t.Errorf("build section leaked manifest content:\n%s", build)
	}
}

// TestHTMLSection_MissingReturnsEmpty: a tab name that doesn't
// exist returns "" so callers can detect the absence cleanly.
func TestHTMLSection_MissingReturnsEmpty(t *testing.T) {
	html := `<section data-tab="overview">x</section>`
	if got := htmlSection(html, "tests"); got != "" {
		t.Errorf("missing tab should return empty; got %q", got)
	}
}
