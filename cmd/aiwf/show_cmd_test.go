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

// TestRun_ShowArchivedIndicatorJSON — M-0087/AC-5 (JSON side): for an
// archived entity, the JSON envelope's ShowView carries `archived:
// true`; for an active entity, the field is omitted (omitempty)
// rather than emitted as `archived: false`. The terse JSON shape
// keeps the indicator unambiguous for downstream tooling without
// polluting active-entity envelopes.
//
// Structural assertion: the test parses the JSON envelope and reads
// the `archived` field directly — substring-matching on `"archived"`
// in the raw output would not distinguish presence from absence (the
// word may appear in any string field).
func TestRun_ShowArchivedIndicatorJSON(t *testing.T) {
	root := setupCLITestRepo(t)
	mkActiveAndArchivedGaps(t, root)

	// Archived id renders archived: true.
	out := captureStdout(t, func() {
		if rc := run([]string{"show", "--format=json", "--root", root, "G-0099"}); rc != exitOK {
			t.Fatalf("show G-0099 (archived): rc = %d", rc)
		}
	})
	var env struct {
		Result struct {
			ID       string `json:"id"`
			Archived *bool  `json:"archived,omitempty"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("parse JSON for archived: %v\n%s", err, out)
	}
	if env.Result.Archived == nil || !*env.Result.Archived {
		t.Errorf("archived id: Result.archived = %v, want true (G-0099 lives under work/gaps/archive/)", env.Result.Archived)
	}

	// Active id has no archived field in the envelope (omitempty).
	out2 := captureStdout(t, func() {
		if rc := run([]string{"show", "--format=json", "--root", root, "G-0001"}); rc != exitOK {
			t.Fatalf("show G-0001 (active): rc = %d", rc)
		}
	})
	if strings.Contains(string(out2), `"archived":`) {
		t.Errorf("active id: envelope leaks `archived` field; should be omitted via omitempty:\n%s", out2)
	}
}

// TestRun_ShowArchivedIndicatorTextHeader — M-0087/AC-5 (text side):
// the human-readable text output appends ` · archived` to the
// header line (first line of output) for an archived entity, and
// emits the unchanged header for an active entity.
//
// Per CLAUDE.md "Substring assertions are not structural assertions":
// the substring match is scoped to the *first line* of the output,
// not flat over the full text — the marker could appear later in
// e.g. a referenced-by list or a history detail and trivially pass a
// flat search even when it's missing from the header.
func TestRun_ShowArchivedIndicatorTextHeader(t *testing.T) {
	root := setupCLITestRepo(t)
	mkActiveAndArchivedGaps(t, root)

	out := captureStdout(t, func() {
		if rc := run([]string{"show", "--root", root, "G-0099"}); rc != exitOK {
			t.Fatalf("show G-0099 (archived) text: rc = %d", rc)
		}
	})
	header := firstLine(string(out))
	if !strings.Contains(header, "archived") {
		t.Errorf("archived id: header line missing `archived` marker:\nheader: %q\nfull output:\n%s", header, out)
	}

	out2 := captureStdout(t, func() {
		if rc := run([]string{"show", "--root", root, "G-0001"}); rc != exitOK {
			t.Fatalf("show G-0001 (active) text: rc = %d", rc)
		}
	})
	header2 := firstLine(string(out2))
	if strings.Contains(header2, "archived") {
		t.Errorf("active id: header line carries spurious `archived` marker:\nheader: %q", header2)
	}
}

// firstLine returns text up to the first newline (or the entire
// string if no newline). Used by header-line scoped substring
// assertions in AC-5 — see CLAUDE.md "Substring assertions are not
// structural assertions" for why a flat strings.Contains would be
// the weaker assertion here.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// mkActiveAndArchivedGaps populates root with one active and one
// archived gap. Direct on-disk write (rather than `aiwf add gap`)
// is intentional: the verb's check-rule preflight would lint the
// archived G-0099 alongside, and we want a tight test fixture.
// Shared by AC-5 JSON and text tests.
func mkActiveAndArchivedGaps(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "work", "gaps"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "work", "gaps", "G-0001-active.md"), []byte(`---
id: G-0001
title: Active gap
status: open
---
## What's missing

Active gap body.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "work", "gaps", "archive"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "work", "gaps", "archive", "G-0099-archived.md"), []byte(`---
id: G-0099
title: Archived gap
status: addressed
addressed_by:
    - M-0001
---
## What's missing

Archived gap body.
`), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestShowCmd_NoArchivedFlag — M-0087/AC-4: `aiwf show` exposes no
// `--archived` flag. The verb's resolver spans active and archive
// directories implicitly via the loader (M-0084); no flag opt-in is
// required to look up an archived entity. Pins the no-flag invariant
// alongside TestRun_ShowResolvesArchivedID — the pair gives "the
// verb resolves archived ids" and "the verb does not require a flag
// to do so" as separate mechanical assertions.
func TestShowCmd_NoArchivedFlag(t *testing.T) {
	cmd := newShowCmd()
	if cmd.Flags().Lookup("archived") != nil {
		t.Errorf("show has --archived flag; archived ids resolve without flag opt-in per ADR-0004 §\"Display surfaces\"")
	}
}

// TestRun_ShowResolvesArchivedID — M-0084 AC-4: `aiwf show <id>`
// resolves an entity living under <kind>/archive/ identically to one
// in the active dir, without flag opt-in. Drives through the in-process
// dispatcher (`run`) so the seam from Cobra → tree.Load → buildShowView
// → t.ByID is exercised end-to-end on a tree carrying both an active
// and an archived entity.
func TestRun_ShowResolvesArchivedID(t *testing.T) {
	root := setupCLITestRepo(t)

	// Active gap (so the loader has at least one active entity to walk
	// past). Plain on-disk write rather than `aiwf add gap` because
	// the verb's check-rule preflight would lint the archived G-0099
	// next to it; see the M-0084 work-log decision on M-0086 scope.
	if err := os.MkdirAll(filepath.Join(root, "work", "gaps"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "work", "gaps", "G-0001-active.md"), []byte(`---
id: G-0001
title: Active gap
status: open
---
## What's missing

Active gap body.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Archived gap (terminal status, lives under work/gaps/archive/
	// per ADR-0004 storage table). Written directly so the test
	// doesn't depend on the unimplemented `aiwf archive` verb (M-0085).
	if err := os.MkdirAll(filepath.Join(root, "work", "gaps", "archive"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "work", "gaps", "archive", "G-0099-archived.md"), []byte(`---
id: G-0099
title: Archived gap
status: addressed
---
## What's missing

Archived gap body.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		// JSON output exposes the resolved entity's `path` field,
		// which structurally proves the lookup landed on the archived
		// file (not, e.g., a same-id active file that shadowed it).
		// A pure substring assertion on the text-format header would
		// not distinguish those cases.
		if rc := run([]string{"show", "--format=json", "--root", root, "G-0099"}); rc != exitOK {
			t.Fatalf("show G-0099 (archived): rc = %d", rc)
		}
	})
	var env struct {
		Result ShowView `json:"result"`
	}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("parse JSON: %v\n%s", err, out)
	}
	if env.Result.ID != "G-0099" {
		t.Errorf("Result.ID = %q, want %q", env.Result.ID, "G-0099")
	}
	if env.Result.Status != "addressed" {
		t.Errorf("Result.Status = %q, want %q (archived terminal status)", env.Result.Status, "addressed")
	}
	wantPath := filepath.ToSlash(filepath.Join("work", "gaps", "archive", "G-0099-archived.md"))
	if filepath.ToSlash(env.Result.Path) != wantPath {
		t.Errorf("Result.Path = %q, want %q (structural proof that the archived file resolved)", env.Result.Path, wantPath)
	}
}

// TestRun_HistoryAcrossArchiveRename — M-0084/AC-5 and M-0087/AC-9:
// `aiwf history <id>` walks across an archive-rename trivially via
// the existing trailer model. The seam test creates two commits
// trailered with the same aiwf-entity id — one before the file moves
// into <kind>/archive/, one after — and asserts the resulting
// history shows both events.
//
// M-0084/AC-5 first pinned the loader-side prerequisite: tree.Load
// must resolve the post-rename id so ResolveByCurrentOrPriorID can
// find the entity on the chain build. M-0087/AC-9 re-pins the same
// shape specifically with the `aiwf-verb: archive` trailer (the
// M-0085 verb's trailer key) — so a future trailer rename in the
// archive verb fails this test even if the more generic
// path-rename walk would still pass.
//
// Per ADR-0004 §"Display surfaces" and the existing trailer-grep
// implementation in readHistoryChain, the cross-rename walk works
// because git log matches on trailer values, not file paths.
func TestRun_HistoryAcrossArchiveRename(t *testing.T) {
	root := setupCLITestRepo(t)

	// Commit 1: gap in active dir, trailered with aiwf-entity: G-0001.
	// Synthetic on-disk write + a manual commit (rather than `aiwf
	// add gap`) so the test directly controls trailer presence and
	// doesn't drift if the verb's trailer set evolves.
	gapBody := `---
id: G-0001
title: Will be archived
status: open
---
## What's missing

Body.
`
	activePath := filepath.Join(root, "work", "gaps", "G-0001-will-be-archived.md")
	if err := os.MkdirAll(filepath.Dir(activePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(activePath, []byte(gapBody), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := osExec(t, root, "git", "add", "-A"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := osExec(t, root, "git", "commit", "-q", "-m",
		"aiwf add gap G-0001 \"Will be archived\"\n\naiwf-verb: add\naiwf-entity: G-0001\naiwf-actor: human/test\n"); err != nil {
		t.Fatalf("git commit (add): %v", err)
	}

	// Commit 2: git mv the file into archive/, trailered with
	// aiwf-verb: archive (the M-0085 verb's trailer). Status is
	// flipped to addressed via in-place edit. This is what an
	// `aiwf archive --apply` sweep would produce.
	archivePath := filepath.Join(root, "work", "gaps", "archive", "G-0001-will-be-archived.md")
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := osExec(t, root, "git", "mv", activePath, archivePath); err != nil {
		t.Fatalf("git mv: %v", err)
	}
	movedBody := strings.Replace(gapBody, "status: open", "status: addressed", 1)
	if err := os.WriteFile(archivePath, []byte(movedBody), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := osExec(t, root, "git", "add", "-A"); err != nil {
		t.Fatalf("git add archive: %v", err)
	}
	if err := osExec(t, root, "git", "commit", "-q", "-m",
		"aiwf archive G-0001\n\naiwf-verb: archive\naiwf-entity: G-0001\naiwf-actor: human/test\n"); err != nil {
		t.Fatalf("git commit (archive): %v", err)
	}

	// JSON output exposes the events array structurally — substring
	// matches on text format would not distinguish "both events
	// present" from "one event present twice."
	out := captureStdout(t, func() {
		if rc := run([]string{"history", "--format=json", "--root", root, "G-0001"}); rc != exitOK {
			t.Fatalf("history G-0001: rc = %d", rc)
		}
	})
	var env struct {
		Result struct {
			Events []HistoryEvent `json:"events"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("parse JSON: %v\n%s", err, out)
	}
	if len(env.Result.Events) != 2 {
		t.Fatalf("history events count = %d, want 2 (pre-archive + archive); raw:\n%s", len(env.Result.Events), out)
	}
	// Verbs should be in chronological order: add first, archive
	// second. Pinning the verb sequence pins the cross-rename walk.
	if env.Result.Events[0].Verb != "add" {
		t.Errorf("Events[0].Verb = %q, want %q", env.Result.Events[0].Verb, "add")
	}
	if env.Result.Events[1].Verb != "archive" {
		t.Errorf("Events[1].Verb = %q, want %q", env.Result.Events[1].Verb, "archive")
	}
}

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
