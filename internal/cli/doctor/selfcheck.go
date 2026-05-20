package doctor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
)

// runSelfCheck drives every aiwf verb end-to-end against a throwaway
// repo. It exists to answer "is my install actually working?" without
// needing a real consumer repo. On success, the temp repo is deleted;
// on failure, it is retained and the path is printed so the user can
// inspect what went wrong.
//
// Each step is run through the in-process Dispatcher (wired by
// cmd/aiwf/main.go) so a self-check failure points at the same code
// path the user would trip over.
func runSelfCheck() int {
	if Dispatcher == nil {
		fmt.Fprintln(os.Stderr, "aiwf doctor --self-check: in-process Dispatcher unset (wiring bug in cmd/aiwf/main.go's init); cannot run")
		return cliutil.ExitInternal
	}
	const actor = "human/self-check"

	tmp, err := os.MkdirTemp("", "aiwf-self-check-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf doctor --self-check: %v\n", err)
		return cliutil.ExitInternal
	}
	keep := false
	defer func() {
		if keep {
			return
		}
		_ = os.RemoveAll(tmp)
	}()

	// Force GOPROXY=off for the duration of self-check so the
	// upgrade-check and check-latest steps are deterministic offline.
	prevGOPROXY, hadGOPROXY := os.LookupEnv("GOPROXY")
	_ = os.Setenv("GOPROXY", "off")
	defer func() {
		if hadGOPROXY {
			_ = os.Setenv("GOPROXY", prevGOPROXY)
		} else {
			_ = os.Unsetenv("GOPROXY")
		}
	}()

	// Redirect HOME to a fresh temp dir so the recommended-plugins
	// steps can construct a synthetic installed_plugins.json without
	// leaking into the operator's real home dir.
	fakeHome, err := os.MkdirTemp("", "aiwf-self-check-home-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf doctor --self-check: %v\n", err)
		return cliutil.ExitInternal
	}
	defer func() { _ = os.RemoveAll(fakeHome) }()
	gitconfig := []byte("[user]\n\temail = self-check@aiwf.local\n\tname = aiwf self-check\n")
	if err := os.WriteFile(filepath.Join(fakeHome, ".gitconfig"), gitconfig, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "aiwf doctor --self-check: %v\n", err)
		return cliutil.ExitInternal
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
		return cliutil.ExitInternal
	}
	if err := setLocalGitIdentity(ctx, tmp); err != nil {
		fmt.Fprintf(os.Stderr, "aiwf doctor --self-check: %v\n", err)
		keep = true
		return cliutil.ExitInternal
	}

	preCommitHook := filepath.Join(tmp, ".git", "hooks", "pre-commit")
	postCommitHook := filepath.Join(tmp, ".git", "hooks", "post-commit")

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
				if _, err := os.Stat(postCommitHook); err != nil {
					return fmt.Errorf("post-commit hook should exist after default update (G-0112): %w", err)
				}
				return nil
			},
		},
		{
			label: "update (status_md.auto_update: false → keeps gate, removes post-commit)",
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
					return fmt.Errorf("pre-commit hook still includes STATUS.md regen (G-0112: regen lives in post-commit):\n%s", body)
				}
				if _, err := os.Stat(postCommitHook); !os.IsNotExist(err) {
					return fmt.Errorf("post-commit hook should be removed under opt-out (G-0112) (stat err=%w)", err)
				}
				return nil
			},
		},
		{
			label: "update (status_md.auto_update: true → reinstates post-commit)",
			args:  []string{"update", "--root", tmp},
			setup: func() error {
				return rewriteAiwfYAMLAutoUpdate(tmp, true)
			},
			verify: func() error {
				if _, err := os.Stat(postCommitHook); err != nil {
					return fmt.Errorf("post-commit hook missing after re-opt-in (G-0112): %w", err)
				}
				body, err := os.ReadFile(postCommitHook)
				if err != nil {
					return fmt.Errorf("read post-commit hook: %w", err)
				}
				if !strings.Contains(string(body), "status --root") {
					return fmt.Errorf("post-commit hook missing STATUS.md regen after re-opt-in:\n%s", body)
				}
				return nil
			},
		},
		{label: "check", args: []string{"check", "--root", tmp}},
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
				if strings.Contains(out, "G-0002 with no aiwf-verb") {
					return fmt.Errorf("audit-only failed to clear G-002 warning; got:\n%s", out)
				}
				return nil
			},
		},
		{label: "doctor", args: []string{"doctor", "--root", tmp}},
		{
			label: "upgrade --check",
			args:  []string{"upgrade", "--check", "--root", tmp},
		},
		{
			label: "doctor --check-latest",
			args:  []string{"doctor", "--check-latest", "--root", tmp},
		},
		{
			label: "doctor recommended-plugins fixture: declare in aiwf.yaml",
			setup: func() error {
				return appendDoctorRecommendedPlugins(tmp, []string{"aiwf-self-check@synthetic-marketplace"})
			},
			args: []string{"doctor", "--root", tmp},
			verifyOutput: func(out string) error {
				if !strings.Contains(out, "recommended-plugin-not-installed") {
					return fmt.Errorf("expected recommended-plugin-not-installed warning before enable fixture; got:\n%s", out)
				}
				if !strings.Contains(out, "aiwf-self-check@synthetic-marketplace") {
					return fmt.Errorf("warning should name the synthetic plugin id; got:\n%s", out)
				}
				if !strings.Contains(out, "PROJECT scope") {
					return fmt.Errorf("warning should include PROJECT-scope install advice (post-G-0138); got:\n%s", out)
				}
				return nil
			},
		},
		{
			label: "doctor recommended-plugins fixture: warning silent after enable in settings.json",
			setup: func() error {
				return writeEnabledPluginsForSelfCheck(tmp, "aiwf-self-check@synthetic-marketplace")
			},
			args: []string{"doctor", "--root", tmp},
			verifyOutput: func(out string) error {
				if strings.Contains(out, "recommended-plugin-not-installed") {
					return fmt.Errorf("enabledPlugins=true should silence the warning; got:\n%s", out)
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
				return cliutil.ExitFindings
			}
		}
		rc, captured := runCaptured(s.args)
		if rc != cliutil.ExitOK {
			fmt.Printf("  FAIL  %s (rc=%d)\n", s.label, rc)
			if captured != "" {
				fmt.Println(indent(captured, "        "))
			}
			fmt.Printf("\nself-check failed at step %d/%d.\nRepo retained at %s for inspection.\n", i+1, len(steps), tmp)
			keep = true
			return cliutil.ExitFindings
		}
		if s.verify != nil {
			if err := s.verify(); err != nil {
				fmt.Printf("  FAIL  %s (verify: %v)\n", s.label, err)
				fmt.Printf("\nself-check failed at step %d/%d.\nRepo retained at %s for inspection.\n", i+1, len(steps), tmp)
				keep = true
				return cliutil.ExitFindings
			}
		}
		if s.verifyOutput != nil {
			if err := s.verifyOutput(captured); err != nil {
				fmt.Printf("  FAIL  %s (verifyOutput: %v)\n", s.label, err)
				fmt.Println(indent(captured, "        "))
				fmt.Printf("\nself-check failed at step %d/%d.\nRepo retained at %s for inspection.\n", i+1, len(steps), tmp)
				keep = true
				return cliutil.ExitFindings
			}
		}
		fmt.Printf("  ok    %s\n", s.label)
	}
	fmt.Printf("\nself-check passed (%d steps).\n", len(steps))
	return cliutil.ExitOK
}

