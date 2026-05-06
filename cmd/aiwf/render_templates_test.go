package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestRender_IndexShowsACRollup: a fixture with one epic carrying
// one milestone with two ACs (one met, one open) renders the
// `1/2` rollup on the index page. Pins the I3 plan §3 "AC met-
// rollup per epic" requirement through the dispatcher seam.
func TestRender_IndexShowsACRollup(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
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
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
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
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
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
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
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
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
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

// TestRender_ACAnchorWiredManifestToBuild: AC anchors land in the
// Manifest tab (id="ac-N"), and the Build tab cross-links to them
// via href="#ac-N". Pins the cross-tab anchor convention (I3 plan
// §3.3). Asserts via htmlSection — the loose substring version
// of this test (which would pass even with the anchor in the
// wrong tab) is what the testing-rules audit caught.
func TestRender_ACAnchorWiredManifestToBuild(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "F", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-01", "--title", "M", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "ac", "--root", root, "--actor", "human/test", "M-001", "--title", "Engine")

	out := filepath.Join(t.TempDir(), "site")
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out)

	mHTML := readFileT(t, filepath.Join(out, "M-001.html"))
	assertContainsIn(t, mHTML, "manifest", `id="ac-1"`, "AC anchor in Manifest tab")
	assertContainsIn(t, mHTML, "build", `href="#ac-1"`, "Build tab cross-link to AC")
	// Inverse: the AC anchor must NOT appear in the Build tab
	// (Build links TO it, doesn't host it).
	assertNotContainsIn(t, mHTML, "build", `id="ac-1"`, "AC anchor must live in Manifest, not Build")
}

// TestRender_TestsTabBadgeIsInsideTestsTab: the strict|advisory
// badge must appear inside the Tests <section>, not in the
// Build tab or anywhere else. The plain substring version of
// this assertion (`policy-strict">strict`) was the first example
// the testing-rules audit flagged as "passes for wrong reasons."
func TestRender_TestsTabBadgeIsInsideTestsTab(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "F", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-01", "--title", "M", "--actor", "human/test", "--root", root)

	out := filepath.Join(t.TempDir(), "site")
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out)

	mHTML := readFileT(t, filepath.Join(out, "M-001.html"))
	assertContainsIn(t, mHTML, "tests", `policy-advisory">advisory`, "advisory badge inside Tests tab")
	assertNotContainsIn(t, mHTML, "build", `policy-advisory`, "policy badge must not leak into Build tab")
	assertNotContainsIn(t, mHTML, "manifest", `policy-advisory`, "policy badge must not leak into Manifest tab")
}

// TestRender_BuildTabExcludesStatusEvents: only TDD-phase
// transitions (red/green/refactor/done) belong in the Build tab.
// A status promotion (`open → met`) writes the same `aiwf-to:`
// trailer as a phase promotion, so an unfiltered render shows
// status events as phase rows. Was a real bug surfaced by the
// I3 step-5 smoke render; this test is the regression pin.
func TestRender_BuildTabExcludesStatusEvents(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "F", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-01", "--title", "M", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "ac", "--root", root, "--actor", "human/test", "M-001", "--title", "Engine")
	mustRun(t, "promote", "--root", root, "--actor", "human/test", "M-001/AC-1", "met") // status, not phase

	out := filepath.Join(t.TempDir(), "site")
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out)
	mHTML := readFileT(t, filepath.Join(out, "M-001.html"))

	// `met` is a status, not a phase. It must NOT appear as a
	// phase event in the Build tab.
	assertNotContainsIn(t, mHTML, "build", `phase phase-met`, "status promotion leaked into Build tab as phase row")
	// Build tab should show "no phase events" instead.
	assertContainsIn(t, mHTML, "build", "No phase events recorded", "Build tab should report empty state for status-only AC")
}

