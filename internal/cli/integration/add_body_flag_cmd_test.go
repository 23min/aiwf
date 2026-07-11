package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// setupAddBodyFlagRepo is the shared git+aiwf-init fixture for this
// file's tests — same shape as TestAdd_BodyFile_BinaryEndToEnd's setup.
func setupAddBodyFlagRepo(t *testing.T) (root, binDir string) {
	t.Helper()
	bin := testutil.AiwfBinary(t)
	binDir = filepath.Dir(bin)
	root = t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q", "-b", "main"); err != nil {
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
	return root, binDir
}

// TestAdd_BodyFlag_BinaryEndToEnd: G-0326 AC — --body "<text>" lands
// as the entity's body in the same atomic create commit, exercised
// against the real binary and dispatcher (not just internal/verb).
func TestAdd_BodyFlag_BinaryEndToEnd(t *testing.T) {
	t.Parallel()
	root, binDir := setupAddBodyFlagRepo(t)

	out, err := testutil.RunBin(t, root, binDir, nil,
		"add", "gap", "--title", "Retry loop spins forever", "--body",
		"## What's missing\n\nRetrying a failed fetch has no backoff cap.\n\n## Why it matters\n\nThe process spins forever.\n")
	if err != nil {
		t.Fatalf("aiwf add --body: %v\n%s", err, out)
	}

	matches, err := filepath.Glob(filepath.Join(root, "work", "gaps", "G-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("glob G-*.md: matches=%v err=%v", matches, err)
	}
	got, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read gap: %v", err)
	}
	if !strings.Contains(string(got), "no backoff cap") {
		t.Errorf("gap missing --body content:\n%s", got)
	}
	if !strings.Contains(string(got), "id: G-0001") {
		t.Errorf("gap missing serialized frontmatter:\n%s", got)
	}
}

// TestAdd_BodyFlag_MutuallyExclusiveWithBodyFile: passing both --body
// and --body-file is a usage error (exit 2); no entity is created.
func TestAdd_BodyFlag_MutuallyExclusiveWithBodyFile(t *testing.T) {
	t.Parallel()
	root, binDir := setupAddBodyFlagRepo(t)

	bodyPath := filepath.Join(root, "gap-body.md")
	if err := os.WriteFile(bodyPath, []byte("## What's missing\n\nX.\n\n## Why it matters\n\nX.\n"), 0o644); err != nil {
		t.Fatalf("write body file: %v", err)
	}

	out, err := testutil.RunBin(t, root, binDir, nil,
		"add", "gap", "--title", "Both flags", "--body", "inline text", "--body-file", bodyPath)
	if err == nil {
		t.Fatalf("expected mutual-exclusivity refusal; got:\n%s", out)
	}
	if !strings.Contains(out, "mutually exclusive") {
		t.Errorf("expected a mutually-exclusive message; got:\n%s", out)
	}
	matches, _ := filepath.Glob(filepath.Join(root, "work", "gaps", "G-*.md"))
	if len(matches) != 0 {
		t.Errorf("entity created despite refusal: %v", matches)
	}
}

// TestAdd_EmptyBodyGate_ForceWithoutReasonRefused_BinaryEndToEnd:
// --force with no --reason (or an all-whitespace one) is a usage
// error; no entity is created.
func TestAdd_EmptyBodyGate_ForceWithoutReasonRefused_BinaryEndToEnd(t *testing.T) {
	t.Parallel()
	root, binDir := setupAddBodyFlagRepo(t)

	out, err := testutil.RunBin(t, root, binDir, nil,
		"add", "gap", "--title", "Forced but reasonless", "--force")
	if err == nil {
		t.Fatalf("expected --reason-required refusal; got:\n%s", out)
	}
	if !strings.Contains(out, "--reason") {
		t.Errorf("expected a --reason-required message; got:\n%s", out)
	}
	matches, _ := filepath.Glob(filepath.Join(root, "work", "gaps", "G-*.md"))
	if len(matches) != 0 {
		t.Errorf("entity created despite refusal: %v", matches)
	}
}

// TestAdd_EmptyBodyGate_ForceWithReasonBypasses_BinaryEndToEnd:
// --force --reason "..." on a born-complete kind with no body content
// at all creates the entity anyway (sovereign override), and the
// create commit's aiwf-force trailer carries the reason.
func TestAdd_EmptyBodyGate_ForceWithReasonBypasses_BinaryEndToEnd(t *testing.T) {
	t.Parallel()
	root, binDir := setupAddBodyFlagRepo(t)

	out, err := testutil.RunBin(t, root, binDir, nil,
		"add", "gap", "--title", "Forced through", "--force", "--reason", "deliberately deferring the writeup")
	if err != nil {
		t.Fatalf("aiwf add --force --reason: %v\n%s", err, out)
	}
	matches, err := filepath.Glob(filepath.Join(root, "work", "gaps", "G-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("glob G-*.md: matches=%v err=%v", matches, err)
	}

	trailerOut, err := testutil.RunGit(root, "log", "-1", "--format=%B")
	if err != nil {
		t.Fatalf("git log: %v\n%s", err, trailerOut)
	}
	if !strings.Contains(trailerOut, "aiwf-force: deliberately deferring the writeup") {
		t.Errorf("HEAD commit missing aiwf-force trailer:\n%s", trailerOut)
	}
}

// TestAdd_EmptyBodyGate_DefaultTemplateRefused_BinaryEndToEnd:
// `aiwf add gap --title "..."` with no --body/--body-file/--force at
// all refuses (the bare per-kind template is headings-only, which is
// exactly what the gate targets) — the end-to-end regression guard
// for G-0326/AC-1 against the real dispatcher.
func TestAdd_EmptyBodyGate_DefaultTemplateRefused_BinaryEndToEnd(t *testing.T) {
	t.Parallel()
	root, binDir := setupAddBodyFlagRepo(t)

	out, err := testutil.RunBin(t, root, binDir, nil,
		"add", "gap", "--title", "Untitled gap")
	if err == nil {
		t.Fatalf("expected empty-body refusal; got:\n%s", out)
	}
	if !strings.Contains(out, "What's missing") || !strings.Contains(out, "Why it matters") {
		t.Errorf("expected both empty sections named; got:\n%s", out)
	}
	matches, _ := filepath.Glob(filepath.Join(root, "work", "gaps", "G-*.md"))
	if len(matches) != 0 {
		t.Errorf("entity created despite refusal: %v", matches)
	}
}
