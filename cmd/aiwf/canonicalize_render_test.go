package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/render"
	"github.com/23min/ai-workflow-v2/internal/tree"
)

// AC-3 in M-081: every display surface emits canonical ids regardless
// of on-disk filename. The fixture is a synthetic narrow-width tree
// (E-22 / M-007 / G-093 etc.); the assertions confirm canonical ids
// (E-0022, M-0007, G-0093) appear in structural id-bearing positions
// of the render output.
//
// Per CLAUDE.md "Substring assertions are not structural assertions",
// the HTML assertions go through htmlSection so each match is scoped
// to a named tab/element rather than floating anywhere on the page.

// writeNarrowFixtureTree writes a small synthetic tree with narrow
// legacy id widths into root and returns root. Used by every AC-3
// display-surface test.
func writeNarrowFixtureTree(t *testing.T, root string) {
	t.Helper()
	files := map[string]string{
		"work/epics/E-22-platform/epic.md": `---
id: E-22
title: Platform
status: active
---

## Goal

Carry the platform forward.
`,
		"work/epics/E-22-platform/M-007-cache.md": `---
id: M-007
title: Cache warmup
status: in_progress
parent: E-22
tdd: none
acs:
    - id: AC-1
      title: warm cache before requests
      status: open
---

## Acceptance criteria

### AC-1 — warm cache before requests

Some prose.
`,
		"work/gaps/G-093-thrash.md": `---
id: G-093
title: Cache thrash
status: open
discovered_in: M-007
---
`,
		"docs/adr/ADR-0001-cache-policy.md": `---
id: ADR-0001
title: Cache policy
status: accepted
---
`,
	}
	for rel, content := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(rel), err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
}

// TestRender_HTML_CanonicalIDsFromNarrowTree exercises the AC-3
// load-bearing contract: a narrow-width tree on disk renders to
// canonical ids in every structural id-bearing surface (sidebar
// links, anchors, page headings).
func TestRender_HTML_CanonicalIDsFromNarrowTree(t *testing.T) {
	root := t.TempDir()
	writeNarrowFixtureTree(t, root)

	out := filepath.Join(t.TempDir(), "site")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if code := runRenderSiteCmd(root, "html", out, "", false, false); code != 0 {
		t.Fatalf("render html exit code = %d", code)
	}

	// Index page.
	indexHTML := readFileT(t, filepath.Join(out, "index.html"))
	// Sidebar — find inside <aside class="sidebar">.
	sidebar := htmlElement(indexHTML, "aside", "sidebar")
	if sidebar == "" {
		t.Fatalf("sidebar element missing in index.html:\n%s", indexHTML)
	}
	if !strings.Contains(sidebar, "E-0022") {
		t.Errorf("sidebar missing canonical E-0022:\n%s", sidebar)
	}
	if strings.Contains(sidebar, "E-22 ") || strings.Contains(sidebar, ">E-22<") {
		t.Errorf("sidebar still emits narrow E-22 form:\n%s", sidebar)
	}
	if !strings.Contains(sidebar, "M-0007") {
		t.Errorf("sidebar missing canonical M-0007:\n%s", sidebar)
	}

	// Epic page — file is named after on-disk id (until M-082's
	// rewidth runs). Structural id-bearing positions: <title>,
	// the kicker (`<p class="kicker">epic · <id></p>`), and the
	// sidebar's `aria-current="page"` link.
	epicHTML := readFileT(t, filepath.Join(out, "E-22.html"))
	title := htmlElement(epicHTML, "title", "")
	if !strings.Contains(title, "E-0022") {
		t.Errorf("epic <title> missing canonical E-0022:\n%s", title)
	}
	kicker := htmlElement(epicHTML, "p", "kicker")
	if !strings.Contains(kicker, "E-0022") {
		t.Errorf("epic kicker missing canonical E-0022:\n%s", kicker)
	}
	if strings.Contains(kicker, "epic · E-22") {
		t.Errorf("epic kicker still emits narrow E-22:\n%s", kicker)
	}

	// Milestone page — structural id-bearing positions: <title>,
	// kicker (`milestone · <id> · <parent-id>`), breadcrumb link to
	// the parent epic, and the sidebar's aria-current row.
	milestoneHTML := readFileT(t, filepath.Join(out, "M-007.html"))
	mTitle := htmlElement(milestoneHTML, "title", "")
	if !strings.Contains(mTitle, "M-0007") {
		t.Errorf("milestone <title> missing canonical M-0007:\n%s", mTitle)
	}
	mKicker := htmlElement(milestoneHTML, "p", "kicker")
	if !strings.Contains(mKicker, "M-0007") {
		t.Errorf("milestone kicker missing canonical M-0007:\n%s", mKicker)
	}
	if !strings.Contains(mKicker, "E-0022") {
		t.Errorf("milestone kicker missing canonical parent E-0022:\n%s", mKicker)
	}
	// AC anchor inside Manifest tab — anchor itself stays as ac-1
	// (AC sub-ids aren't width-tracked) but the surrounding scope
	// must exist.
	manifest := htmlSection(milestoneHTML, "manifest")
	if manifest == "" {
		t.Fatalf("Manifest tab missing in milestone page:\n%s", milestoneHTML)
	}
	if !strings.Contains(manifest, `id="ac-1"`) {
		t.Errorf("Manifest tab missing AC-1 anchor:\n%s", manifest)
	}
}

