package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/tools/internal/initrepo"
	"github.com/23min/ai-workflow-v2/tools/internal/version"
)

// parseVersionForTest is a tiny adapter so tests can keep producing
// version.Info values from raw strings without importing the
// version package at every call site.
func parseVersionForTest(raw string) version.Info {
	return version.Parse(raw)
}

// TestRun_InitThroughDispatcher confirms `aiwf init` wires through the
// dispatcher: scaffolds dirs, writes aiwf.yaml, materializes skills,
// installs the pre-push hook.
func TestRun_InitThroughDispatcher(t *testing.T) {
	root := setupCLITestRepo(t)

	rc := run([]string{"init", "--root", root, "--actor", "human/test"})
	if rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if _, err := os.Stat(filepath.Join(root, "aiwf.yaml")); err != nil {
		t.Errorf("aiwf.yaml missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".claude", "skills", "aiwf-add", "SKILL.md")); err != nil {
		t.Errorf("aiwf-add skill missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "pre-push")); err != nil {
		t.Errorf("pre-push hook missing: %v", err)
	}

	// Re-run to confirm idempotency through the dispatcher.
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Errorf("re-run init: %d", rc)
	}
}

// TestRun_InitDryRun confirms `aiwf init --dry-run` reports the
// would-be ledger, prefixes the output with a dry-run banner, and
// writes nothing to disk.
func TestRun_InitDryRun(t *testing.T) {
	root := setupCLITestRepo(t)

	captured := captureStdout(t, func() {
		if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--dry-run"}); rc != exitOK {
			t.Errorf("got rc=%d, want %d", rc, exitOK)
		}
	})
	out := string(captured)

	for _, want := range []string{
		"dry-run",
		"created    aiwf.yaml",
		"created    work/epics",
		"updated    .claude/skills/aiwf-*",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\nfull output:\n%s", want, out)
		}
	}
	// Nothing on disk.
	for _, p := range []string{
		"aiwf.yaml",
		filepath.Join(".claude", "skills", "aiwf-add", "SKILL.md"),
		filepath.Join(".git", "hooks", "pre-push"),
	} {
		if _, err := os.Stat(filepath.Join(root, p)); !os.IsNotExist(err) {
			t.Errorf("dry-run wrote %s (stat err=%v); should be untouched", p, err)
		}
	}
}

// TestRun_InitSkipHook confirms `aiwf init --skip-hook` lands every
// step except the hook installation. Exit is OK (skip is requested,
// not a conflict).
func TestRun_InitSkipHook(t *testing.T) {
	root := setupCLITestRepo(t)

	captured := captureStdout(t, func() {
		if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
			t.Errorf("got rc=%d, want %d", rc, exitOK)
		}
	})
	out := string(captured)

	for _, want := range []string{
		"skipped    .git/hooks/pre-push",
		"--skip-hook",
		"pre-push hook skipped",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\nfull output:\n%s", want, out)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "aiwf.yaml")); err != nil {
		t.Errorf("aiwf.yaml missing after --skip-hook init: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "pre-push")); !os.IsNotExist(err) {
		t.Errorf("hook installed despite --skip-hook (stat err=%v)", err)
	}
}

// TestRun_InitSkipsAlienHook: when a non-aiwf pre-push hook is in
// place, init lands every other step, leaves the alien hook
// untouched, prints both the ledger and the remediation block, and
// exits with `exitFindings` so CI notices.
func TestRun_InitSkipsAlienHook(t *testing.T) {
	root := setupCLITestRepo(t)
	hookDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hookDir, 0o755); err != nil {
		t.Fatal(err)
	}
	alien := []byte("#!/bin/sh\nexit 0\n")
	if err := os.WriteFile(filepath.Join(hookDir, "pre-push"), alien, 0o755); err != nil {
		t.Fatal(err)
	}

	captured := captureStdout(t, func() {
		if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitFindings {
			t.Errorf("got %d, want %d", rc, exitFindings)
		}
	})
	out := string(captured)

	for _, want := range []string{
		"created    aiwf.yaml", // earlier steps still ran
		"created    work/epics",
		"updated    .claude/skills/aiwf-*",
		"skipped    .git/hooks/pre-push",
		"aiwf init: setup landed except the pre-push hook.",
		"aiwf check || exit 1", // remediation option 1
		"husky/lefthook",       // remediation option 2
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\nfull output:\n%s", want, out)
		}
	}

	// Other steps actually landed on disk.
	if _, err := os.Stat(filepath.Join(root, "aiwf.yaml")); err != nil {
		t.Errorf("aiwf.yaml missing after partial init: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".claude", "skills", "aiwf-add", "SKILL.md")); err != nil {
		t.Errorf("aiwf-add skill missing after partial init: %v", err)
	}
	// Alien hook is intact.
	got, _ := os.ReadFile(filepath.Join(hookDir, "pre-push"))
	if !bytes.Equal(got, alien) {
		t.Errorf("alien hook clobbered:\n%s", got)
	}
}

