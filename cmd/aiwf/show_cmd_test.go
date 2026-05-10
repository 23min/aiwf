package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/tree"
)

// TestRun_ShowMilestoneAggregatesACsHistoryFindings exercises the
// full top-level path: a milestone with two ACs, a TDD phase walk,
// and a status promotion. The text output must contain the header,
// both AC rows, the recent-history block, and the no-findings line.
//
// The epic + milestone + ACs are created with `--body-file` so each
// load-bearing section carries placeholder prose. Without this, the
// M-066 `entity-body-empty` rule (and analogous AC body-empty path)
// would fire on the freshly-scaffolded entities and the
// "Findings: (none)" assertion would fail.
func TestRun_ShowMilestoneAggregatesACsHistoryFindings(t *testing.T) {
	root := setupCLITestRepo(t)
	bodyDir := t.TempDir()
	epicBody := filepath.Join(bodyDir, "epic-body.md")
	if err := os.WriteFile(epicBody, []byte("## Goal\n\nFoundations.\n\n## Scope\n\nEngine warning.\n\n## Out of scope\n\nEverything else.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mBody := filepath.Join(bodyDir, "ms-body.md")
	if err := os.WriteFile(mBody, []byte("## Goal\n\nWarn loudly.\n\n## Approach\n\nIterate on each AC.\n\n## Acceptance criteria\n\nEach AC pins one observable behavior.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	acBody1 := filepath.Join(bodyDir, "ac1-body.md")
	if err := os.WriteFile(acBody1, []byte("AC-1 prose under the heading.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	acBody2 := filepath.Join(bodyDir, "ac2-body.md")
	if err := os.WriteFile(acBody2, []byte("AC-2 prose under the heading.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foundations", "--body-file", epicBody, "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "Engine warning", "--body-file", mBody, "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add milestone: %d", rc)
	}
	if rc := run([]string{"add", "ac", "--actor", "human/test", "--root", root, "M-0001", "--title", "AC one", "--body-file", acBody1}); rc != exitOK {
		t.Fatalf("add ac 1: %d", rc)
	}
	if rc := run([]string{"add", "ac", "--actor", "human/test", "--root", root, "M-0001", "--title", "AC two", "--body-file", acBody2}); rc != exitOK {
		t.Fatalf("add ac 2: %d", rc)
	}
	if rc := run([]string{"promote", "--actor", "human/test", "--root", root, "M-0001/AC-1", "met"}); rc != exitOK {
		t.Fatalf("promote: %d", rc)
	}

	out := captureStdout(t, func() {
		if rc := run([]string{"show", "--root", root, "M-0001"}); rc != exitOK {
			t.Fatalf("show: %d", rc)
		}
	})
	s := string(out)
	for _, want := range []string{
		"M-0001 · Engine warning · status: draft",
		"parent: E-0001",
		"ACs:",
		"AC-1 [met]",
		"AC-2 [open]",
		"Recent history",
		"Findings: (none)",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("show output missing %q in:\n%s", want, s)
		}
	}
}

// TestRun_ShowCompositeIDRendersACSlice: querying a composite id
// renders just that AC plus its history, with the AC's parent
// milestone shown as "parent: M-NNN".
func TestRun_ShowCompositeIDRendersACSlice(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "First", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add milestone: %d", rc)
	}
	if rc := run([]string{"add", "ac", "--actor", "human/test", "--root", root, "M-0001", "--title", "Just one"}); rc != exitOK {
		t.Fatalf("add ac: %d", rc)
	}
	if rc := run([]string{"promote", "--actor", "human/test", "--root", root, "M-0001/AC-1", "--phase", "red"}); rc != exitOK {
		t.Fatalf("promote phase: %d", rc)
	}

	out := captureStdout(t, func() {
		if rc := run([]string{"show", "--root", root, "M-0001/AC-1"}); rc != exitOK {
			t.Fatalf("show: %d", rc)
		}
	})
	s := string(out)
	for _, want := range []string{
		"M-0001/AC-1",
		`"Just one"`,
		"status: open",
		"phase: red",
		"parent: M-0001",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("composite show output missing %q in:\n%s", want, s)
		}
	}
}

// TestRun_ShowJSONEnvelope confirms --format=json emits a structured
// envelope with the right shape.
func TestRun_ShowJSONEnvelope(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic: %d", rc)
	}

	captured := captureStdout(t, func() {
		if rc := run([]string{"show", "--root", root, "--format=json", "E-0001"}); rc != exitOK {
			t.Fatalf("show: %d", rc)
		}
	})
	var env struct {
		Tool   string `json:"tool"`
		Status string `json:"status"`
		Result struct {
			ID     string `json:"id"`
			Kind   string `json:"kind"`
			Status string `json:"status"`
			Title  string `json:"title"`
		} `json:"result"`
	}
	if err := json.Unmarshal(captured, &env); err != nil {
		t.Fatalf("parse JSON: %v\n%s", err, captured)
	}
	if env.Tool != "aiwf" || env.Status != "ok" {
		t.Errorf("envelope tool/status = %q/%q", env.Tool, env.Status)
	}
	if env.Result.ID != "E-0001" || env.Result.Kind != "epic" {
		t.Errorf("result.id/kind = %q/%q", env.Result.ID, env.Result.Kind)
	}
}

// TestRun_ShowUnknownIDIsUsageError surfaces a clean error and
// usage exit code.
func TestRun_ShowUnknownIDIsUsageError(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"show", "--root", root, "E-0099"}); rc != exitUsage {
		t.Errorf("expected exitUsage, got %d", rc)
	}
}

// TestRun_ShowReferencedByPopulated: an entity referenced by others
// surfaces them in ShowView.ReferencedBy and in the text "Referenced by"
// block. Inversion follows entity.ForwardRefs; composite-id rollup is
// covered by tree.TestLoad_ReverseRefs.
func TestRun_ShowReferencedByPopulated(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "First", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add milestone: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "Second", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add milestone 2: %d", rc)
	}

	// Text path: showing E-01 must surface both milestones in the
	// "Referenced by" block.
	out := captureStdout(t, func() {
		if rc := run([]string{"show", "--root", root, "E-0001"}); rc != exitOK {
			t.Fatalf("show: %d", rc)
		}
	})
	s := string(out)
	for _, want := range []string{
		"Referenced by (2):",
		"M-0001",
		"M-0002",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("show output missing %q in:\n%s", want, s)
		}
	}

	// JSON path: result.referenced_by is the sorted referrer list.
	captured := captureStdout(t, func() {
		if rc := run([]string{"show", "--root", root, "--format=json", "E-0001"}); rc != exitOK {
			t.Fatalf("show json: %d", rc)
		}
	})
	var env struct {
		Result struct {
			ReferencedBy []string `json:"referenced_by"`
		} `json:"result"`
	}
	if err := json.Unmarshal(captured, &env); err != nil {
		t.Fatalf("parse JSON: %v\n%s", err, captured)
	}
	want := []string{"M-0001", "M-0002"}
	if len(env.Result.ReferencedBy) != len(want) {
		t.Fatalf("referenced_by = %v, want %v", env.Result.ReferencedBy, want)
	}
	for i := range want {
		if env.Result.ReferencedBy[i] != want[i] {
			t.Errorf("referenced_by[%d] = %q, want %q", i, env.Result.ReferencedBy[i], want[i])
		}
	}
}

