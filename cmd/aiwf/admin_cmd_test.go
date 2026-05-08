package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/initrepo"
)

// TestRun_InitThroughDispatcher confirms `aiwf init` wires through the
// dispatcher: scaffolds dirs, writes aiwf.yaml, materializes skills.
//
// The dispatcher test passes --skip-hook because the in-process
// dispatcher bakes os.Executable() (= the test binary) into any
// hook it installs; firing the test binary as a hook is unsafe
// (see setupCLITestRepo's discipline note). End-to-end hook
// installation is covered by the runBin-style binary integration
// tests, which build a real aiwf and exercise consumer parity.
func TestRun_InitThroughDispatcher(t *testing.T) {
	root := setupCLITestRepo(t)

	rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"})
	if rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if _, err := os.Stat(filepath.Join(root, "aiwf.yaml")); err != nil {
		t.Errorf("aiwf.yaml missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".claude", "skills", "aiwf-add", "SKILL.md")); err != nil {
		t.Errorf("aiwf-add skill missing: %v", err)
	}

	// Re-run to confirm idempotency through the dispatcher.
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Errorf("re-run init: %d", rc)
	}
}

// TestRun_InitDryRun confirms `aiwf init --dry-run` reports the
// would-be ledger, prefixes the output with a dry-run banner, and
// writes nothing to disk.
func TestRun_InitDryRun(t *testing.T) {
	root := setupCLITestRepo(t)

	captured := captureStdout(t, func() {
		if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook", "--dry-run"}); rc != exitOK {
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

// TestRun_InitMigratesAlienHook (G45): when a non-aiwf pre-push hook
// is in place, init auto-migrates it to pre-push.local, installs
// aiwf's chain-aware hook, and exits exitOK. The migrated content
// is preserved byte-for-byte.
func TestRun_InitMigratesAlienHook(t *testing.T) {
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
		// No --skip-hook here: this test exercises the G45 hook
		// migration path and needs init to actually install (and
		// migrate) the hook. The test does not trigger any commits,
		// so the test binary won't be invoked as a hook.
		if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
			t.Errorf("got %d, want %d (G45 auto-migrates, no conflict)", rc, exitOK)
		}
	})
	out := string(captured)

	for _, want := range []string{
		"created    aiwf.yaml",
		"created    work/epics",
		"updated    .claude/skills/aiwf-*",
		"migrated   .git/hooks/pre-push",
		"pre-push.local",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\nfull output:\n%s", want, out)
		}
	}

	// Migrated content lives at .local byte-for-byte.
	migrated, err := os.ReadFile(filepath.Join(hookDir, "pre-push.local"))
	if err != nil {
		t.Fatalf("reading pre-push.local: %v", err)
	}
	if !bytes.Equal(migrated, alien) {
		t.Errorf("migrated content drifted:\n got  %s\n want %s", migrated, alien)
	}
	// pre-push itself is now aiwf's chain-aware hook.
	installed, err := os.ReadFile(filepath.Join(hookDir, "pre-push"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(installed, []byte("# aiwf:pre-push")) {
		t.Errorf("post-migration pre-push lacks aiwf marker")
	}
	if !bytes.Contains(installed, []byte("pre-push.local")) {
		t.Errorf("post-migration pre-push lacks chain reference")
	}
}

// TestRun_UpdateMaterializes wipes a tampered skill file and verifies
// `aiwf update` restores the embedded content byte-for-byte.
func TestRun_UpdateMaterializes(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
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
	// No --skip-hook: this test exercises update's hook-refresh
	// behavior and needs init to land a real hook first. The test
	// triggers no commits, so the embedded test-binary path never
	// fires as a hook.
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
	// No --skip-hook: same rationale as TestRun_UpdateRefreshesPrePushHook.
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

// TestRun_UpdateDropsRegenKeepsGateOnOptOut (G42): run init (hook
// installed by default), flip status_md.auto_update: false, run
// update → the marker-managed pre-commit hook stays installed
// (tree-discipline gate is enforcement, not opt-out-able), but the
// STATUS.md regen step is dropped from the script body.
func TestRun_UpdateDropsRegenKeepsGateOnOptOut(t *testing.T) {
	root := setupCLITestRepo(t)
	// No --skip-hook: this test verifies G42 round-trip behavior
	// (install → opt-out → re-install) which needs real hook
	// installation through init. No commits triggered.
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
	body, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("pre-commit hook missing after opt-out (G42 violation): %v", err)
	}
	if !strings.Contains(string(body), "check --shape-only") {
		t.Errorf("hook missing tree-discipline gate after opt-out:\n%s", body)
	}
	if strings.Contains(string(body), "status --root") {
		t.Errorf("opt-out must drop STATUS.md regen step:\n%s", body)
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
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
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
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
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
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--tdd", "none", "--epic", "E-01", "--title", "First", "--actor", "human/test", "--root", root}); rc != exitOK {
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
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
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
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
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
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
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
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
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
	// No --skip-hook: doctor's "clean" judgement requires both
	// hooks to be installed. The test runs only doctor afterward
	// (read-only), no commits, so the test-binary-as-hook hazard
	// doesn't apply.
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
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
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
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
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
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
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

// TestRun_DoctorReportsLegacyAiwfVersion (G47): a pre-G47 aiwf.yaml
// carrying an `aiwf_version:` key surfaces a deprecation note via
// doctor (mirrors the legacy-actor note). The advisory does not
// increment the doctor problem count.
func TestRun_DoctorReportsLegacyAiwfVersion(t *testing.T) {
	root := setupCLITestRepo(t)
	// No --skip-hook: the test asserts problems == 0 (the legacy
	// field is advisory, not a problem). Without hooks installed
	// the missing-hook problems would mask the assertion. No
	// commits triggered.
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	// Replace aiwf.yaml with one that carries the legacy field.
	contents := []byte("aiwf_version: 9.9.9-legacy\n")
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	lines, problems := doctorReport(root, doctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "9.9.9-legacy") {
		t.Errorf("report should name the legacy aiwf_version value; got:\n%s", joined)
	}
	if !strings.Contains(joined, "deprecated `aiwf_version:") {
		t.Errorf("report should carry a deprecation note for aiwf_version; got:\n%s", joined)
	}
	// Legacy field is advisory — no problem count bump.
	if problems != 0 {
		t.Errorf("legacy aiwf_version should be advisory (problems=0); got problems=%d:\n%s", problems, joined)
	}
	if rc := run([]string{"doctor", "--root", root}); rc != exitOK {
		t.Errorf("CLI exit on advisory legacy aiwf_version = %d, want %d", rc, exitOK)
	}
}

// TestRun_DoctorSelfCheck_Passes runs doctor --self-check end-to-end
// and asserts the run reports a clean pass. The self-check spins up
// its own throwaway repo and exercises every verb including ones
// that commit (which fire the pre-commit hook installed during
// init). Driving it as a subprocess via runBin gives consumer
// parity: the hook bakes in a real aiwf binary path rather than
// the test binary's path, so the hook fires correctly. Running
// in-process via the dispatcher would deadlock (hook execs the
// test binary).
func TestRun_DoctorSelfCheck_Passes(t *testing.T) {
	root := t.TempDir()

	out, err := runBin(t, root, "", nil, "doctor", "--self-check")
	if err != nil {
		t.Fatalf("doctor --self-check: %v\n%s", err, out)
	}
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
		"ok    update (status_md.auto_update: false → keeps gate, drops regen)",
		"ok    update (status_md.auto_update: true → reinstates regen)",
		"ok    check",
		"ok    doctor",
		// M-070/AC-7: end-to-end coverage of the recommended-plugin
		// check, both the warning-fires path and the install-silences
		// path. Adding these labels keeps the seam test honest about
		// what the production self-check actually exercises.
		"ok    doctor recommended-plugins fixture: declare in aiwf.yaml",
		"ok    doctor recommended-plugins fixture: warning silent after install",
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

// TestDoctor_HonorsCoreHooksPath (G48): when the consumer has set
// `core.hooksPath`, doctor reads the hook at the configured location
// (not hardcoded `.git/hooks/`) and reports against it. Without
// G48's helper, doctor would say "missing — pre-push validation not
// installed" because it looked at the wrong location.
func TestDoctor_HonorsCoreHooksPath(t *testing.T) {
	root := setupCLITestRepo(t)
	// Configure a relative tracked-hooks dir before init lands hooks.
	if err := osExec(t, root, "git", "config", "core.hooksPath", "scripts/git-hooks"); err != nil {
		t.Fatalf("git config core.hooksPath: %v", err)
	}
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Confirm the install landed at the configured path (this is
	// what G48 fixes).
	configured := filepath.Join(root, "scripts", "git-hooks")
	for _, name := range []string{"pre-push", "pre-commit"} {
		if _, err := os.Stat(filepath.Join(configured, name)); err != nil {
			t.Fatalf("%s missing at configured hooksPath: %v", name, err)
		}
	}

	lines, problems := doctorReport(root, doctorOptions{})
	joined := strings.Join(lines, "\n")
	if problems != 0 {
		t.Errorf("doctor problems = %d, want 0\n%s", problems, joined)
	}
	// The hook line should report ok against the configured path,
	// not the default. We don't pin the exact phrasing — just
	// confirm doctor isn't lying about a missing hook.
	if strings.Contains(joined, "hook:      missing") {
		t.Errorf("doctor reports pre-push hook missing despite install at configured path:\n%s", joined)
	}
	if strings.Contains(joined, "pre-commit: missing") {
		t.Errorf("doctor reports pre-commit hook missing despite install at configured path:\n%s", joined)
	}
}

// TestDoctor_HookChainReporting (G45): doctor reports the .local
// sibling state for both pre-push and pre-commit hooks. Three states
// matter: absent (no suffix), present + executable ("chains to ..."),
// present + non-executable (error, increments problem count).
func TestDoctor_HookChainReporting(t *testing.T) {
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// State 1: no .local sibling — doctor clean, no chain mention.
	t.Run("absent: no chain mention", func(t *testing.T) {
		lines, problems := doctorReport(root, doctorOptions{})
		joined := strings.Join(lines, "\n")
		if problems != 0 {
			t.Errorf("problems = %d, want 0\n%s", problems, joined)
		}
		if strings.Contains(joined, "chains to") {
			t.Errorf("clean repo report mentions 'chains to' (no .local should be present):\n%s", joined)
		}
	})

	// State 2: executable .local — doctor clean, chain reported.
	t.Run("executable: chains to noted", func(t *testing.T) {
		hooksDir := filepath.Join(root, ".git", "hooks")
		localPP := filepath.Join(hooksDir, "pre-push.local")
		localPC := filepath.Join(hooksDir, "pre-commit.local")
		if err := os.WriteFile(localPP, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(localPC, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = os.Remove(localPP)
			_ = os.Remove(localPC)
		})

		lines, problems := doctorReport(root, doctorOptions{})
		joined := strings.Join(lines, "\n")
		if problems != 0 {
			t.Errorf("problems = %d on executable .local siblings, want 0\n%s", problems, joined)
		}
		if !strings.Contains(joined, "chains to .git/hooks/pre-push.local") {
			t.Errorf("report missing pre-push chain notice:\n%s", joined)
		}
		if !strings.Contains(joined, "chains to .git/hooks/pre-commit.local") {
			t.Errorf("report missing pre-commit chain notice:\n%s", joined)
		}
	})

	// State 3: non-executable .local — doctor flags as error.
	t.Run("non-executable: doctor errors", func(t *testing.T) {
		hooksDir := filepath.Join(root, ".git", "hooks")
		localPP := filepath.Join(hooksDir, "pre-push.local")
		if err := os.WriteFile(localPP, []byte("#!/bin/sh\nexit 0\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Remove(localPP) })

		lines, problems := doctorReport(root, doctorOptions{})
		joined := strings.Join(lines, "\n")
		if problems == 0 {
			t.Errorf("problems = 0 on non-executable .local, want >0\n%s", joined)
		}
		if !strings.Contains(joined, "not executable") {
			t.Errorf("report missing 'not executable':\n%s", joined)
		}
	})
}

// TestDoctorReport_Contents checks the pure helper produces the
// expected lines for a typical fresh repo.
func TestDoctorReport_Contents(t *testing.T) {
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
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
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	lines, _ := doctorReport(root, doctorOptions{}) // CheckLatest false
	if strings.Contains(strings.Join(lines, "\n"), "latest:") {
		t.Errorf("latest: row should not appear without --check-latest:\n%s", strings.Join(lines, "\n"))
	}
}

// TestDoctorReport_HookOK: a freshly-initialised repo has the hook
// installed at .git/hooks/pre-push pointing at an existing binary;
// doctor reports it as ok.
func TestDoctorReport_HookOK(t *testing.T) {
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
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

// TestDoctorReport_PreCommitHookGateOnly (G42): status_md.auto_update
// false leaves the pre-commit hook installed in gate-only mode.
// Doctor reports "ok, gate-only" and counts no problems — that's
// the desired-and-actual-agree state under G42.
func TestDoctorReport_PreCommitHookGateOnly(t *testing.T) {
	root := setupCLITestRepo(t)
	// Pre-write aiwf.yaml with the same Version the binary will
	// stamp on init, so the version-skew check doesn't add a
	// confounding problem to the count.
	yaml := []byte("aiwf_version: " + Version + "\nactor: human/test\nstatus_md:\n  auto_update: false\n")
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatal(err)
	}
	lines, problems := doctorReport(root, doctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "pre-commit: ok, gate-only") {
		t.Errorf("expected 'ok, gate-only' line under G42:\n%s", joined)
	}
	if problems != 0 {
		t.Errorf("gate-only mode should produce no problems; got %d:\n%s", problems, joined)
	}
}

// TestDoctorReport_PreCommitHookMissingButFlagOn: hook removed but
// config still says install — drift, doctor flags as a problem and
// hints `aiwf update`.
func TestDoctorReport_PreCommitHookMissingButFlagOn(t *testing.T) {
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
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

// TestDoctorReport_RecommendedPlugins_EmptyConfig_NoOutputNoFileRead:
// M-070/AC-4 — when `doctor.recommended_plugins` is absent or empty,
// the new check makes zero observations: no `recommended-plugin-not-installed`
// line in the output and no problems++. Verified with two configs:
// the field absent entirely + an explicit `[]`. To make the
// "no file read" half observable without process tracing, $HOME is
// set to a directory we did NOT populate; if the check incorrectly
// tried to read `installed_plugins.json` it would still find nothing
// and return empty — so this test pairs with the fixture-injected
// AC-5/AC-6 tests below where the file presence matters.
func TestDoctorReport_RecommendedPlugins_EmptyConfig_NoOutputNoFileRead(t *testing.T) {
	cases := []struct {
		name      string
		yamlExtra string
	}{
		{name: "field absent", yamlExtra: ""},
		{name: "empty list", yamlExtra: "doctor:\n  recommended_plugins: []\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := setupCLITestRepo(t)
			if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
				t.Fatalf("init: %d", rc)
			}
			// Append (or rewrite) aiwf.yaml with the test's extra block.
			contents := []byte("hosts: [claude-code]\n" + tc.yamlExtra)
			if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), contents, 0o644); err != nil {
				t.Fatal(err)
			}
			t.Setenv("HOME", t.TempDir())
			lines, _ := doctorReport(root, doctorOptions{})
			joined := strings.Join(lines, "\n")
			if strings.Contains(joined, "recommended-plugin-not-installed") {
				t.Errorf("empty config emitted recommended-plugin warnings:\n%s", joined)
			}
		})
	}
}

// TestDoctorReport_RecommendedPlugins_OneMissing_OneWarningWithInstall:
// M-070/AC-3 — one declared, none installed. Exactly one warning line
// per missing plugin; the warning text carries (a) the
// `recommended-plugin-not-installed` finding code so a script can grep,
// (b) the plugin id name@marketplace, and (c) the canonical install
// command `claude /plugin install <id>`.
func TestDoctorReport_RecommendedPlugins_OneMissing_OneWarningWithInstall(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	contents := []byte("hosts: [claude-code]\ndoctor:\n  recommended_plugins:\n    - aiwf-extensions@ai-workflow-rituals\n")
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	lines, _ := doctorReport(root, doctorOptions{})
	joined := strings.Join(lines, "\n")
	if c := strings.Count(joined, "recommended-plugin-not-installed"); c != 1 {
		t.Errorf("count(recommended-plugin-not-installed) = %d, want 1; output:\n%s", c, joined)
	}
	if !strings.Contains(joined, "aiwf-extensions@ai-workflow-rituals") {
		t.Errorf("warning missing plugin id; output:\n%s", joined)
	}
	if !strings.Contains(joined, "claude /plugin install aiwf-extensions@ai-workflow-rituals") {
		t.Errorf("warning missing install command; output:\n%s", joined)
	}
}

// TestDoctorReport_RecommendedPlugins_NoneInstalled_NWarnings: M-070/AC-3
// — N declared, none installed produces exactly N warnings (never
// deduped, never skipped). Order matches declaration order so the user
// can correlate with their aiwf.yaml.
func TestDoctorReport_RecommendedPlugins_NoneInstalled_NWarnings(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	contents := []byte(`hosts: [claude-code]
doctor:
  recommended_plugins:
    - aiwf-extensions@ai-workflow-rituals
    - wf-rituals@ai-workflow-rituals
    - some-third@somewhere
`)
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	lines, _ := doctorReport(root, doctorOptions{})
	joined := strings.Join(lines, "\n")
	if c := strings.Count(joined, "recommended-plugin-not-installed"); c != 3 {
		t.Errorf("count(recommended-plugin-not-installed) = %d, want 3; output:\n%s", c, joined)
	}
	for _, plugin := range []string{
		"aiwf-extensions@ai-workflow-rituals",
		"wf-rituals@ai-workflow-rituals",
		"some-third@somewhere",
	} {
		if !strings.Contains(joined, plugin) {
			t.Errorf("warning missing plugin %q; output:\n%s", plugin, joined)
		}
	}
}

// TestDoctorReport_RecommendedPlugins_AllInstalledForProject_NoWarning:
// M-070/AC-5 — every recommended plugin has a project-scope install
// whose projectPath matches the consumer root → zero
// `recommended-plugin-not-installed` lines, doctor exits OK on this
// section. Fixture mirrors the real installed_plugins.json shape.
func TestDoctorReport_RecommendedPlugins_AllInstalledForProject_NoWarning(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	contents := []byte("hosts: [claude-code]\ndoctor:\n  recommended_plugins:\n    - aiwf-extensions@ai-workflow-rituals\n")
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	writeInstalledPluginsFixture(t, home, fmt.Sprintf(`{
  "version": 2,
  "plugins": {
    "aiwf-extensions@ai-workflow-rituals": [
      {"scope": "project", "projectPath": %q}
    ]
  }
}`, root))
	t.Setenv("HOME", home)
	lines, _ := doctorReport(root, doctorOptions{})
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "recommended-plugin-not-installed") {
		t.Errorf("matched install should silence the warning; got:\n%s", joined)
	}
}

// TestDoctorReport_RecommendedPlugins_InstalledElsewhereStillWarns:
// M-070/AC-6 — the session-canonical case: a recommended plugin is
// installed for ANOTHER repo's project scope (and possibly user scope
// too), but not for THIS consumer's root. Warning still fires;
// installation elsewhere does not silence it.
func TestDoctorReport_RecommendedPlugins_InstalledElsewhereStillWarns(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	contents := []byte("hosts: [claude-code]\ndoctor:\n  recommended_plugins:\n    - aiwf-extensions@ai-workflow-rituals\n")
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	writeInstalledPluginsFixture(t, home, `{
  "version": 2,
  "plugins": {
    "aiwf-extensions@ai-workflow-rituals": [
      {"scope": "project", "projectPath": "/Users/x/Projects/some-other-repo"},
      {"scope": "user"}
    ]
  }
}`)
	t.Setenv("HOME", home)
	lines, _ := doctorReport(root, doctorOptions{})
	joined := strings.Join(lines, "\n")
	if c := strings.Count(joined, "recommended-plugin-not-installed"); c != 1 {
		t.Errorf("install elsewhere must not silence warning: count=%d; output:\n%s", c, joined)
	}
}

// TestDoctorReport_RecommendedPlugins_AreSoftWarning_DoNotIncrementProblems:
// per M-070 spec: "Severity: warning. Plugins are advisory; refusing on
// absence is too strong." Even when the warning fires, doctor's exit
// code stays 0 (problems unchanged). This decoupling is what lets a
// consumer declare recommended plugins without breaking CI.
func TestDoctorReport_RecommendedPlugins_AreSoftWarning_DoNotIncrementProblems(t *testing.T) {
	root := setupCLITestRepo(t)
	// --skip-hook keeps the test independent of hook installation; but
	// with skip-hook, doctor reports a missing-hook problem. The
	// AC-targeted assertion is "the recommended-plugin warning does
	// not contribute to the problem count" — measured by comparing the
	// problem count with vs. without the recommended-plugin block.
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	t.Setenv("HOME", t.TempDir())
	// Baseline problems count with no recommended_plugins declared.
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("hosts: [claude-code]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, baseProblems := doctorReport(root, doctorOptions{})
	// Add a missing recommended plugin; problems must not increase.
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("hosts: [claude-code]\ndoctor:\n  recommended_plugins:\n    - aiwf-extensions@ai-workflow-rituals\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, withProblems := doctorReport(root, doctorOptions{})
	if withProblems != baseProblems {
		t.Errorf("recommended-plugin warning incremented problems: base=%d with=%d (warning must be soft)", baseProblems, withProblems)
	}
}

// TestAppendRecommendedPluginsReport_NilCfg_NoOp: helper called with
// nil cfg returns input unchanged. Reaches the `cfg == nil` early-return
// guard the public doctorReport relies on when `config.Load` failed
// for a non-NotFound reason (cfg comes back nil).
func TestAppendRecommendedPluginsReport_NilCfg_NoOp(t *testing.T) {
	in := []string{"line a", "line b"}
	out := appendRecommendedPluginsReport(in, nil, t.TempDir())
	if len(out) != len(in) {
		t.Fatalf("len = %d, want %d (helper must not mutate input on nil cfg)", len(out), len(in))
	}
	for i, want := range in {
		if out[i] != want {
			t.Errorf("[%d] = %q, want %q", i, out[i], want)
		}
	}
}

// TestDoctorReport_RecommendedPlugins_CorruptedIndex_EmitsAdvisory:
// when installed_plugins.json exists but isn't valid JSON, the helper
// emits a single advisory line naming the failure and skips the
// per-plugin checks (no warnings, no panic). Mirrors how the existing
// validator/render checks treat unrecoverable read failures.
func TestDoctorReport_RecommendedPlugins_CorruptedIndex_EmitsAdvisory(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	contents := []byte("hosts: [claude-code]\ndoctor:\n  recommended_plugins:\n    - aiwf-extensions@ai-workflow-rituals\n")
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	writeInstalledPluginsFixture(t, home, "{not json")
	t.Setenv("HOME", home)
	lines, _ := doctorReport(root, doctorOptions{})
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "recommended-plugin-not-installed") {
		t.Errorf("corrupted index should not produce per-plugin warnings:\n%s", joined)
	}
	if !strings.Contains(joined, "installed_plugins.json") {
		t.Errorf("advisory line should name installed_plugins.json:\n%s", joined)
	}
}

// writeInstalledPluginsFixture writes a synthetic installed_plugins.json
// under <home>/.claude/plugins/ so a t.Setenv("HOME", home) test
// configures the doctor's plugin lookup deterministically. Used by the
// AC-5 / AC-6 tests above.
func writeInstalledPluginsFixture(t *testing.T, home, body string) {
	t.Helper()
	dir := filepath.Join(home, ".claude", "plugins")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "installed_plugins.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestDoctorReport_ValidatorAvailability_StrictIncrementsProblems:
// strict_validators=true makes a missing validator a hard problem
// in the doctor report (matching the verify-time error).
func TestDoctorReport_ValidatorAvailability_StrictIncrementsProblems(t *testing.T) {
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
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
