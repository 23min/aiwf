package initrepo

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/tools/internal/config"
)

// freshGitRepo gives each test an isolated repo with a deterministic
// git identity so deriveActor's user.email path is exercisable.
func freshGitRepo(t *testing.T) string {
	t.Helper()
	t.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	root := t.TempDir()
	cmd := exec.Command("git", "init", "-q", root)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	// Configure user.email locally so deriveActor's git-config path works
	// regardless of the test host's global config.
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		c := exec.Command("git", args...)
		c.Dir = root
		if out, cErr := c.CombinedOutput(); cErr != nil {
			t.Fatalf("git %v: %v\n%s", args, cErr, out)
		}
	}
	return root
}

func TestInit_FreshRepo(t *testing.T) {
	root := freshGitRepo(t)
	res, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if res == nil || len(res.Steps) == 0 {
		t.Fatal("expected non-empty result")
	}

	// aiwf.yaml present and parseable.
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AiwfVersion != "0.1.0" {
		t.Errorf("aiwf_version = %q, want 0.1.0", cfg.AiwfVersion)
	}
	if cfg.Actor != "human/peter" {
		t.Errorf("actor = %q, want human/peter", cfg.Actor)
	}

	// All scaffolded dirs exist.
	for _, d := range []string{
		"work/epics", "work/gaps", "work/decisions", "work/contracts", "docs/adr",
	} {
		info, sErr := os.Stat(filepath.Join(root, d))
		if sErr != nil || !info.IsDir() {
			t.Errorf("dir %s missing or not a dir: %v", d, sErr)
		}
	}

	// Skills materialized.
	for _, name := range []string{"wf-add", "wf-promote", "wf-rename", "wf-reallocate", "wf-history", "wf-check"} {
		path := filepath.Join(root, ".claude", "skills", name, "SKILL.md")
		if _, sErr := os.Stat(path); sErr != nil {
			t.Errorf("skill %s missing: %v", name, sErr)
		}
	}

	// .gitignore contains skill paths.
	gi, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(gi), ".claude/skills/wf-add/") {
		t.Errorf(".gitignore missing skill paths: %s", gi)
	}

	// CLAUDE.md created from template.
	cm, err := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("CLAUDE.md: %v", err)
	}
	if !strings.Contains(string(cm), "aiwf check") {
		t.Errorf("CLAUDE.md template not written: %s", cm)
	}

	// Pre-push hook installed with marker.
	hook, err := os.ReadFile(filepath.Join(root, ".git", "hooks", "pre-push"))
	if err != nil {
		t.Fatalf("pre-push hook: %v", err)
	}
	if !strings.Contains(string(hook), HookMarker()) {
		t.Errorf("pre-push hook missing marker: %s", hook)
	}
}

// TestInit_Idempotent re-runs Init and confirms it preserves
// pre-existing aiwf.yaml and CLAUDE.md byte-for-byte.
func TestInit_Idempotent(t *testing.T) {
	root := freshGitRepo(t)
	if _, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"}); err != nil {
		t.Fatalf("Init #1: %v", err)
	}
	yamlBefore, _ := os.ReadFile(filepath.Join(root, config.FileName))
	claudeBefore, _ := os.ReadFile(filepath.Join(root, "CLAUDE.md"))

	if _, err := Init(context.Background(), root, Options{AiwfVersion: "0.2.0"}); err != nil {
		t.Fatalf("Init #2: %v", err)
	}
	yamlAfter, _ := os.ReadFile(filepath.Join(root, config.FileName))
	claudeAfter, _ := os.ReadFile(filepath.Join(root, "CLAUDE.md"))

	if !bytes.Equal(yamlBefore, yamlAfter) {
		t.Errorf("aiwf.yaml mutated on re-run:\nbefore=%q\nafter=%q", yamlBefore, yamlAfter)
	}
	if !bytes.Equal(claudeBefore, claudeAfter) {
		t.Error("CLAUDE.md mutated on re-run")
	}
}

// TestInit_PreservesExistingConfig checks Init does not overwrite a
// manually-edited aiwf.yaml that already has its own actor.
func TestInit_PreservesExistingConfig(t *testing.T) {
	root := freshGitRepo(t)
	custom := []byte("aiwf_version: 9.9.9\nactor: human/somebody-else\n")
	if err := os.WriteFile(filepath.Join(root, config.FileName), custom, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(root, config.FileName))
	if !bytes.Equal(got, custom) {
		t.Errorf("aiwf.yaml overwritten despite being preserved")
	}
}

// TestInit_PreservesExistingClaudeMd: do not overwrite a project's own
// CLAUDE.md.
func TestInit_PreservesExistingClaudeMd(t *testing.T) {
	root := freshGitRepo(t)
	custom := []byte("# This project has its own CLAUDE.md\n")
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), custom, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if !bytes.Equal(got, custom) {
		t.Error("CLAUDE.md overwritten despite being preserved")
	}
}