// TestRun_UpdateMaterializes wipes a tampered skill file and verifies
// `aiwf update` restores the embedded content byte-for-byte.
func TestRun_UpdateMaterializes(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	skillPath := filepath.Join(root, ".claude", "skills", "aiwf-add", "SKILL.md")
	if err := os.WriteFile(skillPath, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := run([]string{"update", "--root", root}); rc != exitOK {
		t.Fatalf("update: %d", rc)
	}
	got, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "name: aiwf-add") {
		t.Errorf("aiwf-add not restored: %s", got)
	}
}

// TestRun_UpdateRefreshesPrePushHook removes a previously-installed
// pre-push hook and confirms `aiwf update` reinstalls it. Without
// the broadened update verb (step 5), this would fail because
// update only re-materialised skills.
func TestRun_UpdateRefreshesPrePushHook(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	hookPath := filepath.Join(root, ".git", "hooks", "pre-push")
	if err := os.Remove(hookPath); err != nil {
		t.Fatalf("removing pre-push hook: %v", err)
	}
	if rc := run([]string{"update", "--root", root}); rc != exitOK {
		t.Fatalf("update: %d", rc)
	}
	body, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("pre-push hook missing after update: %v", err)
	}
	if !strings.Contains(string(body), initrepo.HookMarker()) {
		t.Errorf("pre-push hook missing marker after update:\n%s", body)
	}
}

// TestRun_UpdateRefreshesPreCommitHook is the same property for the
// new pre-commit hook (default-on per status_md.auto_update).
func TestRun_UpdateRefreshesPreCommitHook(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	hookPath := filepath.Join(root, ".git", "hooks", "pre-commit")
	if err := os.Remove(hookPath); err != nil {
		t.Fatalf("removing pre-commit hook: %v", err)
	}
	if rc := run([]string{"update", "--root", root}); rc != exitOK {
		t.Fatalf("update: %d", rc)
	}
	body, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("pre-commit hook missing after update: %v", err)
	}
	if !strings.Contains(string(body), initrepo.PreCommitHookMarker()) {
		t.Errorf("pre-commit hook missing marker after update:\n%s", body)
	}
}

// TestRun_UpdateUninstallsPreCommitOnOptOut: the canonical flow —
// run init (hook installed by default), flip status_md.auto_update
// to false in aiwf.yaml, run update → marker-managed pre-commit
// hook is removed.
func TestRun_UpdateUninstallsPreCommitOnOptOut(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	hookPath := filepath.Join(root, ".git", "hooks", "pre-commit")
	if _, err := os.Stat(hookPath); err != nil {
		t.Fatalf("pre-commit hook not installed by default Init: %v", err)
	}

	// Flip the opt-out flag.
	yamlPath := filepath.Join(root, "aiwf.yaml")
	updated := []byte(`aiwf_version: 0.1.0
actor: human/test
status_md:
  auto_update: false
`)
	if err := os.WriteFile(yamlPath, updated, 0o644); err != nil {
		t.Fatalf("rewriting aiwf.yaml: %v", err)
	}

	if rc := run([]string{"update", "--root", root}); rc != exitOK {
		t.Fatalf("update: %d", rc)
	}
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Errorf("pre-commit hook still on disk after opt-out (stat err=%v)", err)
	}
}

// TestRun_UpdateMissingConfig: update against a directory with no
// aiwf.yaml is an internal error (config.Load returns ErrNotFound,
// which `aiwf update` cannot continue past — the StatusMdAutoUpdate
// flag has nowhere to come from). The user is expected to run
// `aiwf init` first.
func TestRun_UpdateMissingConfig(t *testing.T) {
	root := setupCLITestRepo(t)
	// No init: aiwf.yaml is absent.
	if rc := run([]string{"update", "--root", root}); rc != exitInternal {
		t.Errorf("rc = %d, want exitInternal (%d)", rc, exitInternal)
	}
}

// TestRun_HistoryShowsAddPromoteCancel exercises the full chain: init,
// add, promote, cancel, then history. The output should list three
// events for the entity, oldest-first.
func TestRun_HistoryShowsAddPromoteCancel(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add: %d", rc)
	}
	if rc := run([]string{"promote", "--actor", "human/test", "--root", root, "E-01", "active"}); rc != exitOK {
		t.Fatalf("promote: %d", rc)
	}
	if rc := run([]string{"cancel", "--actor", "human/test", "--root", root, "E-01"}); rc != exitOK {
		t.Fatalf("cancel: %d", rc)
	}

	events, err := readHistory(context.Background(), root, "E-01")
	if err != nil {
		t.Fatalf("readHistory: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3:\n%+v", len(events), events)
	}
	wantVerbs := []string{"add", "promote", "cancel"}
	for i, w := range wantVerbs {
		if events[i].Verb != w {
			t.Errorf("[%d] verb %q, want %q", i, events[i].Verb, w)
		}
		if events[i].Actor != "human/test" {
			t.Errorf("[%d] actor %q, want human/test", i, events[i].Actor)
		}
	}
}

