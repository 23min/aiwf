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

	// Synthetic artifact for the contract verb's --artifact-source.
	artifact := filepath.Join(tmp, "schema.json")
	if err := os.WriteFile(artifact, []byte(`{"hello":"self-check"}`), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "aiwf doctor --self-check: %v\n", err)
		keep = true
		return exitInternal
	}

	steps := []struct {
		label string
		args  []string
	}{
		{"init", []string{"init", "--root", tmp, "--actor", actor}},
		{"add epic", []string{"add", "epic", "--title", "Self-check epic", "--actor", actor, "--root", tmp}},
		{"add milestone", []string{"add", "milestone", "--epic", "E-01", "--title", "Schema", "--actor", actor, "--root", tmp}},
		{"add adr", []string{"add", "adr", "--title", "Use Postgres", "--actor", actor, "--root", tmp}},
		{"add gap", []string{"add", "gap", "--title", "Auth gap", "--discovered-in", "M-001", "--actor", actor, "--root", tmp}},
		{"add decision", []string{"add", "decision", "--title", "Sunset v1", "--actor", actor, "--root", tmp}},
		{"add contract", []string{"add", "contract", "--title", "Public API", "--format", "json-schema", "--artifact-source", artifact, "--actor", actor, "--root", tmp}},
		{"promote", []string{"promote", "--actor", actor, "--root", tmp, "E-01", "active"}},
		{"cancel", []string{"cancel", "--actor", actor, "--root", tmp, "G-001"}},
		{"rename", []string{"rename", "--actor", actor, "--root", tmp, "E-01", "self-check-renamed"}},
		{"reallocate", []string{"reallocate", "--actor", actor, "--root", tmp, "E-01"}},
		{"history", []string{"history", "--root", tmp, "E-02"}},
		{"render roadmap", []string{"render", "roadmap", "--root", tmp}},
		{"update", []string{"update", "--root", tmp}},
		{"check", []string{"check", "--root", tmp}},
		{"doctor", []string{"doctor", "--root", tmp}},
	}

	fmt.Printf("self-check repo: %s\n\n", tmp)

	for i, s := range steps {
		rc, captured := runCaptured(s.args)
		if rc == exitOK {
			fmt.Printf("  ok    %s\n", s.label)
			continue
		}
		fmt.Printf("  FAIL  %s (rc=%d)\n", s.label, rc)
		if captured != "" {
			fmt.Println(indent(captured, "        "))
		}
		// Stop at the first failure: later verbs build on earlier state,
		// so cascading failures aren't useful.
		fmt.Printf("\nself-check failed at step %d/%d.\nRepo retained at %s for inspection.\n", i+1, len(steps), tmp)
		keep = true
		return exitFindings
	}
	fmt.Printf("\nself-check passed (%d steps).\n", len(steps))
	return exitOK
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
