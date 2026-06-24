package integration

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// areaGroupSlice returns the inner HTML of the `area-group` container
// whose data-area attribute equals area — from its opening tag up to the
// next area-group container (or end of document). Scoping the assertion to
// the container's bounds makes "epic X is INSIDE area Y" a structural
// (containment) check, not a flat substring match — the DOM-structural
// shape M-0175/AC-4 (and CLAUDE.md "substring assertions are not
// structural assertions") requires. area-group sections are siblings, so
// the next-open-tag boundary is exact.
func areaGroupSlice(t *testing.T, html, area string) string {
	t.Helper()
	open := `<section class="area-group" data-area="` + area + `">`
	i := strings.Index(html, open)
	if i < 0 {
		t.Fatalf("area-group data-area=%q not found in status page:\n%s", area, html)
	}
	rest := html[i+len(open):]
	// Bound the slice at the next area-group sibling OR the next section
	// heading (<h2>), whichever comes first — so the last (complement)
	// slice, which has no following area-group, doesn't sweep in the Open
	// decisions / gaps sections below the in-flight area.
	end := len(rest)
	for _, marker := range []string{`<section class="area-group"`, "<h2>"} {
		if j := strings.Index(rest, marker); j >= 0 && j < end {
			end = j
		}
	}
	return rest[:end]
}

// renderStatusHTML renders the site for root and returns status.html.
func renderStatusHTML(t *testing.T, root string) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "site")
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out)
	return testutil.ReadFileT(t, filepath.Join(out, "status.html"))
}

// TestRun_RenderHTML_GroupsByArea pins M-0175/AC-4: the status HTML page
// groups in-flight epics into per-area `area-group` containers (declared
// areas + the untagged complement), asserted structurally — each epic
// lives INSIDE its area's container and not another's.
func TestRun_RenderHTML_GroupsByArea(t *testing.T) {
	root := setupAreaRepo(t) // areas: platform, billing (no default → fallback label)
	mustRun(t, "add", "epic", "--title", "Platform work", "--area", "platform", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "epic", "--title", "Billing work", "--area", "billing", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "epic", "--title", "Untagged work", "--actor", "human/test", "--root", root)
	for _, id := range []string{"E-0001", "E-0002", "E-0003"} {
		mustRun(t, "promote", id, "active", "--actor", "human/test", "--root", root)
	}

	html := renderStatusHTML(t, root)

	platform := areaGroupSlice(t, html, "platform")
	if !strings.Contains(platform, "E-0001.html") {
		t.Errorf("platform area-group should contain E-0001:\n%s", platform)
	}
	if strings.Contains(platform, "E-0002.html") {
		t.Errorf("platform area-group must NOT contain billing E-0002:\n%s", platform)
	}

	billing := areaGroupSlice(t, html, "billing")
	if !strings.Contains(billing, "E-0002.html") {
		t.Errorf("billing area-group should contain E-0002:\n%s", billing)
	}
	if strings.Contains(billing, "E-0001.html") {
		t.Errorf("billing area-group must NOT contain platform E-0001:\n%s", billing)
	}

	// Untagged complement: data-area="" container, fallback label, holds
	// E-0003.
	complement := areaGroupSlice(t, html, "")
	if !strings.Contains(complement, "E-0003.html") {
		t.Errorf("complement area-group should contain untagged E-0003:\n%s", complement)
	}
	if !strings.Contains(complement, "Uncategorized") {
		t.Errorf("complement should carry the fallback label 'Uncategorized':\n%s", complement)
	}
}

// TestRun_RenderHTML_FlatWithoutAreas pins M-0175/AC-6 (html): with no
// areas block, the status page renders flat — no area-group containers —
// and the epics still appear as status-epic sections (today's DOM).
func TestRun_RenderHTML_FlatWithoutAreas(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root)
	mustRun(t, "promote", "E-0001", "active", "--actor", "human/test", "--root", root)

	html := renderStatusHTML(t, root)
	if strings.Contains(html, "area-group") {
		t.Errorf("no areas block → status page must carry no area-group containers:\n%s", html)
	}
	if !strings.Contains(html, `class="status-epic"`) || !strings.Contains(html, "E-0001.html") {
		t.Errorf("flat status page should still render the epic section:\n%s", html)
	}
}