// TestRun_HistoryJSON exercises the --format=json path. Capturing
// stdout requires redirecting os.Stdout for the duration of the call;
// we then parse the envelope and assert its shape.
func TestRun_HistoryJSON(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add: %d", rc)
	}
	if rc := run([]string{"promote", "--actor", "human/test", "--root", root, "E-01", "active"}); rc != exitOK {
		t.Fatalf("promote: %d", rc)
	}

	captured := captureStdout(t, func() {
		if rc := run([]string{"history", "--root", root, "--format=json", "E-01"}); rc != exitOK {
			t.Fatalf("history: %d", rc)
		}
	})

	var env struct {
		Tool    string `json:"tool"`
		Status  string `json:"status"`
		Version string `json:"version"`
		Result  struct {
			ID     string         `json:"id"`
			Events []HistoryEvent `json:"events"`
		} `json:"result"`
	}
	if err := json.Unmarshal(captured, &env); err != nil {
		t.Fatalf("parse JSON: %v\n%s", err, captured)
	}
	if env.Tool != "aiwf" {
		t.Errorf("tool = %q", env.Tool)
	}
	if env.Status != "ok" {
		t.Errorf("status = %q", env.Status)
	}
	if env.Result.ID != "E-01" {
		t.Errorf("result.id = %q", env.Result.ID)
	}
	if len(env.Result.Events) != 2 {
		t.Fatalf("events len = %d, want 2:\n%s", len(env.Result.Events), captured)
	}
	if env.Result.Events[0].Verb != "add" || env.Result.Events[1].Verb != "promote" {
		t.Errorf("verbs = [%q,%q], want [add,promote]",
			env.Result.Events[0].Verb, env.Result.Events[1].Verb)
	}
	for i, e := range env.Result.Events {
		if e.Date == "" || e.Actor == "" || e.Commit == "" {
			t.Errorf("event[%d] missing required field: %+v", i, e)
		}
	}
}

// captureStdout replaces os.Stdout with a pipe for the duration of fn
// and returns whatever was written.
func captureStdout(t *testing.T, fn func()) []byte {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	done := make(chan []byte, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.Bytes()
	}()

	fn()
	_ = w.Close()
	return <-done
}

// TestRun_HistoryMilestonePrefixMatchesACs: querying the bare
// milestone id matches every commit whose aiwf-entity is the bare id
// OR M-NNN/AC-N (path-prefix anchored on `/`). The composite-id query
// matches only that AC.
func TestRun_HistoryMilestonePrefixMatchesACs(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--epic", "E-01", "--title", "First", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add milestone: %d", rc)
	}
	if rc := run([]string{"add", "ac", "--actor", "human/test", "--root", root, "M-001", "--title", "AC one"}); rc != exitOK {
		t.Fatalf("add ac: %d", rc)
	}
	if rc := run([]string{"promote", "--actor", "human/test", "--root", root, "M-001/AC-1", "met"}); rc != exitOK {
		t.Fatalf("promote AC: %d", rc)
	}

	// Bare milestone query matches both milestone and AC events.
	events, err := readHistory(context.Background(), root, "M-001")
	if err != nil {
		t.Fatalf("readHistory M-001: %v", err)
	}
	// Expected events for the bare milestone query: 3 total — add of
	// M-001, add of M-001/AC-1, promote of M-001/AC-1.
	if len(events) != 3 {
		t.Errorf("M-001 history len = %d, want 3:\n%+v", len(events), events)
	}

	// Composite query matches only the AC events.
	events, err = readHistory(context.Background(), root, "M-001/AC-1")
	if err != nil {
		t.Fatalf("readHistory M-001/AC-1: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("M-001/AC-1 history len = %d, want 2 (add + promote):\n%+v", len(events), events)
	}
}

