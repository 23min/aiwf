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
)

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

// TestRun_DoctorVersionSkew exercises the path where aiwf.yaml's
// aiwf_version differs from the binary's Version constant. The CLI
// should exit with `findings` and the report should mention both
// values so the user knows what changed.
func TestRun_DoctorVersionSkew(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	// Replace aiwf.yaml with a version that does not match Version.
	contents := []byte("aiwf_version: 9.9.9-skew\nactor: human/test\n")
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	lines, problems := doctorReport(root)
	if problems == 0 {
		t.Errorf("expected version-skew problem, got clean report:\n%s", strings.Join(lines, "\n"))
	}
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "9.9.9-skew") || !strings.Contains(joined, Version) {
		t.Errorf("report should mention both versions; got:\n%s", joined)
	}
	if rc := run([]string{"doctor", "--root", root}); rc != exitFindings {
		t.Errorf("CLI exit on version skew = %d, want %d", rc, exitFindings)
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
	// Each verb appears in the step list.
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
		"ok    update",
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
	lines, problems := doctorReport(root)
	if problems != 0 {
		t.Errorf("problems = %d on a fresh init, want 0\n%s", problems, strings.Join(lines, "\n"))
	}
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"config:", "skills:", "ids:"} {
		if !strings.Contains(joined, want) {
			t.Errorf("report missing %q:\n%s", want, joined)
		}
	}
}