// TestInit_SkipsAlienPreHook: a pre-push hook that doesn't carry the
// marker is treated as user-managed. Init reports the skip via
// HookConflict + a "skipped" step in the ledger, leaves the user's
// hook untouched, and still completes every other step so the user
// sees the full picture of what landed.
func TestInit_SkipsAlienPreHook(t *testing.T) {
	root := freshGitRepo(t)
	hookDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hookDir, 0o755); err != nil {
		t.Fatal(err)
	}
	alien := []byte("#!/bin/sh\n# someone else's hook\nexit 0\n")
	if err := os.WriteFile(filepath.Join(hookDir, "pre-push"), alien, 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"})
	if err != nil {
		t.Fatalf("Init returned error on alien hook (should be soft skip): %v", err)
	}
	if !res.HookConflict {
		t.Errorf("HookConflict = false, want true")
	}
	// Alien hook is intact.
	got, _ := os.ReadFile(filepath.Join(hookDir, "pre-push"))
	if !bytes.Equal(got, alien) {
		t.Errorf("alien hook clobbered: %s", got)
	}
	// All non-hook steps still ran. Ledger contains the expected What
	// values, with the hook step itself marked Skipped.
	wantWhats := []string{
		"aiwf.yaml",
		"work/epics", "work/gaps", "work/decisions", "work/contracts",
		"docs/adr",
		".claude/skills/wf-*",
		".gitignore",
		"CLAUDE.md",
		".git/hooks/pre-push",
	}
	gotWhats := make([]string, len(res.Steps))
	for i, s := range res.Steps {
		gotWhats[i] = s.What
	}
	if strings.Join(gotWhats, "|") != strings.Join(wantWhats, "|") {
		t.Errorf("step ledger:\n got  %v\n want %v", gotWhats, wantWhats)
	}
	hookStep := res.Steps[len(res.Steps)-1]
	if hookStep.Action != ActionSkipped {
		t.Errorf("hook step action = %q, want %q", hookStep.Action, ActionSkipped)
	}
	if hookStep.Detail == "" {
		t.Errorf("hook step Detail empty; want a remediation hint")
	}
}

// TestInit_OverwritesOwnHook: re-running init when our own hook is in
// place must succeed (idempotent).
func TestInit_OverwritesOwnHook(t *testing.T) {
	root := freshGitRepo(t)
	if _, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"}); err != nil {
		t.Fatalf("Init #1: %v", err)
	}
	// Tamper with the hook in a way that keeps the marker.
	hookPath := filepath.Join(root, ".git", "hooks", "pre-push")
	tampered := []byte("#!/bin/sh\n" + HookMarker() + "\n# tampered\nexit 1\n")
	if err := os.WriteFile(hookPath, tampered, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"}); err != nil {
		t.Fatalf("Init #2: %v", err)
	}
	got, _ := os.ReadFile(hookPath)
	if bytes.Equal(got, tampered) {
		t.Errorf("hook not refreshed; still tampered version")
	}
	if !strings.Contains(string(got), HookMarker()) {
		t.Errorf("hook lost its marker: %s", got)
	}
}

// TestInit_RejectsBadActorOverride catches malformed --actor values
// before any disk writes (aiwf.yaml shouldn't be created when it
// would be invalid).
func TestInit_RejectsBadActorOverride(t *testing.T) {
	root := freshGitRepo(t)
	_, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0", ActorOverride: "no slashes here"})
	if err == nil {
		t.Fatal("expected error from malformed --actor")
	}
	if _, sErr := os.Stat(filepath.Join(root, config.FileName)); !os.IsNotExist(sErr) {
		t.Errorf("aiwf.yaml should not exist after a rejected actor; stat err=%v", sErr)
	}
}

// TestInit_GitignorePreservesExisting: an existing .gitignore with
// unrelated entries is preserved verbatim with skill paths appended.
func TestInit_GitignorePreservesExisting(t *testing.T) {
	root := freshGitRepo(t)
	existing := []byte("# user gitignore\nnode_modules/\n")
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), existing, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(root, ".gitignore"))
	if !strings.HasPrefix(string(got), "# user gitignore\nnode_modules/\n") {
		t.Errorf("existing .gitignore prefix lost: %s", got)
	}
	if !strings.Contains(string(got), ".claude/skills/wf-add/") {
		t.Errorf("skill paths not appended: %s", got)
	}
}

// TestInit_GitignoreNoDoubleAppend: re-running init does not add the
// skill paths twice.
func TestInit_GitignoreNoDoubleAppend(t *testing.T) {
	root := freshGitRepo(t)
	if _, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"}); err != nil {
		t.Fatalf("Init #1: %v", err)
	}
	if _, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"}); err != nil {
		t.Fatalf("Init #2: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(root, ".gitignore"))
	count := strings.Count(string(got), ".claude/skills/wf-add/")
	if count != 1 {
		t.Errorf("skill path appears %d times, want 1\n%s", count, got)
	}
}

// TestInit_PreservesExistingEntities: pre-existing entity files in
// work/ must not be touched (they show up as findings on the next
// `aiwf check` and serve as a migration to-do list).
func TestInit_PreservesExistingEntities(t *testing.T) {
	root := freshGitRepo(t)
	dir := filepath.Join(root, "work", "epics", "E-01-foo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte("---\nid: E-01\ntitle: Foo\nstatus: active\n---\n")
	if err := os.WriteFile(filepath.Join(dir, "epic.md"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "epic.md"))
	if !bytes.Equal(got, body) {
		t.Errorf("existing entity file mutated: %s", got)
	}
}