// TestRun_HistoryReadsAiwfToAndForce confirms readHistory pulls the
// I2 trailers (`aiwf-to:` and `aiwf-force:`) into HistoryEvent.To and
// .Force, and renders dashes / blanks for events that don't carry
// them. The mix of add (no aiwf-to), promote (with aiwf-to), and
// promote --force (with aiwf-to AND aiwf-force) covers the load-
// bearing field-projection paths.
func TestRun_HistoryReadsAiwfToAndForce(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add: %d", rc)
	}
	if rc := run([]string{"promote", "--actor", "human/test", "--root", root, "E-01", "active"}); rc != exitOK {
		t.Fatalf("promote 1: %d", rc)
	}
	// Force-jump from active straight to cancelled — illegal for epics
	// (active→cancelled is legal so let's pick proposed→done... but
	// E-01 is now active). Force the FSM-illegal active→done jump
	// using a different epic to keep this test focused.
	if rc := run([]string{"add", "epic", "--title", "Bar", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add 2: %d", rc)
	}
	if rc := run([]string{"promote", "--actor", "human/test", "--root", root, "E-02", "done", "--force", "--reason", "sandbox emergency"}); rc != exitOK {
		t.Fatalf("forced promote: %d", rc)
	}

	// E-01: add (no to/force), promote → active (to=active, no force).
	events, err := readHistory(context.Background(), root, "E-01")
	if err != nil {
		t.Fatalf("readHistory E-01: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("E-01 got %d events, want 2:\n%+v", len(events), events)
	}
	if events[0].To != "" || events[0].Force != "" {
		t.Errorf("E-01 add event should have empty To/Force, got %+v", events[0])
	}
	if events[1].To != "active" {
		t.Errorf("E-01 promote To = %q, want active", events[1].To)
	}
	if events[1].Force != "" {
		t.Errorf("E-01 promote Force = %q, want empty", events[1].Force)
	}

	// E-02: add (no to/force), forced promote → done (to=done, force=reason).
	events, err = readHistory(context.Background(), root, "E-02")
	if err != nil {
		t.Fatalf("readHistory E-02: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("E-02 got %d events, want 2:\n%+v", len(events), events)
	}
	if events[1].To != "done" {
		t.Errorf("E-02 forced promote To = %q, want done", events[1].To)
	}
	if events[1].Force != "sandbox emergency" {
		t.Errorf("E-02 forced promote Force = %q, want %q", events[1].Force, "sandbox emergency")
	}
}

// TestRun_HistoryRenderToDash confirms the text renderer produces a
// dash (`-`) for events without an `aiwf-to:` trailer, and `→ <to>`
// when one is present. This is the load-bearing backwards-compat
// rendering for pre-I2 commits.
func TestRun_HistoryRenderToDash(t *testing.T) {
	tests := []struct {
		to   string
		want string
	}{
		{"", "-"},
		{"active", "→ active"},
		{"in_progress", "→ in_progress"},
		{"green", "→ green"},
	}
	for _, tt := range tests {
		t.Run(tt.to, func(t *testing.T) {
			if got := renderTo(tt.to); got != tt.want {
				t.Errorf("renderTo(%q) = %q, want %q", tt.to, got, tt.want)
			}
		})
	}
}

// TestRun_HistoryTextOutputIncludesForceLine: when an event has a
// force reason, the renderer emits a `[forced: <reason>]` line below
// the main row. Pinned via captured stdout.
func TestRun_HistoryTextOutputIncludesForceLine(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add: %d", rc)
	}
	if rc := run([]string{"promote", "--actor", "human/test", "--root", root, "E-01", "done", "--force", "--reason", "policy override"}); rc != exitOK {
		t.Fatalf("forced promote: %d", rc)
	}

	captured := captureStdout(t, func() {
		if rc := run([]string{"history", "--root", root, "E-01"}); rc != exitOK {
			t.Fatalf("history: %d", rc)
		}
	})
	out := string(captured)
	if !strings.Contains(out, "→ done") {
		t.Errorf("history text should contain `→ done` for forced promote; got:\n%s", out)
	}
	if !strings.Contains(out, "[forced: policy override]") {
		t.Errorf("history text should contain `[forced: policy override]`; got:\n%s", out)
	}
	// The add event has no aiwf-to — its column should render dash.
	if !strings.Contains(out, "-           ") && !strings.Contains(out, "-  ") {
		t.Errorf("history text should contain a dash for the add row's to-column; got:\n%s", out)
	}
}

// TestRun_HistoryUnknownIDIsEmpty: querying a never-allocated id
// returns no events and exits cleanly.
func TestRun_HistoryUnknownIDIsEmpty(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"history", "--root", root, "E-99"}); rc != exitOK {
		t.Errorf("got %d, want %d", rc, exitOK)
	}
}

// TestRun_HistoryReallocateBridgesBothIDs verifies the
// `aiwf-prior-entity:` trailer is queryable: after reallocating, the
// old id still surfaces a final event.
func TestRun_HistoryReallocateBridgesBothIDs(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Bar", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add: %d", rc)
	}
	if rc := run([]string{"reallocate", "--actor", "human/test", "--root", root, "E-01"}); rc != exitOK {
		t.Fatalf("reallocate: %d", rc)
	}

	// Old id sees the reallocate via aiwf-prior-entity.
	old, err := readHistory(context.Background(), root, "E-01")
	if err != nil {
		t.Fatal(err)
	}
	if len(old) < 2 {
		t.Fatalf("expected at least 2 events for E-01 (add + reallocate), got %d", len(old))
	}
	if old[len(old)-1].Verb != "reallocate" {
		t.Errorf("last event for E-01 verb = %q, want reallocate", old[len(old)-1].Verb)
	}

	// New id sees the reallocate via aiwf-entity.
	newH, err := readHistory(context.Background(), root, "E-02")
	if err != nil {
		t.Fatal(err)
	}
	if len(newH) != 1 || newH[0].Verb != "reallocate" {
		t.Errorf("E-02 history = %+v, want one reallocate event", newH)
	}
}