// TestRun_ShowReferencedByEmptyIsPresent: an unreferenced entity must
// still emit `referenced_by: []` in JSON (not absent, not null) so
// downstream consumers don't have to check for field presence.
func TestRun_ShowReferencedByEmptyIsPresent(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Lonely", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic: %d", rc)
	}

	captured := captureStdout(t, func() {
		if rc := run([]string{"show", "--root", root, "--format=json", "E-0001"}); rc != exitOK {
			t.Fatalf("show json: %d", rc)
		}
	})
	if !strings.Contains(string(captured), `"referenced_by":[]`) {
		t.Errorf("expected referenced_by:[] in JSON; got:\n%s", captured)
	}
}

// TestRun_ShowFindingsScopedToEntity: when the entity has a real
// finding, show surfaces it. The standing check
// `milestone-done-incomplete-acs` catches the inconsistent state on
// every check pass — even when the file landed via a hand-edit
// rather than the verb path. This is the load-bearing reason that
// finding runs at check time, not just at verb-projection time.
//
// We can't get a milestone into status: done with an open AC via the
// verb path (the projection check that becomes the standing finding
// also blocks the verb), so the test hand-edits the file on disk
// and commits — exactly the scenario the standing check exists to
// catch.
func TestRun_ShowFindingsScopedToEntity(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "Done milestone", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add milestone: %d", rc)
	}
	if rc := run([]string{"add", "ac", "--actor", "human/test", "--root", root, "M-0001", "--title", "Open AC"}); rc != exitOK {
		t.Fatalf("add ac: %d", rc)
	}

	// Hand-edit the milestone to status: done while AC-1 is still
	// open — the inconsistent state the standing check exists to
	// catch. The verb path would refuse this; that's the point.
	mPath := filepath.Join(root, "work", "epics", "E-0001-foo", "M-0001-done-milestone.md")
	raw, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	patched := strings.Replace(string(raw), "status: draft", "status: done", 1)
	if writeErr := os.WriteFile(mPath, []byte(patched), 0o644); writeErr != nil {
		t.Fatalf("write patched: %v", writeErr)
	}

	tr, _, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	view, ok := buildShowView(context.Background(), root, tr, nil, "M-0001", 5)
	if !ok {
		t.Fatal("show view missing")
	}
	if len(view.Findings) == 0 {
		t.Fatal("expected milestone-done-incomplete-acs finding")
	}
	found := false
	for _, f := range view.Findings {
		if f.Code == "milestone-done-incomplete-acs" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected milestone-done-incomplete-acs in findings; got %+v", view.Findings)
	}
}

