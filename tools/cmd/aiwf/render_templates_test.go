package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRender_IndexShowsACRollup: a fixture with one epic carrying
// one milestone with two ACs (one met, one open) renders the
// `1/2` rollup on the index page. Pins the I3 plan §3 "AC met-
// rollup per epic" requirement through the dispatcher seam.
func TestRender_IndexShowsACRollup(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test")
	mustRun(t, "add", "epic", "--title", "F", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-01", "--title", "M", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "ac", "--root", root, "--actor", "human/test", "M-001", "--title", "A1")
	mustRun(t, "add", "ac", "--root", root, "--actor", "human/test", "M-001", "--title", "A2")
	mustRun(t, "promote", "--root", root, "--actor", "human/test", "M-001/AC-1", "met")

	out := filepath.Join(t.TempDir(), "site")
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out)

	indexHTML := readFileT(t, filepath.Join(out, "index.html"))
	if !strings.Contains(indexHTML, "1/2") {
		t.Errorf("index.html missing AC rollup '1/2':\n%s", indexHTML)
	}
}

// TestRender_MilestoneEmitsSixTabs: every milestone page carries
// the six tabs (Overview, Manifest, Build, Tests, Commits,
// Provenance). The :target-driven show/hide is a CSS concern; the
// HTML must declare every section regardless of content so the
// nav links resolve.
func TestRender_MilestoneEmitsSixTabs(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test")
	mustRun(t, "add", "epic", "--title", "F", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-01", "--title", "M", "--actor", "human/test", "--root", root)

	out := filepath.Join(t.TempDir(), "site")
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out)

	mHTML := readFileT(t, filepath.Join(out, "M-001.html"))
	for _, want := range []string{
		`id="tab-overview"`,
		`id="tab-manifest"`,
		`id="tab-build"`,
		`id="tab-tests"`,
		`id="tab-commits"`,
		`id="tab-provenance"`,
		`href="#tab-build"`,
		`href="#tab-tests"`,
	} {
		if !strings.Contains(mHTML, want) {
			t.Errorf("M-001.html missing %q\n", want)
		}
	}
}

// TestRender_TestsTabPolicyBadge_Advisory: with require_test_metrics
// off, the Tests tab shows the "advisory" badge.
func TestRender_TestsTabPolicyBadge_Advisory(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test")
	mustRun(t, "add", "epic", "--title", "F", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-01", "--title", "M", "--actor", "human/test", "--root", root)

	out := filepath.Join(t.TempDir(), "site")
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out)

	mHTML := readFileT(t, filepath.Join(out, "M-001.html"))
	if !strings.Contains(mHTML, `policy-advisory">advisory`) {
		t.Errorf("expected policy-advisory badge in Tests tab:\n%s", mHTML)
	}
}

// TestRender_TestsTabPolicyBadge_Strict: with require_test_metrics
// on, the badge flips to "strict".
func TestRender_TestsTabPolicyBadge_Strict(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test")
	yamlPath := filepath.Join(root, "aiwf.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	patched := string(raw) + "tdd:\n  require_test_metrics: true\n"
	if err := os.WriteFile(yamlPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
	mustRun(t, "add", "epic", "--title", "F", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-01", "--title", "M", "--actor", "human/test", "--root", root)

	out := filepath.Join(t.TempDir(), "site")
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out)

	mHTML := readFileT(t, filepath.Join(out, "M-001.html"))
	if !strings.Contains(mHTML, `policy-strict">strict`) {
		t.Errorf("expected policy-strict badge in Tests tab:\n%s", mHTML)
	}
}

// TestRender_EpicBodyGoalRendered: when the epic file carries a
// populated `## Goal` section, the rendered page includes a Goal
// block. Pins the body-section parser (step 1) end-to-end through
// the templates.
func TestRender_EpicBodyGoalRendered(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test")
	mustRun(t, "add", "epic", "--title", "F", "--actor", "human/test", "--root", root)
	ePath := filepath.Join(root, "work", "epics", "E-01-f", "epic.md")
	raw, err := os.ReadFile(ePath)
	if err != nil {
		t.Fatalf("read epic: %v", err)
	}
	patched := strings.Replace(string(raw),
		"\n## Goal\n\n## Scope",
		"\n## Goal\n\nbuild the kernel\n\n## Scope",
		1)
	if err := os.WriteFile(ePath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write epic: %v", err)
	}

	out := filepath.Join(t.TempDir(), "site")
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out)

	html := readFileT(t, filepath.Join(out, "E-01.html"))
	if !strings.Contains(html, "build the kernel") {
		t.Errorf("E-01.html missing goal body 'build the kernel':\n%s", html)
	}
}

// TestRender_ACAnchorWiredManifestToBuild: every AC inside the
// Manifest tab carries an id="ac-N"; the Build tab links to it.
// Pins the cross-tab anchor convention (I3 plan §3.3) — without
// it, "click on AC in build tab to see manifest" doesn't work.
func TestRender_ACAnchorWiredManifestToBuild(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test")
	mustRun(t, "add", "epic", "--title", "F", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-01", "--title", "M", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "ac", "--root", root, "--actor", "human/test", "M-001", "--title", "Engine")

	out := filepath.Join(t.TempDir(), "site")
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out)

	mHTML := readFileT(t, filepath.Join(out, "M-001.html"))
	if !strings.Contains(mHTML, `id="ac-1"`) {
		t.Errorf("Manifest tab missing AC anchor id=\"ac-1\":\n%s", mHTML)
	}
	if !strings.Contains(mHTML, `href="#ac-1"`) {
		t.Errorf("Build tab missing cross-link href=\"#ac-1\":\n%s", mHTML)
	}
}