// TestRun_DoctorClean reports problems=0 in a freshly-initialized repo.
func TestRun_DoctorClean(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"doctor", "--root", root}); rc != exitOK {
		t.Errorf("doctor on clean repo = %d, want %d", rc, exitOK)
	}
}

// TestRun_DoctorDetectsSkillDrift: tamper with a materialized skill
// and confirm doctor surfaces it as a problem.
func TestRun_DoctorDetectsSkillDrift(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	skillPath := filepath.Join(root, ".claude", "skills", "aiwf-add", "SKILL.md")
	if err := os.WriteFile(skillPath, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := run([]string{"doctor", "--root", root}); rc != exitFindings {
		t.Errorf("doctor on drifted repo = %d, want %d", rc, exitFindings)
	}
}

// TestRun_DoctorReportsMissingConfig: a repo without aiwf.yaml is a
// problem (run init).
func TestRun_DoctorReportsMissingConfig(t *testing.T) {
	root := t.TempDir()
	if rc := run([]string{"doctor", "--root", root}); rc != exitFindings {
		t.Errorf("doctor on un-init'd repo = %d, want %d", rc, exitFindings)
	}
}

// TestRun_DoctorReportsLegacyActor: a pre-I2.5 aiwf.yaml that still
// carries `actor:` must surface a deprecation note in doctor's
// output. The note is informational — it does NOT increment problems
// (the field is harmless, just unnecessary).
func TestRun_DoctorReportsLegacyActor(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	// Append the legacy `actor:` line to simulate a pre-I2.5 repo.
	contents := []byte("aiwf_version: " + Version + "\nactor: human/legacy\n")
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	lines, _ := doctorReport(root, doctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "deprecated") || !strings.Contains(joined, "human/legacy") {
		t.Errorf("doctor should surface the legacy actor as deprecated; got:\n%s", joined)
	}
}

// TestRun_DoctorReportsRuntimeIdentity: doctor should echo the
// runtime-derived actor + its source so the user can confirm what
// the next mutating verb's aiwf-actor: trailer would say.
func TestRun_DoctorReportsRuntimeIdentity(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	lines, _ := doctorReport(root, doctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "actor:") {
		t.Errorf("doctor should include an `actor:` line surfacing runtime identity:\n%s", joined)
	}
	// The setupCLITestRepo helper configures a deterministic git
	// identity; the source label must be "git config user.email".
	if !strings.Contains(joined, "git config user.email") {
		t.Errorf("doctor's actor line should name git config user.email as the source:\n%s", joined)
	}
}

// TestRun_DoctorVersionSkew exercises the path where aiwf.yaml's
// aiwf_version differs from the running binary. Per
// upgrade-flow-plan.md, pin coherence is *advisory* — the doctor
// surfaces the mismatch on a `pin:` row but does not increment the
// problem count. Hardening the pin into a refusal is a deliberate
// later decision.
func TestRun_DoctorVersionSkew(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	// Replace aiwf.yaml with a version that does not match the binary.
	contents := []byte("aiwf_version: 9.9.9-skew\n")
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	lines, problems := doctorReport(root, doctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "9.9.9-skew") {
		t.Errorf("report should name the pin value; got:\n%s", joined)
	}
	if !strings.Contains(joined, "pin:") {
		t.Errorf("report should carry a `pin:` advisory row; got:\n%s", joined)
	}
	// Skew is advisory; the only problems should come from unrelated
	// rows (none in a fresh init repo). Running doctor exits exitOK
	// when the only difference vs. green is the pin.
	if problems != 0 {
		t.Errorf("pin skew should be advisory (problems=0); got problems=%d:\n%s", problems, joined)
	}
	if rc := run([]string{"doctor", "--root", root}); rc != exitOK {
		t.Errorf("CLI exit on advisory pin skew = %d, want %d", rc, exitOK)
	}
}

// TestRun_DoctorSelfCheck_Passes runs doctor --self-check end-to-end
// and asserts the run reports a clean pass. The self-check spins up
// its own throwaway repo, so no setup is needed beyond the test
// process's git identity (which setupCLITestRepo provides).
func TestRun_DoctorSelfCheck_Passes(t *testing.T) {
	// The test process needs git identity for the self-check repo's
	// commits; setupCLITestRepo already exports it. We don't actually
	// use the returned root — self-check ignores --root.
	_ = setupCLITestRepo(t)

	captured := captureStdout(t, func() {
		if rc := run([]string{"doctor", "--self-check"}); rc != exitOK {
			t.Fatalf("doctor --self-check rc = %d, want %d", rc, exitOK)
		}
	})

	out := string(captured)
	if !strings.Contains(out, "self-check passed") {
		t.Errorf("output missing pass marker:\n%s", out)
	}
	// Each verb appears in the step list. The three update entries
	// pin the install / opt-out / re-install transition added in
	// step 7 of update-broaden-plan.md so a regression that breaks
	// the round-trip surfaces here, not in the field.
	for _, label := range []string{
		"ok    init",
		"ok    add epic",
		"ok    add milestone",
		"ok    add adr",
		"ok    add gap",
		"ok    add decision",
		"ok    add contract",
		"ok    promote",
		"ok    cancel",
		"ok    rename",
		"ok    reallocate",
		"ok    history",
		"ok    render roadmap",
		"ok    update (default install)",
		"ok    update (status_md.auto_update: false → uninstalls hook)",
		"ok    update (status_md.auto_update: true → reinstalls hook)",
		"ok    check",
		"ok    doctor",
	} {
		if !strings.Contains(out, label) {
			t.Errorf("output missing %q:\n%s", label, out)
		}
	}

	// On success the self-check repo should be removed; the path is
	// printed at the start of the run.
	prefix := "self-check repo: "
	idx := strings.Index(out, prefix)
	if idx < 0 {
		t.Fatalf("missing repo path line:\n%s", out)
	}
	after := out[idx+len(prefix):]
	end := strings.IndexByte(after, '\n')
	if end < 0 {
		t.Fatalf("malformed repo path line:\n%s", out)
	}
	repoPath := strings.TrimSpace(after[:end])
	if _, err := os.Stat(repoPath); !os.IsNotExist(err) {
		t.Errorf("self-check should clean up its repo on success: stat %s err=%v", repoPath, err)
	}
}

// TestDoctorReport_Contents checks the pure helper produces the
// expected lines for a typical fresh repo.
func TestDoctorReport_Contents(t *testing.T) {
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		AiwfVersion:   Version,
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	lines, problems := doctorReport(root, doctorOptions{})
	if problems != 0 {
		t.Errorf("problems = %d on a fresh init, want 0\n%s", problems, strings.Join(lines, "\n"))
	}
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"binary:", "config:", "skills:", "ids:"} {
		if !strings.Contains(joined, want) {
			t.Errorf("report missing %q:\n%s", want, joined)
		}
	}
}

