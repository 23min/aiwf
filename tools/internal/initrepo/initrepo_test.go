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
	// Identity is no longer stored — aiwf init must omit `actor:`
	// from the fresh aiwf.yaml. The git-config-derived actor still
	// gates init's success (deriveActor refuses if absent).
	if cfg.LegacyActor != "" {
		t.Errorf("LegacyActor = %q, want empty (init must not write actor: to fresh aiwf.yaml)", cfg.LegacyActor)
	}
	yamlBytes, readErr := os.ReadFile(filepath.Join(root, config.FileName))
	if readErr != nil {
		t.Fatalf("read aiwf.yaml: %v", readErr)
	}
	if strings.Contains(string(yamlBytes), "actor:") {
		t.Errorf("aiwf.yaml contains actor: key (post-I2.5 init must omit it):\n%s", yamlBytes)
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
	for _, name := range []string{"aiwf-add", "aiwf-promote", "aiwf-rename", "aiwf-reallocate", "aiwf-history", "aiwf-check", "aiwf-status"} {
		path := filepath.Join(root, ".claude", "skills", name, "SKILL.md")
		if _, sErr := os.Stat(path); sErr != nil {
			t.Errorf("skill %s missing: %v", name, sErr)
		}
	}

	// .gitignore contains the wildcard skill pattern + manifest (G19).
	gi, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(gi), ".claude/skills/aiwf-*/") {
		t.Errorf(".gitignore missing skill wildcard: %s", gi)
	}
	if !strings.Contains(string(gi), ".claude/skills/.aiwf-owned") {
		t.Errorf(".gitignore missing manifest entry: %s", gi)
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
	// values, with the pre-push step itself marked Skipped.
	wantWhats := []string{
		"aiwf.yaml",
		"work/epics", "work/gaps", "work/decisions", "work/contracts",
		"docs/adr",
		"CLAUDE.md",
		".claude/skills/aiwf-*",
		".gitignore",
		".git/hooks/pre-push",
		".git/hooks/pre-commit",
	}
	gotWhats := make([]string, len(res.Steps))
	for i, s := range res.Steps {
		gotWhats[i] = s.What
	}
	if strings.Join(gotWhats, "|") != strings.Join(wantWhats, "|") {
		t.Errorf("step ledger:\n got  %v\n want %v", gotWhats, wantWhats)
	}
	prePushStep := findStep(t, res.Steps, ".git/hooks/pre-push")
	if prePushStep.Action != ActionSkipped {
		t.Errorf("pre-push step action = %q, want %q", prePushStep.Action, ActionSkipped)
	}
	if prePushStep.Detail == "" {
		t.Errorf("pre-push step Detail empty; want a remediation hint")
	}
}

// findStep returns the StepResult with What == what; fails the test
// if not found. Tests that target a specific ledger row (rather than
// the last) use this so step-order tweaks don't ripple.
func findStep(t *testing.T, steps []StepResult, what string) StepResult {
	t.Helper()
	for _, s := range steps {
		if s.What == what {
			return s
		}
	}
	t.Fatalf("ledger has no step %q; got %v", what, steps)
	return StepResult{}
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
	if !strings.Contains(string(got), ".claude/skills/aiwf-*/") {
		t.Errorf("skill wildcard not appended: %s", got)
	}
}

// TestInit_GitignoreNoDoubleAppend: re-running init does not add the
// skill wildcard twice (G19: with the wildcard, no per-skill drift).
func TestInit_GitignoreNoDoubleAppend(t *testing.T) {
	root := freshGitRepo(t)
	if _, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"}); err != nil {
		t.Fatalf("Init #1: %v", err)
	}
	if _, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"}); err != nil {
		t.Fatalf("Init #2: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(root, ".gitignore"))
	if c := strings.Count(string(got), ".claude/skills/aiwf-*/"); c != 1 {
		t.Errorf("skill wildcard appears %d times, want 1\n%s", c, got)
	}
	if c := strings.Count(string(got), ".claude/skills/.aiwf-owned"); c != 1 {
		t.Errorf("manifest entry appears %d times, want 1\n%s", c, got)
	}
}

// TestInit_GitignoreFutureProof: a consumer with a pre-existing
// .gitignore that already covers the wildcard should not get the
// wildcard appended a second time, even if their .gitignore predates
// G19. Confirms the future-proof property: once the wildcard is in
// place, adding a new aiwf-* skill to the embedded set does not require
// the consumer to re-run aiwf init.
func TestInit_GitignoreFutureProof(t *testing.T) {
	root := freshGitRepo(t)
	existing := []byte("# pre-existing\n.claude/skills/aiwf-*/\n.claude/skills/.aiwf-owned\n")
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), existing, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(root, ".gitignore"))
	if c := strings.Count(string(got), ".claude/skills/aiwf-*/"); c != 1 {
		t.Errorf("wildcard appended despite already being present; appears %d times\n%s", c, got)
	}
}