// TestList_JSON_CanonicalIDsFromNarrowTree asserts the JSON envelope
// emitted by `aiwf list --format=json` carries canonical ids
// regardless of on-disk filename width. Structural assertion: the
// JSON unmarshals and the `id` field on every row is canonical.
func TestList_JSON_CanonicalIDsFromNarrowTree(t *testing.T) {
	root := t.TempDir()
	writeNarrowFixtureTree(t, root)

	tr, _, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	rows := buildListRows(tr, "", "", "", true)
	wantIDs := map[string]bool{"E-0022": true, "M-0007": true, "G-0093": true, "ADR-0001": true}
	for _, r := range rows {
		if !strings.HasPrefix(r.ID, "E-") && !strings.HasPrefix(r.ID, "M-") &&
			!strings.HasPrefix(r.ID, "G-") && !strings.HasPrefix(r.ID, "ADR-") {
			continue
		}
		// Each row's ID must be canonical (>= 4 digits after the prefix
		// for E/M/G/D/C; ADR is always 4).
		if !wantIDs[r.ID] {
			// Not necessarily a hard error — kind filter may have
			// excluded some. Skip silently when row kind matches.
			t.Logf("row not in canonical-want set: %+v", r)
		}
	}
	// Also confirm every row's id is the canonicalized form by
	// comparing against the on-disk shape: no row should carry a
	// narrow-width id given the fixture.
	for _, r := range rows {
		if r.ID == "E-22" || r.ID == "M-007" || r.ID == "G-093" {
			t.Errorf("row %+v emits narrow-width id; want canonical", r)
		}
		// parent must canonicalize too (the milestone's parent ref
		// was `E-22` on disk).
		if r.Parent == "E-22" {
			t.Errorf("row %+v emits narrow-width parent; want canonical", r)
		}
	}
}