// TestDoctor_CheckLatest_ProxyDisabled verifies the opt-in latest
// row is shown when --check-latest is set, and that GOPROXY=off
// produces a benign "unavailable" advisory rather than a failure.
func TestDoctor_CheckLatest_ProxyDisabled(t *testing.T) {
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		AiwfVersion:   Version,
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Setenv("GOPROXY", "off")

	lines, problems := doctorReport(root, doctorOptions{CheckLatest: true})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "latest:") {
		t.Errorf("expected `latest:` row when --check-latest is set:\n%s", joined)
	}
	if !strings.Contains(joined, "proxy disabled") {
		t.Errorf("expected proxy-disabled advisory:\n%s", joined)
	}
	if problems != 0 {
		t.Errorf("proxy-disabled should not increment problems; got %d:\n%s", problems, joined)
	}
}

// TestDoctor_CheckLatest_DefaultOff confirms the latest row does not
// appear in the default (no --check-latest) report path. Doctor must
// stay offline by default.
func TestDoctor_CheckLatest_DefaultOff(t *testing.T) {
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		AiwfVersion:   Version,
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	lines, _ := doctorReport(root, doctorOptions{}) // CheckLatest false
	if strings.Contains(strings.Join(lines, "\n"), "latest:") {
		t.Errorf("latest: row should not appear without --check-latest:\n%s", strings.Join(lines, "\n"))
	}
}

// TestRenderPinCoherence covers the four cases the pin: row can
// produce. Cases use Parse-able inputs directly so the test does
// not depend on runtime/debug.ReadBuildInfo state.
func TestRenderPinCoherence(t *testing.T) {
	cases := []struct {
		name    string
		current string
		pinRaw  string
		wantSub string
	}{
		{"matches", "v0.1.0", "v0.1.0", "matches binary"},
		{"matches no v prefix", "v0.1.0", "0.1.0", "matches binary"},
		{"binary newer", "v0.2.0", "v0.1.0", "binary newer"},
		{"binary older", "v0.1.0", "v0.2.0", "binary older"},
		{"unknown — devel binary", "(devel)", "v0.1.0", "skew unknown"},
		{"unknown — pre-release pin", "v0.1.0", "v0.1.0-rc1", "skew unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := renderPinCoherence(parseVersionForTest(tc.current), tc.pinRaw)
			if !strings.Contains(got, tc.wantSub) {
				t.Errorf("renderPinCoherence(%q, %q) = %q, want substring %q",
					tc.current, tc.pinRaw, got, tc.wantSub)
			}
		})
	}
}