// TestRun_ShowEpicBodySectionsParsed: hand-editing the epic file with
// populated body sections must surface them on ShowView.Body keyed by
// the slugified `## ` heading. Load-bearing for the HTML render in I3
// step 5, which reads these slugs to populate the per-tab content.
func TestRun_ShowEpicBodySectionsParsed(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic: %d", rc)
	}

	// Hand-populate the body — the scaffolded body has empty sections.
	ePath := filepath.Join(root, "work", "epics", "E-0001-foundations", "epic.md")
	raw, err := os.ReadFile(ePath)
	if err != nil {
		t.Fatalf("read epic: %v", err)
	}
	populated := strings.Replace(string(raw),
		"\n## Goal\n\n## Scope\n\n## Out of scope\n",
		"\n## Goal\n\nbuild the kernel\n\n## Scope\n\nplanning verbs\n\n## Out of scope\n\nthe UI\n",
		1)
	if writeErr := os.WriteFile(ePath, []byte(populated), 0o644); writeErr != nil {
		t.Fatalf("write epic: %v", writeErr)
	}

	captured := captureStdout(t, func() {
		if rc := run([]string{"show", "--root", root, "--format=json", "E-0001"}); rc != exitOK {
			t.Fatalf("show json: %d", rc)
		}
	})
	var env struct {
		Result struct {
			Body map[string]string `json:"body"`
		} `json:"result"`
	}
	if err := json.Unmarshal(captured, &env); err != nil {
		t.Fatalf("parse JSON: %v\n%s", err, captured)
	}
	want := map[string]string{
		"goal":         "build the kernel",
		"scope":        "planning verbs",
		"out_of_scope": "the UI",
	}
	for k, v := range want {
		if env.Result.Body[k] != v {
			t.Errorf("body[%q] = %q, want %q (full: %v)", k, env.Result.Body[k], v, env.Result.Body)
		}
	}
}

