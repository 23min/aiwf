package policies_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoRootForHook(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}

func hookScriptPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRootForHook(t), ".claude", "hooks", "validate-agent-isolation.sh")
}

func runIsolationHook(t *testing.T, hookInputJSON string) (stdout []byte, exitCode int) {
	t.Helper()
	cmd := exec.Command(hookScriptPath(t))
	cmd.Stdin = strings.NewReader(hookInputJSON)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return out.Bytes(), exitErr.ExitCode()
		}
		t.Fatalf("hook script run failed: %v", err)
	}
	return out.Bytes(), 0
}

type isolationHookResponse struct {
	HookSpecificOutput struct {
		HookEventName            string `json:"hookEventName"`
		PermissionDecision       string `json:"permissionDecision"`
		PermissionDecisionReason string `json:"permissionDecisionReason"`
	} `json:"hookSpecificOutput"`
}

func TestAgentIsolationHook_DeniesWorktreeIsolation(t *testing.T) {
	t.Parallel()
	input := `{"tool_name":"Agent","tool_input":{"prompt":"do work","isolation":"worktree","subagent_type":"general-purpose"}}`
	stdout, exitCode := runIsolationHook(t, input)
	if exitCode != 0 {
		t.Fatalf("expected exit 0 (decision is in stdout JSON), got %d; stdout=%s", exitCode, stdout)
	}
	var resp isolationHookResponse
	if err := json.Unmarshal(stdout, &resp); err != nil {
		t.Fatalf("expected JSON output on stdout, got: %s (err: %v)", stdout, err)
	}
	if got, want := resp.HookSpecificOutput.HookEventName, "PreToolUse"; got != want {
		t.Errorf("hookEventName = %q, want %q", got, want)
	}
	if got, want := resp.HookSpecificOutput.PermissionDecision, "deny"; got != want {
		t.Errorf("permissionDecision = %q, want %q", got, want)
	}
	reason := resp.HookSpecificOutput.PermissionDecisionReason
	for _, mustContain := range []string{
		"G-0099",
		"git worktree add",
		"git worktree list",
		"CLAUDE.md",
	} {
		if !strings.Contains(reason, mustContain) {
			t.Errorf("permissionDecisionReason should mention %q; got: %q", mustContain, reason)
		}
	}
}

func TestAgentIsolationHook_AllowsWhenIsolationAbsent(t *testing.T) {
	t.Parallel()
	input := `{"tool_name":"Agent","tool_input":{"prompt":"do work","subagent_type":"general-purpose"}}`
	stdout, exitCode := runIsolationHook(t, input)
	if exitCode != 0 {
		t.Fatalf("expected exit 0 (allow), got %d; stdout=%s", exitCode, stdout)
	}
	if got := bytes.TrimSpace(stdout); len(got) > 0 {
		t.Errorf("expected empty stdout for allow path, got: %s", got)
	}
}

func TestAgentIsolationHook_AllowsEmptyIsolationString(t *testing.T) {
	t.Parallel()
	input := `{"tool_name":"Agent","tool_input":{"prompt":"do work","isolation":""}}`
	stdout, exitCode := runIsolationHook(t, input)
	if exitCode != 0 {
		t.Fatalf("expected exit 0 (allow), got %d; stdout=%s", exitCode, stdout)
	}
	if got := bytes.TrimSpace(stdout); len(got) > 0 {
		t.Errorf("expected empty stdout for allow path, got: %s", got)
	}
}

func TestAgentIsolationHook_AllowsUnrelatedIsolationValue(t *testing.T) {
	t.Parallel()
	// Only "worktree" is the documented failure mode in G-0099. Future
	// isolation modes (e.g. "container") shouldn't be policed by this
	// hook — they'd need their own gap + chokepoint.
	input := `{"tool_name":"Agent","tool_input":{"prompt":"do work","isolation":"container"}}`
	stdout, exitCode := runIsolationHook(t, input)
	if exitCode != 0 {
		t.Fatalf("expected exit 0 (allow), got %d; stdout=%s", exitCode, stdout)
	}
	if got := bytes.TrimSpace(stdout); len(got) > 0 {
		t.Errorf("expected empty stdout for allow path, got: %s", got)
	}
}