// TestDoctorReport_HookOK: a freshly-initialised repo has the hook
// installed at .git/hooks/pre-push pointing at an existing binary;
// doctor reports it as ok.
func TestDoctorReport_HookOK(t *testing.T) {
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		AiwfVersion:   Version,
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatal(err)
	}
	lines, problems := doctorReport(root, doctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "hook:") {
		t.Errorf("doctor should include a hook: line:\n%s", joined)
	}
	if !strings.Contains(joined, "hook:      ok") {
		t.Errorf("hook line should report ok on a fresh init:\n%s", joined)
	}
	if problems != 0 {
		t.Errorf("fresh init should produce no problems; got %d:\n%s", problems, joined)
	}
}

// TestDoctorReport_HookStalePath_DetectsDrift is the load-bearing
// test for G12: when the binary that init recorded in
// .git/hooks/pre-push no longer exists at that path (binary moved /
// upgraded to a different location / removed), doctor reports the
// drift and increments problems so users see the issue without
// having to discover it on a failed push.
func TestDoctorReport_HookStalePath_DetectsDrift(t *testing.T) {
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		AiwfVersion:   Version,
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatal(err)
	}
	// Hand-edit the hook to point at a non-existent path, simulating
	// a binary that's been moved away.
	hookPath := filepath.Join(root, ".git", "hooks", "pre-push")
	stale := `#!/bin/sh
# aiwf:pre-push
exec /nonexistent/path/to/old-aiwf check
`
	if err := os.WriteFile(hookPath, []byte(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	lines, problems := doctorReport(root, doctorOptions{})
	joined := strings.Join(lines, "\n")
	if problems == 0 {
		t.Errorf("stale hook path should be a problem; got 0:\n%s", joined)
	}
	if !strings.Contains(joined, "stale") && !strings.Contains(joined, "missing") {
		t.Errorf("hook line should describe the staleness:\n%s", joined)
	}
}

// TestDoctorReport_HookMissing: when no .git/hooks/pre-push exists
// at all, doctor reports it as missing (so the user knows pre-push
// validation isn't gating their push, even if everything else is
// clean).
func TestDoctorReport_HookMissing(t *testing.T) {
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		AiwfVersion:   Version,
		ActorOverride: "human/test",
		SkipHook:      true,
	}); err != nil {
		t.Fatal(err)
	}
	lines, _ := doctorReport(root, doctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "hook:") {
		t.Errorf("doctor should include hook: line:\n%s", joined)
	}
	if !strings.Contains(joined, "missing") && !strings.Contains(joined, "not installed") {
		t.Errorf("hook line should describe absence:\n%s", joined)
	}
}

// TestDoctorReport_PreCommitHookOK: fresh init lands the pre-commit
// hook with the marker; doctor reports it ok and increments no
// problems.
func TestDoctorReport_PreCommitHookOK(t *testing.T) {
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		AiwfVersion:   Version,
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatal(err)
	}
	lines, problems := doctorReport(root, doctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "pre-commit: ok") {
		t.Errorf("pre-commit line should report ok on a fresh init:\n%s", joined)
	}
	if problems != 0 {
		t.Errorf("fresh init should produce no problems; got %d:\n%s", problems, joined)
	}
}

// TestDoctorReport_PreCommitHookDisabledByConfig: status_md.auto_update
// false plus no hook on disk is the desired-and-actual-agree state.
// Doctor reports "disabled by config" and increments no pre-commit
// problems.
func TestDoctorReport_PreCommitHookDisabledByConfig(t *testing.T) {
	root := setupCLITestRepo(t)
	// Pre-write aiwf.yaml with the same Version the binary will
	// stamp on init, so the version-skew check doesn't add a
	// confounding problem to the count.
	yaml := []byte("aiwf_version: " + Version + "\nactor: human/test\nstatus_md:\n  auto_update: false\n")
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		AiwfVersion:   Version,
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatal(err)
	}
	lines, problems := doctorReport(root, doctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "pre-commit: disabled by config") {
		t.Errorf("expected 'disabled by config' line:\n%s", joined)
	}
	if problems != 0 {
		t.Errorf("opt-out should produce no problems; got %d:\n%s", problems, joined)
	}
}

// TestDoctorReport_PreCommitHookMissingButFlagOn: hook removed but
// config still says install — drift, doctor flags as a problem and
// hints `aiwf update`.
func TestDoctorReport_PreCommitHookMissingButFlagOn(t *testing.T) {
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		AiwfVersion:   Version,
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, ".git", "hooks", "pre-commit")); err != nil {
		t.Fatal(err)
	}
	lines, problems := doctorReport(root, doctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "pre-commit: missing") {
		t.Errorf("expected 'pre-commit: missing' line:\n%s", joined)
	}
	if problems == 0 {
		t.Errorf("missing pre-commit hook with flag on should be a problem")
	}
	if !strings.Contains(joined, "aiwf update") {
		t.Errorf("remediation should reference `aiwf update`:\n%s", joined)
	}
}

