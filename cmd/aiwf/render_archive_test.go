package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupArchiveRenderFixture writes a fresh repo containing one active
// epic, one active gap, and one archived gap. Returns the consumer
// repo root. The fixture is sized so that the AC-6/7/8 assertions
// have at least one each of:
//
//   - an active epic (E-0001) — must appear on the active per-kind
//     epic index
//   - an active gap (G-0001) — must appear on the active per-kind
//     gap index AND on the all-set gap page
//   - an archived gap (G-0099) — must NOT appear on the active gap
//     index, must appear on the gaps-all page, and MUST have its
//     own per-entity page rendered so deep links resolve (AC-8)
//
// Direct on-disk writes for the archived gap (rather than the M-0085
// archive verb) keep the fixture independent of the verb's preflight
// — the renderer's contract is "render what's loaded," not "run the
// archive verb first."
func setupArchiveRenderFixture(t *testing.T) string {
	t.Helper()
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Active Epic", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "gap", "--title", "Active gap", "--actor", "human/test", "--root", root)

	// Archived gap: terminal status, lives under work/gaps/archive/
	// per ADR-0004 storage table.
	archiveDir := filepath.Join(root, "work", "gaps", "archive")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(archiveDir, "G-0099-archived.md"), []byte(`---
id: G-0099
title: Archived gap
status: addressed
addressed_by:
    - M-0001
---
## What's missing

Archived gap body for render fixture.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// TestRender_PerKindIndexShowsActiveOnly — M-0087/AC-6: the per-kind
// gaps index page lists active gaps and excludes archived ones.
// Structural assertion: parse the gaps.html page, scope the substring
// match to the <main> region (the listing body — the sidebar is a
// separate nav surface), and assert both (a) the active id is present
// and (b) the archived id is absent in that scope. A flat
// strings.Contains would not distinguish "archived id appears in
// listing" from "archived id appears only in sidebar nav" — the
// <main> scope is the load-bearing guarantee.
//
// Why <main>: after the M-0087 visibility regression fix, the
// listing body lives directly under <main> rather than inside a
// <section data-tab="kind-listing"> wrapper (the wrapper was
// hidden by default per the stylesheet's section[data-tab] rule).
// The visibility test in render_archive_visibility_test.go pins
// the no-hidden-wrapper property; this test pins the active-only
// content property.
func TestRender_PerKindIndexShowsActiveOnly(t *testing.T) {
	root := setupArchiveRenderFixture(t)
	out := filepath.Join(t.TempDir(), "site")
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out)

	gapsHTML := readFileT(t, filepath.Join(out, "gaps.html"))
	listing := htmlMain(gapsHTML)
	if listing == "" {
		t.Fatalf("gaps.html missing <main> body:\n%s", gapsHTML)
	}
	if !strings.Contains(listing, "G-0001") {
		t.Errorf("gaps.html <main> missing active id G-0001:\n%s", listing)
	}
	if strings.Contains(listing, "G-0099") {
		t.Errorf("gaps.html <main> leaks archived id G-0099 (active-default must hide archive):\n%s", listing)
	}
}

// (Removed under M-0099/AC-1: TestRender_PerKindIndexLinksToAllPage,
//  TestRender_KindAllPageShowsActiveAndArchived,
//  TestRender_KindAllPageLinksBackToActiveDefault — these pinned the
//  pre-migration active/all-pair design where `gaps-all.html` was a
//  separate file with an escape-hatch link between the two pages.
//  E-0029/M-0099 collapses that to a single emitted file per kind
//  with a :target-driven chip filter handling the active-vs-all
//  toggle client-side; the new behavior is asserted in
//  e2e/playwright/tests/render.spec.ts under the
//  `kind-index — file emission` and `kind-index — chip filter`
//  describes. The branch-coverage of the resolver's KindIndexData
//  with includeArchived=true is preserved by AC-3's behavior tests
//  once they land.)

// TestRender_PerEntityPageRendersForArchivedEntity — M-0087/AC-8:
// per-entity HTML pages render regardless of status — deep links
// to archived entity ids resolve rather than 404. Asserts both
// the file exists and its content names the archived entity's
// title.
func TestRender_PerEntityPageRendersForArchivedEntity(t *testing.T) {
	root := setupArchiveRenderFixture(t)
	out := filepath.Join(t.TempDir(), "site")
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out)

	archivedPagePath := filepath.Join(out, "G-0099.html")
	if _, err := os.Stat(archivedPagePath); err != nil {
		t.Fatalf("per-entity page for archived id missing: %v", err)
	}
	body := readFileT(t, archivedPagePath)
	if !strings.Contains(body, "Archived gap") {
		t.Errorf("G-0099.html missing archived entity's title:\n%s", body)
	}
}

// TestRender_PerEntityPageMarksArchivedState — M-0087/AC-8: the
// per-entity page for an archived entity carries a visible
// archived-state marker, paralleling the `aiwf show` indicator
// (AC-5). Structural assertion scoped to the page's status
// element: the marker lives near the status badge in the page
// header. Active per-entity pages have no marker.
func TestRender_PerEntityPageMarksArchivedState(t *testing.T) {
	root := setupArchiveRenderFixture(t)
	out := filepath.Join(t.TempDir(), "site")
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out)

	archivedBody := readFileT(t, filepath.Join(out, "G-0099.html"))
	// The marker is the dedicated `<span class="archived-marker">`
	// next to the status pill. Class-based selector avoids the
	// substring-collision-with-prose risk that a bare "archived"
	// match would have (the gap body literally contains "Archived
	// gap body for render fixture").
	if !strings.Contains(archivedBody, `class="archived-marker"`) {
		t.Errorf("G-0099.html (archived) missing archived-marker span:\n%s", archivedBody)
	}

	activeBody := readFileT(t, filepath.Join(out, "G-0001.html"))
	if strings.Contains(activeBody, `class="archived-marker"`) {
		t.Errorf("G-0001.html (active) carries archived-marker span; only archived per-entity pages should:\n%s", activeBody)
	}
}

// TestRender_IndexLinksPerKindPages — M-0087/AC-6: the home/index
// page surfaces the new per-kind index links (gaps.html,
// decisions.html, adrs.html, contracts.html). Without these the
// AC-6 active-default pages are unreachable from the home nav.
//
// Scoped assertion: the links must appear in the page's
// kind-index nav block (a `<nav class="kind-index">` element)
// inside <main>, not in the sidebar — the sidebar is a different
// navigation surface and may evolve independently.
//
// Why <nav class="kind-index"> rather than the earlier
// <section data-tab="kind-index"> wrapper: see
// render_archive_visibility_test.go for the regression that
// prompted the change — the data-tab wrapper was hidden by
// default per the embedded stylesheet's section[data-tab] rule.
func TestRender_IndexLinksPerKindPages(t *testing.T) {
	root := setupArchiveRenderFixture(t)
	out := filepath.Join(t.TempDir(), "site")
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out)

	indexHTML := readFileT(t, filepath.Join(out, "index.html"))
	nav := htmlElementByClass(indexHTML, "nav", "kind-index")
	if nav == "" {
		t.Fatalf("index.html missing <nav class=\"kind-index\"> block:\n%s", indexHTML)
	}
	for _, href := range []string{
		`href="gaps.html"`,
		`href="decisions.html"`,
		`href="adrs.html"`,
		`href="contracts.html"`,
	} {
		if !strings.Contains(nav, href) {
			t.Errorf("index.html kind-index nav missing %s:\n%s", href, nav)
		}
	}
}
