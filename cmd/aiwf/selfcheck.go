package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/gitops"
)

// runSelfCheck drives every aiwf verb end-to-end against a throwaway
// repo. It exists to answer "is my install actually working?" without
// needing a real consumer repo. On success, the temp repo is deleted;
// on failure, it is retained and the path is printed so the user can
// inspect what went wrong.
//
// Each step is run through the same `run` dispatcher real users hit,
// so a self-check failure points at the same code path the user would
// trip over.
func runSelfCheck() int {
	const actor = "human/self-check"

	tmp, err := os.MkdirTemp("", "aiwf-self-check-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf doctor --self-check: %v\n", err)
		return exitInternal
	}
	keep := false
	defer func() {
		if keep {
			return
		}
		_ = os.RemoveAll(tmp)
	}()

	// Force GOPROXY=off for the duration of self-check so the
	// upgrade-check and check-latest steps are deterministic offline
	// (CI environments often have no network). Saved and restored so
	// the unit-test process that wraps run() is not polluted.
	prevGOPROXY, hadGOPROXY := os.LookupEnv("GOPROXY")
	_ = os.Setenv("GOPROXY", "off")
	defer func() {
		if hadGOPROXY {
			_ = os.Setenv("GOPROXY", prevGOPROXY)
		} else {
			_ = os.Unsetenv("GOPROXY")
		}
	}()

	// Redirect HOME to a fresh temp dir for the duration of the
	// self-check so the M-070 recommended-plugins steps below can
	// construct a synthetic `~/.claude/plugins/installed_plugins.json`
	// without leaking into the operator's real home dir. Saved and
	// restored on exit. Doctor is the only verb that consults $HOME
	// today; redirecting it here keeps the rest of the self-check
	// unaffected.
	fakeHome, err := os.MkdirTemp("", "aiwf-self-check-home-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf doctor --self-check: %v\n", err)
		return exitInternal
	}
	defer func() { _ = os.RemoveAll(fakeHome) }()
	// Seed a synthetic ~/.gitconfig so identity-resolution
	// (`exec.Command("git", "config", "user.email")` in resolveActor)
	// still produces a valid actor under the redirected HOME — every
	// step that runs a mutating verb depends on this.
	gitconfig := []byte("[user]\n\temail = self-check@aiwf.local\n\tname = aiwf self-check\n")
	if err := os.WriteFile(filepath.Join(fakeHome, ".gitconfig"), gitconfig, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "aiwf doctor --self-check: %v\n", err)
		return exitInternal
	}
	prevHome, hadHome := os.LookupEnv("HOME")
	_ = os.Setenv("HOME", fakeHome)
	defer func() {
		if hadHome {
			_ = os.Setenv("HOME", prevHome)
		} else {
			_ = os.Unsetenv("HOME")
		}
	}()

	ctx := context.Background()
	if err := gitops.Init(ctx, tmp); err != nil {
		fmt.Fprintf(os.Stderr, "aiwf doctor --self-check: git init: %v\n", err)
		keep = true
		return exitInternal
	}
	if err := setLocalGitIdentity(ctx, tmp); err != nil {
		fmt.Fprintf(os.Stderr, "aiwf doctor --self-check: %v\n", err)
		keep = true
		return exitInternal
	}

	preCommitHook := filepath.Join(tmp, ".git", "hooks", "pre-commit")

	// Each step optionally carries a `setup` hook that runs before
	// the CLI invocation, a `verify` hook that runs after (used to
	// assert filesystem state the verb's exit code alone doesn't
	// capture), and a `verifyOutput` hook that receives the verb's
	// captured stdout/stderr so an output substring can be asserted.
	steps := []struct {
		label        string
		args         []string
		setup        func() error
		verify       func() error
		verifyOutput func(string) error
	}{
		{label: "init", args: []string{"init", "--root", tmp, "--actor", actor}},
		{label: "whoami", args: []string{"whoami", "--root", tmp}},
		{label: "add epic", args: []string{"add", "epic", "--title", "Self-check epic", "--actor", actor, "--root", tmp}},
		{label: "add milestone", args: []string{"add", "milestone", "--epic", "E-01", "--tdd", "none", "--title", "Schema", "--actor", actor, "--root", tmp}},
		{label: "add adr", args: []string{"add", "adr", "--title", "Use Postgres", "--actor", actor, "--root", tmp}},
		{label: "add gap", args: []string{"add", "gap", "--title", "Auth gap", "--discovered-in", "M-001", "--actor", actor, "--root", tmp}},
		{label: "add decision", args: []string{"add", "decision", "--title", "Sunset v1", "--actor", actor, "--root", tmp}},
		{label: "add contract", args: []string{"add", "contract", "--title", "Public API", "--actor", actor, "--root", tmp}},
		{label: "promote", args: []string{"promote", "--actor", actor, "--root", tmp, "E-01", "active"}},
		{label: "cancel", args: []string{"cancel", "--actor", actor, "--root", tmp, "G-001"}},
		{label: "rename", args: []string{"rename", "--actor", actor, "--root", tmp, "E-01", "self-check-renamed"}},
		{label: "reallocate", args: []string{"reallocate", "--actor", actor, "--root", tmp, "E-01"}},
		{label: "add move-target epic", args: []string{"add", "epic", "--title", "Move target", "--actor", actor, "--root", tmp}},
		{label: "move", args: []string{"move", "--actor", actor, "--root", tmp, "--epic", "E-03", "M-001"}},
		{label: "history", args: []string{"history", "--root", tmp, "E-02"}},
		{label: "status", args: []string{"status", "--root", tmp}},
		{label: "render roadmap", args: []string{"render", "roadmap", "--root", tmp}},
		{
			label: "update (default install)",
			args:  []string{"update", "--root", tmp},
			verify: func() error {
				if _, err := os.Stat(preCommitHook); err != nil {
					return fmt.Errorf("pre-commit hook should exist after default update: %w", err)
				}
				return nil
			},
		},
		{
			label: "update (status_md.auto_update: false → keeps gate, drops regen)",
			args:  []string{"update", "--root", tmp},
			setup: func() error {
				return rewriteAiwfYAMLAutoUpdate(tmp, false)
			},
			verify: func() error {
				body, err := os.ReadFile(preCommitHook)
				if err != nil {
					return fmt.Errorf("pre-commit hook should remain installed under G42 (gate is enforcement): %w", err)
				}
				if !strings.Contains(string(body), "check --shape-only") {
					return fmt.Errorf("pre-commit hook lost the tree-discipline gate after opt-out:\n%s", body)
				}
				if strings.Contains(string(body), "status --root") {
					return fmt.Errorf("pre-commit hook still includes STATUS.md regen after opt-out:\n%s", body)
				}
				return nil
			},
		},
		{
			label: "update (status_md.auto_update: true → reinstates regen)",
			args:  []string{"update", "--root", tmp},
			setup: func() error {
				return rewriteAiwfYAMLAutoUpdate(tmp, true)
			},
			verify: func() error {
				body, err := os.ReadFile(preCommitHook)
				if err != nil {
					return fmt.Errorf("pre-commit hook missing after re-opt-in: %w", err)
				}
				if !strings.Contains(string(body), "status --root") {
					return fmt.Errorf("pre-commit hook missing STATUS.md regen after re-opt-in:\n%s", body)
				}
				return nil
			},
		},
		{label: "check", args: []string{"check", "--root", tmp}},
		// G33: end-to-end coverage of the audit-only recovery loop.
		// Adds a fresh gap, then synthesizes a manual untrailered
		// commit that flips its status, then asserts `aiwf check`
		// surfaces the warning, then runs `aiwf cancel --audit-only`
		// to backfill, then asserts the warning has cleared. The
		// `--since` flag scopes the audit deterministically without
		// requiring a bare-repo upstream in the self-check fixture.
		{
			label: "add gap (audit-only fixture)",
			args:  []string{"add", "gap", "--title", "audit-only fixture", "--actor", actor, "--root", tmp},
		},
		{
			label: "audit-only fixture: synthetic untrailered flip + check fires",
			setup: func() error {
				return synthesizeUntrailedFlip(ctx, tmp, "G-002", "wontfix")
			},
			args: []string{"check", "--root", tmp, "--since", "HEAD~2"},
			verifyOutput: func(out string) error {
				if !strings.Contains(out, "provenance-untrailered-entity-commit") {
					return fmt.Errorf("expected provenance-untrailered-entity-commit to fire after manual flip; got:\n%s", out)
				}
				// Canonical-width id per AC-3 — display surfaces emit
				// the canonical form regardless of input width.
				if !strings.Contains(out, "G-0002") {
					return fmt.Errorf("warning should name G-0002 as the affected entity; got:\n%s", out)
				}
				return nil
			},
		},
		{
			label: "audit-only fixture: cancel --audit-only repairs",
			args:  []string{"cancel", "G-002", "--audit-only", "--reason", "self-check audit-only loop", "--actor", actor, "--root", tmp},
		},
		{
			label: "audit-only fixture: check no longer fires for G-002",
			args:  []string{"check", "--root", tmp, "--since", "HEAD~3"},
			verifyOutput: func(out string) error {
				// Other untrailered findings (G-001 was cancelled
				// via the verb, so no warning for it; the only
				// candidate is G-002). The repair must have cleared
				// it; the substring "G-0002 with no aiwf-verb" must
				// be absent.
				if strings.Contains(out, "G-0002 with no aiwf-verb") {
					return fmt.Errorf("audit-only failed to clear G-002 warning; got:\n%s", out)
				}
				return nil
			},
		},
		{label: "doctor", args: []string{"doctor", "--root", tmp}},
		{
			// upgrade --check exercises version.Current() and the
			// proxy-disabled fallback path. The function-level
			// GOPROXY=off makes the "proxy disabled" advisory
			// deterministic offline; an error here means version
			// resolution itself broke, not network.
			label: "upgrade --check",
			args:  []string{"upgrade", "--check", "--root", tmp},
		},
		{
			// doctor --check-latest exercises the opt-in network row;
			// with GOPROXY=off the row renders as "unavailable" and
			// increments no problem counter.
			label: "doctor --check-latest",
			args:  []string{"doctor", "--check-latest", "--root", tmp},
		},
		// M-070/AC-7: end-to-end coverage of the recommended-plugin
		// check. The fake HOME redirect at the top of runSelfCheck
		// scopes the synthetic installed_plugins.json fixture to this
		// process. The synthetic plugin name avoids coupling to real
		// marketplace state.
		{
			label: "doctor recommended-plugins fixture: declare in aiwf.yaml",
			setup: func() error {
				return appendDoctorRecommendedPlugins(tmp, []string{"aiwf-self-check@synthetic-marketplace"})
			},
			args: []string{"doctor", "--root", tmp},
			verifyOutput: func(out string) error {
				if !strings.Contains(out, "recommended-plugin-not-installed") {
					return fmt.Errorf("expected recommended-plugin-not-installed warning before install fixture; got:\n%s", out)
				}
				if !strings.Contains(out, "aiwf-self-check@synthetic-marketplace") {
					return fmt.Errorf("warning should name the synthetic plugin id; got:\n%s", out)
				}
				if !strings.Contains(out, "claude /plugin install aiwf-self-check@synthetic-marketplace") {
					return fmt.Errorf("warning should include install command; got:\n%s", out)
				}
				return nil
			},
		},
		{
			label: "doctor recommended-plugins fixture: warning silent after install",
			setup: func() error {
				return writeInstalledPluginsForSelfCheck(fakeHome, "aiwf-self-check@synthetic-marketplace", tmp)
			},
			args: []string{"doctor", "--root", tmp},
			verifyOutput: func(out string) error {
				if strings.Contains(out, "recommended-plugin-not-installed") {
					return fmt.Errorf("matching install fixture should silence the warning; got:\n%s", out)
				}
				return nil
			},
		},
	}

	fmt.Printf("self-check repo: %s\n\n", tmp)

	for i, s := range steps {
		if s.setup != nil {
			if err := s.setup(); err != nil {
				fmt.Printf("  FAIL  %s (setup: %v)\n", s.label, err)
				fmt.Printf("\nself-check failed at step %d/%d.\nRepo retained at %s for inspection.\n", i+1, len(steps), tmp)
				keep = true
				return exitFindings
			}
		}
		rc, captured := runCaptured(s.args)
		if rc != exitOK {
			fmt.Printf("  FAIL  %s (rc=%d)\n", s.label, rc)
			if captured != "" {
				fmt.Println(indent(captured, "        "))
			}
			// Stop at the first failure: later verbs build on earlier
			// state, so cascading failures aren't useful.
			fmt.Printf("\nself-check failed at step %d/%d.\nRepo retained at %s for inspection.\n", i+1, len(steps), tmp)
			keep = true
			return exitFindings
		}
		if s.verify != nil {
			if err := s.verify(); err != nil {
				fmt.Printf("  FAIL  %s (verify: %v)\n", s.label, err)
				fmt.Printf("\nself-check failed at step %d/%d.\nRepo retained at %s for inspection.\n", i+1, len(steps), tmp)
				keep = true
				return exitFindings
			}
		}
		if s.verifyOutput != nil {
			if err := s.verifyOutput(captured); err != nil {
				fmt.Printf("  FAIL  %s (verifyOutput: %v)\n", s.label, err)
				fmt.Println(indent(captured, "        "))
				fmt.Printf("\nself-check failed at step %d/%d.\nRepo retained at %s for inspection.\n", i+1, len(steps), tmp)
				keep = true
				return exitFindings
			}
		}
		fmt.Printf("  ok    %s\n", s.label)
	}
	fmt.Printf("\nself-check passed (%d steps).\n", len(steps))
	return exitOK
}