// TestRun_ShowMilestoneACDescriptionsParsed: the per-AC body section
// (`### AC-N — <title>`) populates ShowAC.Description when the
// milestone body carries it. Load-bearing for the milestone Manifest
// tab in I3 step 5, which renders each AC's body inline.
func TestRun_ShowMilestoneACDescriptionsParsed(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "First", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add milestone: %d", rc)
	}
	if rc := run([]string{"add", "ac", "--actor", "human/test", "--root", root, "M-0001", "--title", "Engine starts"}); rc != exitOK {
		t.Fatalf("add ac: %d", rc)
	}

	// `aiwf add ac` scaffolds a `### AC-1 — <title>` heading with no
	// body underneath; populate it with prose by hand-editing.
	mPath := filepath.Join(root, "work", "epics", "E-0001-foo", "M-0001-first.md")
	raw, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	body := string(raw)
	const heading = "### AC-1 — Engine starts"
	idx := strings.Index(body, heading)
	if idx < 0 {
		t.Fatalf("missing scaffolded AC heading; body:\n%s", body)
	}
	insert := idx + len(heading)
	patched := body[:insert] + "\n\nthe engine MUST start within 3 seconds.\n" + body[insert:]
	if writeErr := os.WriteFile(mPath, []byte(patched), 0o644); writeErr != nil {
		t.Fatalf("write milestone: %v", writeErr)
	}

	captured := captureStdout(t, func() {
		if rc := run([]string{"show", "--root", root, "--format=json", "M-0001"}); rc != exitOK {
			t.Fatalf("show: %d", rc)
		}
	})
	var env struct {
		Result struct {
			ACs []struct {
				ID          string `json:"id"`
				Description string `json:"description"`
			} `json:"acs"`
		} `json:"result"`
	}
	if err := json.Unmarshal(captured, &env); err != nil {
		t.Fatalf("parse JSON: %v\n%s", err, captured)
	}
	if len(env.Result.ACs) != 1 || env.Result.ACs[0].ID != "AC-1" {
		t.Fatalf("unexpected ACs: %+v", env.Result.ACs)
	}
	if want := "the engine MUST start within 3 seconds."; env.Result.ACs[0].Description != want {
		t.Errorf("AC-1 description = %q, want %q", env.Result.ACs[0].Description, want)
	}

	// Composite-id show should also surface description on the AC
	// payload.
	captured = captureStdout(t, func() {
		if rc := run([]string{"show", "--root", root, "--format=json", "M-0001/AC-1"}); rc != exitOK {
			t.Fatalf("composite show: %d", rc)
		}
	})
	var env2 struct {
		Result struct {
			AC struct {
				ID          string `json:"id"`
				Description string `json:"description"`
			} `json:"ac"`
		} `json:"result"`
	}
	if err := json.Unmarshal(captured, &env2); err != nil {
		t.Fatalf("parse JSON (composite): %v\n%s", err, captured)
	}
	if env2.Result.AC.ID != "AC-1" {
		t.Fatalf("composite AC id = %q, want AC-1", env2.Result.AC.ID)
	}
	if want := "the engine MUST start within 3 seconds."; env2.Result.AC.Description != want {
		t.Errorf("composite AC description = %q, want %q", env2.Result.AC.Description, want)
	}
}

// TestRun_ShowHistoryParsesAiwfTestsTrailer: a commit carrying an
// aiwf-tests trailer must surface the parsed metrics on the
// HistoryEvent. The trailer is written by hand here (kernel write
// path lands in I3 step 2); read-side parsing must already work in
// step 1 so step-5 templates have data to render.
func TestRun_ShowHistoryParsesAiwfTestsTrailer(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "First", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add milestone: %d", rc)
	}
	if rc := run([]string{"add", "ac", "--actor", "human/test", "--root", root, "M-0001", "--title", "Engine starts"}); rc != exitOK {
		t.Fatalf("add ac: %d", rc)
	}

	// Hand-author an empty commit on M-001/AC-1 carrying aiwf-tests.
	// `aiwf history` reads the trailer via git log %(trailers:...);
	// what matters is that the trailer line is present on the commit.
	const subject = "promote(M-001/AC-1) green with metrics"
	const body = "aiwf-verb: promote\naiwf-entity: M-001/AC-1\naiwf-actor: human/test\naiwf-to: green\naiwf-tests: pass=12 fail=0 skip=1\n"
	if err := osExec(t, root, "git", "commit", "--allow-empty",
		"-m", subject, "-m", body); err != nil {
		t.Fatalf("git commit --allow-empty: %v", err)
	}

	captured := captureStdout(t, func() {
		if rc := run([]string{"show", "--root", root, "--format=json", "M-0001/AC-1"}); rc != exitOK {
			t.Fatalf("show: %d", rc)
		}
	})
	var env struct {
		Result struct {
			History []struct {
				Verb  string `json:"verb"`
				Tests *struct {
					Pass int `json:"pass"`
					Fail int `json:"fail"`
					Skip int `json:"skip"`
				} `json:"tests"`
			} `json:"history"`
		} `json:"result"`
	}
	if err := json.Unmarshal(captured, &env); err != nil {
		t.Fatalf("parse JSON: %v\n%s", err, captured)
	}
	var withMetrics int
	for _, e := range env.Result.History {
		if e.Tests == nil {
			continue
		}
		withMetrics++
		if e.Tests.Pass != 12 || e.Tests.Fail != 0 || e.Tests.Skip != 1 {
			t.Errorf("history tests = %+v, want pass=12 fail=0 skip=1", e.Tests)
		}
	}
	if withMetrics != 1 {
		t.Errorf("expected exactly one history event with tests metrics; got %d (history: %+v)", withMetrics, env.Result.History)
	}
}
