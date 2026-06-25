package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// setupAreaScopeRepo builds a repo with a declared areas block, one
// epic tagged `platform`, and an ACTIVE scope authorizing ai/claude on
// that epic — opened cleanly on a ritual branch so the M-0103 AI-target
// preflight passes without --force. Returns (root, bin).
//
// The shape mirrors setupTwoEpicScopeRepo (M-0141/AC-2): the human
// (default git-config actor) does setup; the agent later acts as
// `--actor ai/claude --principal human/peter`.
func setupAreaScopeRepo(t *testing.T) (root, bin string) {
	t.Helper()
	bin = testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)
	root = t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := testutil.RunGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}

	// Declare an areas block and commit it before the area-tagged add.
	yamlPath := filepath.Join(root, "aiwf.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	patched := string(raw) + "areas:\n  members:\n    - platform\n    - billing\n"
	if err := os.WriteFile(yamlPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
	if out, err := testutil.RunGit(root, "add", "aiwf.yaml"); err != nil {
		t.Fatalf("git add aiwf.yaml: %v\n%s", err, out)
	}
	if out, err := testutil.RunGit(root, "commit", "-q", "-m", "chore: declare areas"); err != nil {
		t.Fatalf("git commit areas: %v\n%s", err, out)
	}

	if out, err := testutil.RunBin(t, root, binDir, nil,
		"add", "epic", "--title", "Platform", "--area", "platform"); err != nil {
		t.Fatalf("aiwf add epic --area platform: %v\n%s", err, out)
	}

	// M-0103: move HEAD to a ritual-shape branch so the AI-target
	// preflight's implicit-current signal passes when opening the
	// E-0001 scope below (no --force needed).
	if out, err := testutil.RunGit(root, "checkout", "-b", "epic/E-0001-platform"); err != nil {
		t.Fatalf("git checkout -b: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil,
		"authorize", "E-0001", "--to", "ai/claude", "--reason", "implement E-0001"); err != nil {
		t.Fatalf("aiwf authorize E-0001: %v\n%s", err, out)
	}
	return root, bin
}

// TestRenameArea_AuthorizedAIRefused pins the ratified human-only
// provenance posture (E-0044, M-0177 review follow-up): an authorized
// AI agent — one holding an ACTIVE scope on E-0001 — is still refused
// when it runs `aiwf rename-area`, because the verb is a repo-wide
// config mutation with no single target entity (ProvenanceContext
// TargetID is empty), so VerbAct scope-reachability cannot be
// satisfied. The refusal carries the structured
// provenance-authorization-out-of-scope code, exits 1, and leaves both
// aiwf.yaml and HEAD untouched.
//
// REGRESSION GUARD: the scope is real and active, not a bare no-scope
// AI. If a future change sets a non-empty TargetID in renamearea.go's
// Run (e.g. the scoped entity's id), the scoped AI would reach the
// scope and be ALLOWED — this test would then go red (the rename would
// succeed and the out-of-scope assertion would fail). That is the
// regression the pin exists to catch.
func TestRenameArea_AuthorizedAIRefused(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	root, bin := setupAreaScopeRepo(t)

	yamlPath := filepath.Join(root, "aiwf.yaml")
	yamlBefore, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml before: %v", err)
	}
	headBefore, err := testutil.RunGit(root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v\n%s", err, headBefore)
	}

	stdout, stderr, code := runSplit(t, root, bin,
		"rename-area", "platform", "infra",
		"--actor", "ai/claude", "--principal", "human/peter", "--format=json")
	if code != 1 {
		t.Fatalf("authorized-AI rename-area exit = %d, want 1 (refused)\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	if stderr != "" {
		t.Errorf("JSON mode must write nothing to stderr; got:\n%s", stderr)
	}
	var env codedEnvelope
	if jerr := json.Unmarshal([]byte(stdout), &env); jerr != nil {
		t.Fatalf("stdout is not a single JSON envelope: %v\nstdout:\n%s", jerr, stdout)
	}
	if env.Status != "error" {
		t.Errorf("status = %q, want \"error\"", env.Status)
	}
	if env.Error == nil {
		t.Fatalf("envelope has no error object:\n%s", stdout)
	}
	if env.Error.Code != "provenance-authorization-out-of-scope" {
		t.Errorf("error.code = %q, want \"provenance-authorization-out-of-scope\"", env.Error.Code)
	}

	// aiwf.yaml unchanged — the refused mutation writes nothing.
	yamlAfter, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml after: %v", err)
	}
	if !bytes.Equal(yamlAfter, yamlBefore) {
		t.Errorf("aiwf.yaml changed on a refused verb:\n%s", yamlAfter)
	}
	if bytes.Contains(yamlAfter, []byte("infra")) {
		t.Errorf("aiwf.yaml carries the would-be rename:\n%s", yamlAfter)
	}

	// HEAD unchanged — no commit landed.
	headAfter, err := testutil.RunGit(root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse HEAD (after): %v\n%s", err, headAfter)
	}
	if strings.TrimSpace(headBefore) != strings.TrimSpace(headAfter) {
		t.Errorf("HEAD moved on a refused verb: before=%s after=%s",
			strings.TrimSpace(headBefore), strings.TrimSpace(headAfter))
	}
}