// TestDoctorReport_PreCommitHookPresentButFlagOff: hook on disk but
// the user just flipped the flag — drift in the other direction.
// `aiwf update` removes it.
func TestDoctorReport_PreCommitHookPresentButFlagOff(t *testing.T) {
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		AiwfVersion:   Version,
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatal(err)
	}
	yaml := []byte(`aiwf_version: 0.1.0
actor: human/test
status_md:
  auto_update: false
`)
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	lines, problems := doctorReport(root, doctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "config says off") {
		t.Errorf("expected 'config says off' diagnostic:\n%s", joined)
	}
	if problems == 0 {
		t.Errorf("hook-present-but-config-off should be a problem")
	}
}

// TestDoctorReport_PreCommitHookAlien: a non-marker hook in place.
// Doctor reports it but does not increment problems (the user owns
// the hook; aiwf can't and won't touch it).
func TestDoctorReport_PreCommitHookAlien(t *testing.T) {
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		AiwfVersion:   Version,
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatal(err)
	}
	hookPath := filepath.Join(root, ".git", "hooks", "pre-commit")
	alien := []byte("#!/bin/sh\n# user's own hook, no marker\nexit 0\n")
	if err := os.WriteFile(hookPath, alien, 0o755); err != nil {
		t.Fatal(err)
	}
	lines, problems := doctorReport(root, doctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "pre-commit: present but not aiwf-managed") {
		t.Errorf("expected 'not aiwf-managed' diagnostic:\n%s", joined)
	}
	if problems != 0 {
		t.Errorf("alien pre-commit hook should be informational, got %d problems", problems)
	}
}

// TestDoctorReport_PreCommitHookStalePath: marker present but the
// exec path no longer exists. Same drift class as G12 for pre-push.
func TestDoctorReport_PreCommitHookStalePath(t *testing.T) {
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		AiwfVersion:   Version,
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatal(err)
	}
	hookPath := filepath.Join(root, ".git", "hooks", "pre-commit")
	stale := []byte(`#!/bin/sh
# aiwf:pre-commit
set -e
repo_root="$(git rev-parse --show-toplevel)"
[ -f "$repo_root/aiwf.yaml" ] || exit 0
tmp="$repo_root/STATUS.md.tmp"
if '/nonexistent/path/to/old-aiwf' status --root "$repo_root" --format=md >"$tmp" 2>/dev/null; then
    mv "$tmp" "$repo_root/STATUS.md"
    git add "$repo_root/STATUS.md"
else
    rm -f "$tmp"
fi
exit 0
`)
	if err := os.WriteFile(hookPath, stale, 0o755); err != nil {
		t.Fatal(err)
	}
	lines, problems := doctorReport(root, doctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "pre-commit: stale path") {
		t.Errorf("expected 'pre-commit: stale path' line:\n%s", joined)
	}
	if problems == 0 {
		t.Errorf("stale path should be a problem")
	}
}

// TestDoctorReport_ReportsFilesystemCaseSensitivity: doctor names
// the filesystem's case-sensitivity so users on macOS APFS know
// they're on a case-insensitive volume (where E-01-foo and
// E-01-Foo collapse to one path) before they hit the footgun.
func TestDoctorReport_ReportsFilesystemCaseSensitivity(t *testing.T) {
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		AiwfVersion:   Version,
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatal(err)
	}
	lines, _ := doctorReport(root, doctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "filesystem:") {
		t.Errorf("doctor should report filesystem case-sensitivity:\n%s", joined)
	}
}

// TestDoctorReport_ValidatorAvailability_Warning: a configured
// validator binary missing from PATH appears as a warning line in
// the report and does NOT increment problems (default lenient).
func TestDoctorReport_ValidatorAvailability_Warning(t *testing.T) {
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		AiwfVersion:   Version,
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(`aiwf_version: `+Version+`
actor: human/test
contracts:
  validators:
    cue-missing:
      command: /nonexistent/cue-12345
      args: []
    echo-ok:
      command: echo
      args: []
  entries: []
`), 0o644); err != nil {
		t.Fatal(err)
	}
	lines, problems := doctorReport(root, doctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "validator: cue-missing missing") {
		t.Errorf("missing validator should be reported:\n%s", joined)
	}
	if !strings.Contains(joined, "validator: echo-ok ok") {
		t.Errorf("present validator should be reported:\n%s", joined)
	}
	if problems != 0 {
		t.Errorf("missing validator should NOT increment problems in default mode; got %d\n%s", problems, joined)
	}
}

// TestDoctorReport_ValidatorAvailability_StrictIncrementsProblems:
// strict_validators=true makes a missing validator a hard problem
// in the doctor report (matching the verify-time error).
func TestDoctorReport_ValidatorAvailability_StrictIncrementsProblems(t *testing.T) {
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		AiwfVersion:   Version,
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(`aiwf_version: `+Version+`
actor: human/test
contracts:
  strict_validators: true
  validators:
    cue-missing:
      command: /nonexistent/cue-12345
      args: []
  entries: []
`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, problems := doctorReport(root, doctorOptions{})
	if problems == 0 {
		t.Error("strict_validators=true must make missing validator a problem")
	}
}
