package integration

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// priorityRowSlice returns the inner HTML of the <tr> row linking to
// id.html on a per-kind index page — from that row's opening <tr ...>
// tag up to its closing </tr> — so a priority-badge assertion is scoped
// to the one entity's own row, not a flat substring match against the
// whole table (CLAUDE.md "substring assertions are not structural
// assertions"; mirrors areaGroupSlice's containment-scoping idiom).
//
// The search is scoped to the <table class="kind-index"> body: every
// page also carries a sidebar with its own href="<id>.html" links (the
// epic-tree nav), so a page-wide search would false-match there instead
// of the table row.
func priorityRowSlice(t *testing.T, html, id string) string {
	t.Helper()
	tableStart := strings.Index(html, `<table class="kind-index">`)
	if tableStart < 0 {
		t.Fatalf("no kind-index table found:\n%s", html)
	}
	table := html[tableStart:]
	link := `href="` + id + `.html"`
	i := strings.Index(table, link)
	if i < 0 {
		t.Fatalf("no row linking to %s.html found in the kind-index table:\n%s", id, table)
	}
	rowStart := strings.LastIndex(table[:i], "<tr")
	if rowStart < 0 {
		t.Fatalf("no <tr opening before the %s row", id)
	}
	rowEndRel := strings.Index(table[i:], "</tr>")
	if rowEndRel < 0 {
		t.Fatalf("no closing </tr> after the %s row", id)
	}
	return table[rowStart : i+rowEndRel]
}

// entityHeaderSlice returns an individual entity detail page's header
// block (status / priority / archived markers) — from the closing
// </h1> tag up to the first body <section> — so a priority-badge
// assertion is scoped to the header, not the whole page.
func entityHeaderSlice(t *testing.T, html string) string {
	t.Helper()
	i := strings.Index(html, "</h1>")
	if i < 0 {
		t.Fatalf("no </h1> found:\n%s", html)
	}
	rest := html[i:]
	end := strings.Index(rest, "<section")
	if end < 0 {
		end = len(rest)
	}
	return rest[:end]
}

// renderSiteHTML renders root into a fresh tempdir and returns the
// output directory.
func renderSiteHTML(t *testing.T, root string) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "site")
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out)
	return out
}

// TestRun_RenderHTML_PriorityBadge_KindIndexRows pins M-0264/AC-1: a
// gap or decision carrying a priority gets a `priority priority-<level>`
// badge on its own row in the per-kind index page (gaps.html /
// decisions.html); an untagged gap's row renders no badge at all — not
// an empty one.
func TestRun_RenderHTML_PriorityBadge_KindIndexRows(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--body", fixtureGapBody, "--title", "Urgent leak", "--priority", "urgent", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "gap", "--body", fixtureGapBody, "--title", "Untagged leak", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "decision", "--body", fixtureDecisionBody, "--title", "Pick one", "--priority", "medium", "--actor", "human/test", "--root", root)

	out := renderSiteHTML(t, root)

	gapsHTML := testutil.ReadFileT(t, filepath.Join(out, "gaps.html"))
	urgentRow := priorityRowSlice(t, gapsHTML, "G-0001")
	if !strings.Contains(urgentRow, `class="priority priority-urgent"`) || !strings.Contains(urgentRow, ">urgent<") {
		t.Errorf("G-0001 row missing the urgent priority badge:\n%s", urgentRow)
	}
	untaggedRow := priorityRowSlice(t, gapsHTML, "G-0002")
	if strings.Contains(untaggedRow, `class="priority`) {
		t.Errorf("G-0002 (untagged) row must render no priority badge, not an empty one:\n%s", untaggedRow)
	}

	decisionsHTML := testutil.ReadFileT(t, filepath.Join(out, "decisions.html"))
	medRow := priorityRowSlice(t, decisionsHTML, "D-0001")
	if !strings.Contains(medRow, `class="priority priority-medium"`) || !strings.Contains(medRow, ">medium<") {
		t.Errorf("D-0001 row missing the medium priority badge:\n%s", medRow)
	}
}

// TestRun_RenderHTML_PriorityBadge_EntityDetailPage pins the other half
// of M-0264/AC-1: the individual gap/decision detail page's header
// block carries the same badge, scoped structurally to the header (not
// a page-wide substring match) and absent entirely when unset.
func TestRun_RenderHTML_PriorityBadge_EntityDetailPage(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--body", fixtureGapBody, "--title", "Urgent leak", "--priority", "urgent", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "gap", "--body", fixtureGapBody, "--title", "Untagged leak", "--actor", "human/test", "--root", root)

	out := renderSiteHTML(t, root)

	urgentHeader := entityHeaderSlice(t, testutil.ReadFileT(t, filepath.Join(out, "G-0001.html")))
	if !strings.Contains(urgentHeader, `class="priority priority-urgent"`) {
		t.Errorf("G-0001 detail page header missing the urgent priority badge:\n%s", urgentHeader)
	}

	untaggedHeader := entityHeaderSlice(t, testutil.ReadFileT(t, filepath.Join(out, "G-0002.html")))
	if strings.Contains(untaggedHeader, `class="priority`) {
		t.Errorf("G-0002 (untagged) detail page must render no priority badge, not an empty one:\n%s", untaggedHeader)
	}
}

// TestRun_RenderHTML_PriorityBadge_NonCarryingKind_NeverRendersBadge
// pins the "no separate kind-gate needed" mechanism already established
// for the list/status filter (M-0263): a kind that never carries a
// priority (e.g. epic) has an always-empty Priority field, so its
// kind-index row renders no badge — the same absent-value path an
// untagged gap/decision takes, with no special-case kind check.
func TestRun_RenderHTML_PriorityBadge_NonCarryingKind_NeverRendersBadge(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Epic one", "--actor", "human/test", "--root", root)

	out := renderSiteHTML(t, root)

	epicsHTML := testutil.ReadFileT(t, filepath.Join(out, "epics.html"))
	row := priorityRowSlice(t, epicsHTML, "E-0001")
	if strings.Contains(row, `class="priority`) {
		t.Errorf("epic row must never render a priority badge:\n%s", row)
	}
}