// appendDoctorRecommendedPlugins appends a `doctor:` /
// `recommended_plugins:` block to <repo>/aiwf.yaml if absent. Used
// by the M-070 self-check step to drive the new check via the verb
// surface (rather than synthetically calling the helper). Idempotent:
// a second invocation with the same plugin set is a no-op.
func appendDoctorRecommendedPlugins(repo string, plugins []string) error {
	path := filepath.Join(repo, "aiwf.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	if strings.Contains(string(raw), "recommended_plugins:") {
		return nil
	}
	var block strings.Builder
	if !strings.HasSuffix(string(raw), "\n") {
		block.WriteString("\n")
	}
	block.WriteString("doctor:\n  recommended_plugins:\n")
	for _, p := range plugins {
		block.WriteString("    - " + p + "\n")
	}
	out := append([]byte(nil), raw...)
	out = append(out, []byte(block.String())...)
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// writeInstalledPluginsForSelfCheck writes the synthetic
// installed_plugins.json under <home>/.claude/plugins/ with one
// project-scope entry for `plugin` whose `projectPath` matches the
// self-check repo `repo`. The fixture mirrors the real Claude Code
// JSON shape so the matcher exercises the same parse + match path
// it would in a real install.
func writeInstalledPluginsForSelfCheck(home, plugin, repo string) error {
	dir := filepath.Join(home, ".claude", "plugins")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	body := fmt.Sprintf(`{
  "version": 2,
  "plugins": {
    %q: [
      {"scope": "project", "projectPath": %q, "installPath": "/synthetic/cache", "version": "self-check"}
    ]
  }
}`, plugin, repo)
	if err := os.WriteFile(filepath.Join(dir, "installed_plugins.json"), []byte(body), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", dir, err)
	}
	return nil
}

// rewriteAiwfYAMLAutoUpdate rewrites <repo>/aiwf.yaml so that
// status_md.auto_update == auto. Used by the self-check to drive
// the install / uninstall transition through the verb surface
// rather than a synthetic in-memory call.
func rewriteAiwfYAMLAutoUpdate(repo string, auto bool) error {
	path := filepath.Join(repo, "aiwf.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	// The selfcheck repo is initialised by Init in the same run, so
	// aiwf.yaml has the canonical two-line shape (aiwf_version,
	// actor) plus whatever any prior step appended. Any prior
	// status_md block is dropped before the new one is appended.
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	out := make([]string, 0, len(lines)+3)
	skipBlock := false
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "status_md:"):
			skipBlock = true
			continue
		case skipBlock && (strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t")):
			continue
		default:
			skipBlock = false
			out = append(out, line)
		}
	}
	out = append(out, "status_md:", fmt.Sprintf("  auto_update: %t", auto), "")
	if err := os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// synthesizeUntrailedFlip simulates the G24 hand-edit case: the
// gap with id `gapID` has its frontmatter status flipped to the
// named target via direct file edit and a plain `git commit` —
// no aiwf trailers — so the next `aiwf check` surfaces the
// untrailered-entity warning. Used by the G33 self-check step
// that drives the audit-only recovery loop end to end.
//
// The gap file lives at `work/gaps/<gapID>-<slug>.md`; we glob
// the directory to find the slug rather than reproducing the
// kernel's slug-derivation rules here.
func synthesizeUntrailedFlip(ctx context.Context, repo, gapID, target string) error {
	gapDir := filepath.Join(repo, "work", "gaps")
	entries, err := os.ReadDir(gapDir)
	if err != nil {
		return fmt.Errorf("reading %s: %w", gapDir, err)
	}
	// Match candidate filenames by canonical id so a self-check
	// driven with a narrow legacy id (`G-002`) finds the canonical
	// on-disk file (`G-0002-...`) per AC-2's parser-tolerance rule.
	canonGap := entity.Canonicalize(gapID)
	var path string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		// Reuse entity.IDFromPath against the gap's repo-relative
		// path; canonicalizing both sides handles the width mismatch.
		relPath := filepath.Join("work", "gaps", name)
		idPortion, ok := entity.IDFromPath(relPath, entity.KindGap)
		if !ok {
			continue
		}
		if entity.Canonicalize(idPortion) == canonGap {
			path = filepath.Join(gapDir, name)
			break
		}
	}
	if path == "" {
		return fmt.Errorf("no gap file matching %s-* under %s", gapID, gapDir)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	flipped := strings.Replace(string(raw), "status: open", "status: "+target, 1)
	if flipped == string(raw) {
		return fmt.Errorf("expected to flip status: open in %s; file unchanged", path)
	}
	if wErr := os.WriteFile(path, []byte(flipped), 0o644); wErr != nil {
		return fmt.Errorf("writing %s: %w", path, wErr)
	}
	relPath, err := filepath.Rel(repo, path)
	if err != nil {
		return fmt.Errorf("rel path: %w", err)
	}
	for _, args := range [][]string{
		{"add", relPath},
		{"commit", "-m", "manual: flip " + gapID + " " + target},
	} {
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

// runCaptured invokes run(args) with os.Stdout and os.Stderr swapped
// out for an in-memory pipe so the caller can decide whether to echo
// the verb's chatter. Returns the exit code and the combined output.
func runCaptured(args []string) (rc int, output string) {
	r, w, err := os.Pipe()
	if err != nil {
		return exitInternal, fmt.Sprintf("os.Pipe: %v", err)
	}
	origOut, origErr := os.Stdout, os.Stderr
	os.Stdout = w
	os.Stderr = w

	done := make(chan []byte, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.Bytes()
	}()

	rc = run(args)

	_ = w.Close()
	os.Stdout = origOut
	os.Stderr = origErr
	out := <-done
	return rc, string(out)
}

// setLocalGitIdentity writes a local user.email/user.name to the
// throwaway repo. Doing this in repo-local config (not the user's
// global config or process env) keeps the rest of the system
// untouched.
func setLocalGitIdentity(ctx context.Context, repo string) error {
	for _, args := range [][]string{
		{"config", "user.email", "self-check@aiwf.local"},
		{"config", "user.name", "aiwf self-check"},
	} {
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

// indent prefixes every line of s with prefix.
func indent(s, prefix string) string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}
