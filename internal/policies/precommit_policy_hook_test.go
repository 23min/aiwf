package policies_test

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// precommit_policy_hook_test.go — G-0280 chokepoint pins.
//
// Behavioral subprocess tests for scripts/git-hooks/pre-commit (the
// kernel policy-lint hook installed as .git/hooks/pre-commit.local via
// `make install-hooks`). They pin the G-0280 gate: the (~70s)
// `go test ./internal/policies/...` suite runs only when the commit
// stages a Go/build input, and is skipped for planning-only commits.
// The AIWF_PRECOMMIT_POLICY_CMD override stands in for the real suite —
// `true` (green) / `false` (red) make the "did the suite run?" decision
// observable without running it, mirroring AIWF_PREPUSH_LINT_CMD in
// prepush_lint_hook_test.go.

func precommitHookPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRootForHook(t), "scripts", "git-hooks", "pre-commit")
}

// runPrecommitHook stages the given files in a fresh fixture repo and
// runs the pre-commit hook with the policy-command override, returning
// the exit code and combined stderr. Each stage entry is a
// "relpath=content" pair. The repo's .gitleaks.toml is copied in so the
// hook's (untouched-by-G-0280) gitleaks block has a valid config and
// never false-blocks an exit-0 case on a machine that has gitleaks
// installed; where gitleaks is absent the block self-skips with a
// warning.
func runPrecommitHook(t *testing.T, policyCmd string, stage ...string) (exitCode int, stderr string) {
	t.Helper()
	dir := t.TempDir()
	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	runGit("init", "-q", "-b", "main")

	if cfg, err := os.ReadFile(filepath.Join(repoRootForHook(t), ".gitleaks.toml")); err == nil {
		if err := os.WriteFile(filepath.Join(dir, ".gitleaks.toml"), cfg, 0o644); err != nil {
			t.Fatalf("writing fixture .gitleaks.toml: %v", err)
		}
	}

	for _, pair := range stage {
		rel, content, _ := strings.Cut(pair, "=")
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", rel, err)
		}
	}
	runGit("add", "-A")

	cmd := exec.Command(precommitHookPath(t))
	cmd.Dir = dir
	env := os.Environ()[:0:0]
	for _, kv := range os.Environ() {
		if key, _, _ := strings.Cut(kv, "="); key == "AIWF_PRECOMMIT_POLICY_CMD" {
			continue
		}
		env = append(env, kv)
	}
	env = append(env, "AIWF_PRECOMMIT_POLICY_CMD="+policyCmd)
	cmd.Env = env
	var errb bytes.Buffer
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), errb.String()
		}
		t.Fatalf("hook run failed: %v", err)
	}
	return 0, errb.String()
}

func TestPrecommitPolicyHook_GateDecision(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		policyCmd string // stand-in for `go test ./internal/policies/...`
		stage     []string
		wantExit  int
		wantErr   []string
	}{
		{
			name:      "planning-only commit skips the suite",
			policyCmd: "false", // would exit 1 if the suite ran
			stage:     []string{"work/gaps/G-9999-x.md=planning prose\n", "ROADMAP.md=table\n"},
			wantExit:  0,
		},
		{
			name:      "go file runs the suite, red blocks",
			policyCmd: "false",
			stage:     []string{"foo.go=package main\n"},
			wantExit:  1,
			wantErr:   []string{"policies failed"},
		},
		{
			name:      "go file runs the suite, green passes",
			policyCmd: "true",
			stage:     []string{"foo.go=package main\n"},
			wantExit:  0,
		},
		{
			name:      "go.mod runs the suite",
			policyCmd: "false",
			stage:     []string{"go.mod=module example.test/x\n"},
			wantExit:  1,
		},
		{
			name:      "go.sum runs the suite",
			policyCmd: "false",
			stage:     []string{"go.sum=example.test/x v1.0.0 h1:abc=\n"},
			wantExit:  1,
		},
		{
			name:      "Makefile runs the suite",
			policyCmd: "false",
			stage:     []string{"Makefile=all:\n\t@echo hi\n"},
			wantExit:  1,
		},
		{
			name:      "workflow file runs the suite",
			policyCmd: "false",
			stage:     []string{".github/workflows/ci.yml=name: ci\n"},
			wantExit:  1,
		},
		{
			name:      "embedded skill file runs the suite",
			policyCmd: "false", // skill_coverage et al. read SKILL.md from disk
			stage:     []string{"internal/skills/embedded/plugins/x/skills/y/SKILL.md=---\nname: y\n---\nbody\n"},
			wantExit:  1,
		},
		{
			name:      "hook script runs the suite",
			policyCmd: "false", // the *_hook policy tests read these scripts
			stage:     []string{"scripts/git-hooks/pre-commit=#!/bin/sh\nexit 0\n"},
			wantExit:  1,
		},
		{
			name:      "mixed go + markdown runs the suite",
			policyCmd: "false",
			stage:     []string{"foo.go=package main\n", "doc.md=hi\n"},
			wantExit:  1,
		},
		{
			// wantErr is "policy lint skipped", not bare "skipped" — the
			// gitleaks-absent branch also prints "…skipped" (T2).
			name:      "missing policy tool tolerated with warning",
			policyCmd: "aiwf-no-such-tool-7d3f run",
			stage:     []string{"foo.go=package main\n"},
			wantExit:  0,
			wantErr:   []string{"policy lint skipped"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			exitCode, stderr := runPrecommitHook(t, tt.policyCmd, tt.stage...)
			if exitCode != tt.wantExit {
				t.Errorf("exit = %d, want %d\nstderr: %s", exitCode, tt.wantExit, stderr)
			}
			for _, w := range tt.wantErr {
				if !strings.Contains(stderr, w) {
					t.Errorf("stderr should contain %q; got:\n%s", w, stderr)
				}
			}
		})
	}
}

// TestPrecommitPolicyHook_InstallWiring pins the install path, mirroring
// TestPrepushLintHook_InstallWiring: the tracked script exists, is
// executable, and `make install-hooks` symlinks it into the chain target.
// Drop any of the three and the gate silently stops being installable.
func TestPrecommitPolicyHook_InstallWiring(t *testing.T) {
	t.Parallel()
	root := repoRootForHook(t)

	info, err := os.Stat(precommitHookPath(t))
	if err != nil {
		t.Fatalf("tracked hook script missing: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Errorf("scripts/git-hooks/pre-commit must be executable; mode = %v", info.Mode())
	}

	makefile, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatalf("reading Makefile: %v", err)
	}
	wantLine := `ln -sfn ../../scripts/git-hooks/pre-commit "$$HOOKS_DIR/pre-commit.local"`
	if !strings.Contains(string(makefile), wantLine) {
		t.Errorf("Makefile install-hooks must symlink the pre-commit hook into the chain target; missing line: %s", wantLine)
	}
}