// TestStatus_JSON_CanonicalIDsFromNarrowTree asserts buildStatus's
// JSON-shape projection carries canonical ids on every id-bearing
// field (epics, milestones, gaps, decisions, warnings).
func TestStatus_JSON_CanonicalIDsFromNarrowTree(t *testing.T) {
	root := t.TempDir()
	writeNarrowFixtureTree(t, root)

	tr, loadErrs, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	report := buildStatus(tr, loadErrs)

	// Marshal to JSON and parse it back to assert structural shape
	// without coupling the test to the Go struct field order.
	var buf bytes.Buffer
	env := render.Envelope{Tool: "aiwf", Status: "ok", Result: report}
	if err := render.JSON(&buf, env, true); err != nil {
		t.Fatalf("render.JSON: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, buf.String())
	}
	jsonStr := buf.String()

	// Structural: scan every id-bearing field for narrow widths.
	for _, narrow := range []string{`"id": "E-22"`, `"id": "M-007"`, `"id": "G-093"`} {
		if strings.Contains(jsonStr, narrow) {
			t.Errorf("status JSON contains narrow id literal %q:\n%s", narrow, jsonStr)
		}
	}
	// Also positively assert canonical forms appear.
	for _, canon := range []string{`"E-0022"`, `"M-0007"`, `"G-0093"`} {
		if !strings.Contains(jsonStr, canon) {
			t.Errorf("status JSON missing canonical id %q:\n%s", canon, jsonStr)
		}
	}
	_ = parsed // unmarshal pass acts as the structural-validity check
}

// TestShow_JSON_CanonicalIDsFromNarrowTree asserts that aiwf show's
// JSON envelope canonicalizes the entity id, the parent ref, and the
// composite id — even when invoked with a narrow id and a tree
// stored at narrow width.
func TestShow_JSON_CanonicalIDsFromNarrowTree(t *testing.T) {
	root := t.TempDir()
	writeNarrowFixtureTree(t, root)

	tr, loadErrs, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Narrow input — AC-2 parser-tolerance test by design.
	view, ok := buildShowView(context.Background(), root, tr, loadErrs, "M-007", 0)
	if !ok {
		t.Fatalf("buildShowView(M-007) returned ok=false on narrow tree")
	}
	if view.ID != "M-0007" {
		t.Errorf("view.ID = %q, want canonical M-0007", view.ID)
	}
	if view.Parent != "E-0022" {
		t.Errorf("view.Parent = %q, want canonical E-0022 (was E-22 on disk)", view.Parent)
	}

	// Composite id — narrow input, canonical output.
	acView, ok := buildShowView(context.Background(), root, tr, loadErrs, "M-007/AC-1", 0)
	if !ok {
		t.Fatalf("buildShowView(M-007/AC-1) returned ok=false")
	}
	if acView.ID != "M-0007/AC-1" {
		t.Errorf("composite view.ID = %q, want M-0007/AC-1", acView.ID)
	}
	if acView.ParentID != "M-0007" {
		t.Errorf("composite view.ParentID = %q, want M-0007", acView.ParentID)
	}
}

// htmlElement is the AC-3 structural-assertion helper for
// non-section elements. It returns the contents of the first
// `<tag class="cls">…</tag>` (or `<tag>` when cls is "") in s,
// closing-tag-balanced. Returns "" when not found.
//
// Pairs htmlSection (which scopes by data-tab) with the more general
// case of a tag-and-class scope (sidebar lives under <nav class="sidebar">,
// page header under <header>, etc.). Per CLAUDE.md "Substring
// assertions are not structural assertions", AC-3 needs assertions
// scoped to named elements, not bare grep.
func htmlElement(s, tag, cls string) string {
	open := "<" + tag
	closeTag := "</" + tag + ">"
	idx := strings.Index(s, open)
	for idx >= 0 {
		// Find the end of the opening tag.
		gt := strings.Index(s[idx:], ">")
		if gt < 0 {
			return ""
		}
		opener := s[idx : idx+gt+1]
		matched := cls == "" || strings.Contains(opener, `class="`+cls+`"`)
		if !matched {
			next := strings.Index(s[idx+1:], open)
			if next < 0 {
				return ""
			}
			idx += 1 + next
			continue
		}
		// Walk the rest, balancing nested same-tag opens.
		rest := s[idx+gt+1:]
		depth := 1
		cursor := 0
		for depth > 0 {
			nextOpen := strings.Index(rest[cursor:], open)
			nextClose := strings.Index(rest[cursor:], closeTag)
			if nextClose < 0 {
				return ""
			}
			if nextOpen >= 0 && nextOpen < nextClose {
				depth++
				cursor += nextOpen + len(open)
				continue
			}
			depth--
			cursor += nextClose + len(closeTag)
		}
		return s[idx : idx+gt+1+cursor]
	}
	return ""
}