// TestRender_BuildTabIncludesPhaseHistory: walking an AC through
// red→green→done writes three phase events, all of which must
// appear in the Build tab's timeline. Pins the populated-Build
// branch the previous test set didn't exercise (the smoke fixture
// had only status promotions; no phase history was rendered into
// any test).
func TestRender_BuildTabIncludesPhaseHistory(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "F", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-01", "--title", "M", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "ac", "--root", root, "--actor", "human/test", "M-001", "--title", "Engine")
	mustRun(t, "promote", "--root", root, "--actor", "human/test", "M-001/AC-1", "--phase", "red")
	mustRun(t, "promote", "--root", root, "--actor", "human/test", "M-001/AC-1", "--phase", "green",
		"--tests", "pass=12 fail=0 skip=0")
	mustRun(t, "promote", "--root", root, "--actor", "human/test", "M-001/AC-1", "--phase", "done")

	out := filepath.Join(t.TempDir(), "site")
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out)
	mHTML := readFileT(t, filepath.Join(out, "M-001.html"))

	for _, phase := range []string{"phase-red", "phase-green", "phase-done"} {
		assertContainsIn(t, mHTML, "build", phase, "Build tab missing phase row")
	}
	// The aiwf-tests trailer on green should surface inline.
	assertContainsIn(t, mHTML, "build", "pass=12", "Build tab missing aiwf-tests metrics")
	// And the strict-policy Tests-tab table should pick the green
	// commit's metrics for AC-1 (advisory mode here, but data is
	// still surfaced).
	assertContainsIn(t, mHTML, "tests", "<td>12</td>", "Tests tab missing Pass=12 cell for AC-1")
}

// TestRender_ProvenanceTabShowsAuthorizeScope: opening an
// authorize scope on a milestone surfaces the scope row in the
// Provenance tab's scopes table. Pins the provenanceFor branch
// the previous test set never exercised (no fixture had any
// authorize commits).
func TestRender_ProvenanceTabShowsAuthorizeScope(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "F", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-01", "--title", "M", "--actor", "human/test", "--root", root)
	mustRun(t, "authorize", "--root", root, "--actor", "human/test", "M-001", "--to", "ai/claude")

	out := filepath.Join(t.TempDir(), "site")
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out)
	mHTML := readFileT(t, filepath.Join(out, "M-001.html"))

	assertContainsIn(t, mHTML, "provenance", `scope-state-active`, "Provenance tab missing active scope state")
	assertContainsIn(t, mHTML, "provenance", `ai/claude`, "Provenance tab missing agent")
	assertContainsIn(t, mHTML, "provenance", `human/test`, "Provenance tab missing principal")
	assertContainsIn(t, mHTML, "provenance", `<table class="scopes">`, "Provenance scopes table missing")
	// The scope is reported via a <table>, not the empty-state
	// fallback line.
	assertNotContainsIn(t, mHTML, "provenance", "No authorized scopes", "active scope must override empty-state")
}

// TestRender_OverviewSuppressesEmptyLinkedDecisions: when the
// milestone has no linked decisions, the Overview tab must not
// render a stub "Linked decisions" heading + empty <ul>. Bug
// surfaced by the smoke render.
func TestRender_OverviewSuppressesEmptyLinkedDecisions(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "F", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-01", "--title", "M", "--actor", "human/test", "--root", root)

	out := filepath.Join(t.TempDir(), "site")
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out)
	mHTML := readFileT(t, filepath.Join(out, "M-001.html"))

	assertNotContainsIn(t, mHTML, "overview", "Linked decisions", "empty Linked decisions heading must not render")
}

// TestRender_CommitDatesAreDateOnly: the Commits tab and the
// Provenance Timeline must render dates as YYYY-MM-DD, not as
// full ISO timestamps. Bug surfaced by the smoke render —
// historyEventToRow originally passed the raw Date through.
func TestRender_CommitDatesAreDateOnly(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "F", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-01", "--title", "M", "--actor", "human/test", "--root", root)

	out := filepath.Join(t.TempDir(), "site")
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out)
	mHTML := readFileT(t, filepath.Join(out, "M-001.html"))

	// No section should carry a `T...:` ISO time component in a
	// date cell. The pattern T<digits>:<digits> uniquely identifies
	// the time portion of an ISO 8601 timestamp.
	timePattern := regexp.MustCompile(`>20\d\d-\d\d-\d\dT\d\d:\d\d`)
	if loc := timePattern.FindStringIndex(mHTML); loc != nil {
		t.Errorf("found ISO time component in rendered output (should be date-only):\n%s", mHTML[loc[0]:loc[1]+20])
	}
}
