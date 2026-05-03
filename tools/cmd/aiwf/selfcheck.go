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

	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
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

	// Each step optionally carries a `setup` hook that runs before the
	// CLI invocation (used by the pre-commit-hook transition steps to
	// mutate aiwf.yaml between updates) and a `verify` hook that runs
	// after (used to assert filesystem state the verb's exit code
	// alone doesn't capture).
	steps := []struct {
		label  string
		args   []string
		setup  func() error
		verify func() error
	}{
		{label: "init", args: []string{"init", "--root", tmp, "--actor", actor}},
		{label: "whoami", args: []string{"whoami", "--root", tmp}},
		{label: "add epic", args: []string{"add", "epic", "--title", "Self-check epic", "--actor", actor, "--root", tmp}},
		{label: "add milestone", args: []string{"add", "milestone", "--epic", "E-01", "--title", "Schema", "--actor", actor, "--root", tmp}},
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
			label: "update (status_md.auto_update: false → uninstalls hook)",
			args:  []string{"update", "--root", tmp},
			setup: func() error {
				return rewriteAiwfYAMLAutoUpdate(tmp, false)
			},
			verify: func() error {
				if _, err := os.Stat(preCommitHook); !os.IsNotExist(err) {
					return fmt.Errorf("pre-commit hook should be removed when opt-out flag is set (stat err=%v)", err)
				}
				return nil
			},
		},
		{
			label: "update (status_md.auto_update: true → reinstalls hook)",
			args:  []string{"update", "--root", tmp},
			setup: func() error {
				return rewriteAiwfYAMLAutoUpdate(tmp, true)
			},
			verify: func() error {
				if _, err := os.Stat(preCommitHook); err != nil {
					return fmt.Errorf("pre-commit hook should be reinstalled after re-opt-in: %w", err)
				}
				return nil
			},
		},
		{label: "check", args: []string{"check", "--root", tmp}},
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
		fmt.Printf("  ok    %s\n", s.label)
	}
	fmt.Printf("\nself-check passed (%d steps).\n", len(steps))
	return exitOK
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