// appendDoctorRecommendedPlugins appends a `doctor:` /
// `recommended_plugins:` block to <repo>/aiwf.yaml if absent.
// Idempotent: a second invocation with the same plugin set is a no-op.
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

// writeEnabledPluginsForSelfCheck writes <repo>/.claude/settings.json
// declaring `plugin` enabled, so the doctor's recommended-plugins
// check (post-G-0138 / M-0133 / AC-3) reads it as satisfied.
func writeEnabledPluginsForSelfCheck(repo, plugin string) error {
	dir := filepath.Join(repo, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	body := fmt.Sprintf(`{%q: {%q: true}}`, "enabledPlugins", plugin)
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(body), 0o644); err != nil {
		return fmt.Errorf("writing settings.json: %w", err)
	}
	return nil
}

// rewriteAiwfYAMLAutoUpdate rewrites <repo>/aiwf.yaml so that
// status_md.auto_update == auto.
func rewriteAiwfYAMLAutoUpdate(repo string, auto bool) error {
	path := filepath.Join(repo, "aiwf.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
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
// no aiwf trailers.
func synthesizeUntrailedFlip(ctx context.Context, repo, gapID, target string) error {
	gapDir := filepath.Join(repo, "work", "gaps")
	entries, err := os.ReadDir(gapDir)
	if err != nil {
		return fmt.Errorf("reading %s: %w", gapDir, err)
	}
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

// runCaptured invokes the in-process Dispatcher with os.Stdout and
// os.Stderr swapped out for an in-memory pipe so the caller can
// decide whether to echo the verb's chatter. Returns the exit code
// and the combined output.
func runCaptured(args []string) (rc int, output string) {
	r, w, err := os.Pipe()
	if err != nil {
		return cliutil.ExitInternal, fmt.Sprintf("os.Pipe: %v", err)
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

	rc = Dispatcher(args)

	_ = w.Close()
	os.Stdout = origOut
	os.Stderr = origErr
	out := <-done
	return rc, string(out)
}

// setLocalGitIdentity writes a local user.email/user.name to the
// throwaway repo.
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