// TestInit_GitignoreHTMLOutDir_DefaultIsIgnored: a fresh init lands
// `site/` in the gitignore so the renderer's default output is
// invisible to git unless the consumer flips html.commit_output: true.
func TestInit_GitignoreHTMLOutDir_DefaultIsIgnored(t *testing.T) {
	root := freshGitRepo(t)
	if _, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(root, ".gitignore"))
	if !strings.Contains(string(got), "\nsite/\n") && !strings.HasSuffix(string(got), "site/\n") {
		t.Errorf("default render out_dir 'site/' missing from .gitignore:\n%s", got)
	}
}

// TestInit_GitignoreHTMLOutDir_CommitOutputTrue: with
// html.commit_output: true in aiwf.yaml, init does not add the
// out_dir line — the consumer wants to commit the rendered files.
func TestInit_GitignoreHTMLOutDir_CommitOutputTrue(t *testing.T) {
	root := freshGitRepo(t)
	yamlPath := filepath.Join(root, "aiwf.yaml")
	if err := os.WriteFile(yamlPath, []byte("aiwf_version: 0.1.0\nhtml:\n  out_dir: docs/site\n  commit_output: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(root, ".gitignore"))
	if strings.Contains(string(got), "docs/site/") || strings.Contains(string(got), "site/") {
		t.Errorf("expected no html out_dir gitignore entry under commit_output: true:\n%s", got)
	}
}

// TestInit_GitignoreHTMLOutDir_FlipFalseToTrue: a previous run that
// landed `site/` in .gitignore must be reconciled when the consumer
// flips html.commit_output to true on the next init/update — the
// stale line is removed.
func TestInit_GitignoreHTMLOutDir_FlipFalseToTrue(t *testing.T) {
	root := freshGitRepo(t)
	// First pass: default config → site/ ends up in .gitignore.
	if _, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	gi, _ := os.ReadFile(filepath.Join(root, ".gitignore"))
	if !strings.Contains(string(gi), "site/") {
		t.Fatalf("expected site/ after fresh init; got:\n%s", gi)
	}

	// Flip aiwf.yaml to commit_output: true and re-init.
	yamlPath := filepath.Join(root, "aiwf.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	patched := string(raw) + "html:\n  commit_output: true\n"
	if err := os.WriteFile(yamlPath, []byte(patched), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"}); err != nil {
		t.Fatalf("Init (flip): %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(root, ".gitignore"))
	if strings.Contains(string(got), "\nsite/\n") || strings.HasSuffix(string(got), "site/\n") {
		t.Errorf("site/ still present after commit_output: true flip:\n%s", got)
	}
}

// TestInit_GitignoreHTMLOutDir_FlipTrueToFalse: a consumer who had
// commit_output: true and decides to ungitignore the output gets
// `site/` re-added on next init.
func TestInit_GitignoreHTMLOutDir_FlipTrueToFalse(t *testing.T) {
	root := freshGitRepo(t)
	yamlPath := filepath.Join(root, "aiwf.yaml")
	if err := os.WriteFile(yamlPath, []byte("aiwf_version: 0.1.0\nhtml:\n  commit_output: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	gi, _ := os.ReadFile(filepath.Join(root, ".gitignore"))
	if strings.Contains(string(gi), "site/") {
		t.Fatalf("unexpected site/ under commit_output: true:\n%s", gi)
	}

	// Flip back to false (default).
	if err := os.WriteFile(yamlPath, []byte("aiwf_version: 0.1.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"}); err != nil {
		t.Fatalf("Init (flip back): %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(root, ".gitignore"))
	if !strings.Contains(string(got), "site/") {
		t.Errorf("expected site/ after commit_output: false flip back:\n%s", got)
	}
}

// TestInit_GitignoreHTMLOutDir_PreservesUserDir: a user-authored
// directory entry in .gitignore must survive every reconciliation
// path. The reconciler only matches the configured out_dir or the
// default; arbitrary user content is untouched.
func TestInit_GitignoreHTMLOutDir_PreservesUserDir(t *testing.T) {
	root := freshGitRepo(t)
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("node_modules/\nbuild/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	yamlPath := filepath.Join(root, "aiwf.yaml")
	if err := os.WriteFile(yamlPath, []byte("aiwf_version: 0.1.0\nhtml:\n  commit_output: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(root, ".gitignore"))
	for _, want := range []string{"node_modules/", "build/"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("user-authored entry %q lost from .gitignore:\n%s", want, got)
		}
	}
}

// TestInit_DryRun reports the would-be ledger but writes nothing.
// A second non-dry-run pass on the same repo must still treat it as
// fresh (i.e. dry-run leaves no side effects).
func TestInit_DryRun(t *testing.T) {
	root := freshGitRepo(t)
	res, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0", DryRun: true})
	if err != nil {
		t.Fatalf("Init dry-run: %v", err)
	}
	if !res.DryRun {
		t.Errorf("Result.DryRun = false, want true")
	}
	if len(res.Steps) == 0 {
		t.Fatal("expected non-empty step ledger from dry-run")
	}
	// Ledger reports actions as if writes happened.
	for _, s := range res.Steps {
		if s.Action == "" {
			t.Errorf("step %q has empty action", s.What)
		}
	}
	// No artifacts on disk.
	for _, p := range []string{
		config.FileName,
		"CLAUDE.md",
		".gitignore",
		filepath.Join("work", "epics"),
		filepath.Join(".claude", "skills", "aiwf-add", "SKILL.md"),
		filepath.Join(".git", "hooks", "pre-push"),
	} {
		if _, sErr := os.Stat(filepath.Join(root, p)); !os.IsNotExist(sErr) {
			t.Errorf("dry-run wrote %s (stat err=%v); should be untouched", p, sErr)
		}
	}
	// Real init still runs cleanly afterwards.
	res2, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"})
	if err != nil {
		t.Fatalf("Init after dry-run: %v", err)
	}
	if res2.DryRun {
		t.Errorf("second pass DryRun = true, want false")
	}
	if _, sErr := os.Stat(filepath.Join(root, config.FileName)); sErr != nil {
		t.Errorf("aiwf.yaml missing after real init: %v", sErr)
	}
}

// TestInit_SkipHook: every step except hook installation runs; the
// hook step is reported as Skipped with a clear detail; HookConflict
// is not set (skipping is by user request, not a conflict).
func TestInit_SkipHook(t *testing.T) {
	root := freshGitRepo(t)
	res, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0", SkipHook: true})
	if err != nil {
		t.Fatalf("Init --skip-hook: %v", err)
	}
	if res.HookConflict {
		t.Errorf("HookConflict = true on --skip-hook; want false (skip is requested, not a conflict)")
	}
	// All other artifacts present.
	for _, p := range []string{
		config.FileName,
		"CLAUDE.md",
		".gitignore",
		filepath.Join(".claude", "skills", "aiwf-add", "SKILL.md"),
	} {
		if _, sErr := os.Stat(filepath.Join(root, p)); sErr != nil {
			t.Errorf("expected %s to exist after --skip-hook init: %v", p, sErr)
		}
	}
	// Both hooks absent.
	for _, h := range []string{"pre-push", "pre-commit"} {
		if _, sErr := os.Stat(filepath.Join(root, ".git", "hooks", h)); !os.IsNotExist(sErr) {
			t.Errorf("%s hook installed despite --skip-hook (stat err=%v)", h, sErr)
		}
	}
	// Both hook steps marked Skipped with a --skip-hook detail.
	for _, what := range []string{".git/hooks/pre-push", ".git/hooks/pre-commit"} {
		step := findStep(t, res.Steps, what)
		if step.Action != ActionSkipped {
			t.Errorf("%s step.Action = %q, want %q", what, step.Action, ActionSkipped)
		}
		if !strings.Contains(step.Detail, "skip-hook") {
			t.Errorf("%s step.Detail = %q, want a reference to --skip-hook", what, step.Detail)
		}
	}
}

// TestInit_DryRunWithSkipHook combines both flags. The hook step is
// reported skipped (not "would-create"), and nothing is written.
func TestInit_DryRunWithSkipHook(t *testing.T) {
	root := freshGitRepo(t)
	res, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0", DryRun: true, SkipHook: true})
	if err != nil {
		t.Fatalf("Init dry-run + skip-hook: %v", err)
	}
	if !res.DryRun {
		t.Errorf("Result.DryRun = false, want true")
	}
	prePushStep := findStep(t, res.Steps, ".git/hooks/pre-push")
	if prePushStep.Action != ActionSkipped {
		t.Errorf("pre-push step.Action = %q, want %q", prePushStep.Action, ActionSkipped)
	}
	preCommitStep := findStep(t, res.Steps, ".git/hooks/pre-commit")
	if preCommitStep.Action != ActionSkipped {
		t.Errorf("pre-commit step.Action = %q, want %q", preCommitStep.Action, ActionSkipped)
	}
	if _, sErr := os.Stat(filepath.Join(root, config.FileName)); !os.IsNotExist(sErr) {
		t.Errorf("dry-run wrote aiwf.yaml (stat err=%v)", sErr)
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
