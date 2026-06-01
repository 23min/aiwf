package integration

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// setupTwoEpicScopeRepo builds a repo with two epics, each owning one
// milestone, and an active scope authorizing ai/claude on E-0001:
//
//	E-0001 ── M-0001   (in scope: M-0001 reaches E-0001 via parent)
//	E-0002 ── M-0002   (out of scope: M-0002 reaches E-0002, not E-0001)
//
// Returns (root, bin). The agent acts as `--actor ai/claude --principal
// human/peter`; the human (default actor, from git config) does setup.
func setupTwoEpicScopeRepo(t *testing.T) (root, bin string) {
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
	for _, args := range [][]string{
		{"init"},
		{"add", "epic", "--title", "Platform"},
		{"add", "milestone", "--epic", "E-0001", "--tdd", "none", "--title", "In scope"},
		{"add", "epic", "--title", "Billing"},
		{"add", "milestone", "--epic", "E-0002", "--tdd", "none", "--title", "Out of scope"},
	} {
		if out, err := testutil.RunBin(t, root, binDir, nil, args...); err != nil {
			t.Fatalf("aiwf %v: %v\n%s", args, err, out)
		}
	}
	// M-0103: move HEAD to a ritual-shape branch so the AI-target
	// preflight's implicit-current signal passes when opening the
	// E-0001 scope below.
	if out, err := testutil.RunGit(root, "checkout", "-b", "epic/E-0001-platform"); err != nil {
		t.Fatalf("git checkout -b: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil,
		"authorize", "E-0001", "--to", "ai/claude", "--reason", "implement E-0001"); err != nil {
		t.Fatalf("aiwf authorize E-0001: %v\n%s", err, out)
	}
	return root, bin
}

// TestScopeReach_OutOfScopeRefusal_AC2 is M-0141/AC-2: an authorized
// agent's verb on an in-scope target succeeds; on an out-of-scope target
// it refuses with the structured provenance-authorization-out-of-scope
// code (errors.As-able, surfaced as error.code under --format=json),
// exits 1, and leaves HEAD unchanged. D-0006 three-edge reachability is
// the boundary; D-0014 the reconcile.
func TestScopeReach_OutOfScopeRefusal_AC2(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	root, bin := setupTwoEpicScopeRepo(t)

	// Positive arm: agent promotes M-0001 (in E-0001's scope tree). Legal
	// FSM transition + reachable -> succeeds.
	if stdout, stderr, code := runSplit(t, root, bin,
		"promote", "M-0001", "in_progress",
		"--actor", "ai/claude", "--principal", "human/peter"); code != 0 {
		t.Fatalf("in-scope promote exit = %d, want 0\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}

	// Negative arm: agent promotes M-0002 (under E-0002, out of E-0001's
	// scope tree). Legal FSM transition but NOT reachable -> refused.
	headBefore, err := testutil.RunGit(root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v\n%s", err, headBefore)
	}

	stdout, stderr, code := runSplit(t, root, bin,
		"promote", "M-0002", "in_progress",
		"--actor", "ai/claude", "--principal", "human/peter", "--format=json")
	if code != 1 {
		t.Errorf("out-of-scope promote exit = %d, want 1\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
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
	if env.Error.Message == "" {
		t.Error("error.message is empty")
	}

	// HEAD must be unchanged — the refused mutation lands no commit.
	headAfter, err := testutil.RunGit(root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse HEAD (after): %v\n%s", err, headAfter)
	}
	if strings.TrimSpace(headBefore) != strings.TrimSpace(headAfter) {
		t.Errorf("HEAD moved on a refused verb: before=%s after=%s", strings.TrimSpace(headBefore), strings.TrimSpace(headAfter))
	}
}

// TestScopeReach_DiscoveredInFriction_AC3 is M-0141/AC-3, the D-0006
// friction case: an agent authorized on E-0001 files a gap with
// --discovered-in M-0001 (M-0001 is in E-0001's subtree), then promotes
// that gap. The promote is reachable ONLY via the discovered_in-reverse
// edge — gaps have no parent — so strict-parent-only reachability would
// wrongly refuse it, forcing a hand-back to the human. The narrowed
// three-edge ReachesScope must still allow it.
func TestScopeReach_DiscoveredInFriction_AC3(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)
	root := t.TempDir()
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
	for _, args := range [][]string{
		{"init"},
		{"add", "epic", "--title", "Platform"},
		{"add", "milestone", "--epic", "E-0001", "--tdd", "none", "--title", "Cache"},
	} {
		if out, err := testutil.RunBin(t, root, binDir, nil, args...); err != nil {
			t.Fatalf("aiwf %v: %v\n%s", args, err, out)
		}
	}
	// M-0103: ritual branch satisfies the AI-target preflight.
	if out, err := testutil.RunGit(root, "checkout", "-b", "epic/E-0001-platform"); err != nil {
		t.Fatalf("git checkout -b: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil,
		"authorize", "E-0001", "--to", "ai/claude", "--reason", "implement E-0001"); err != nil {
		t.Fatalf("aiwf authorize: %v\n%s", err, out)
	}

	// Agent files a gap discovered_in M-0001 — a creation act whose ref
	// (M-0001) reaches E-0001 via parent, so it is in scope.
	if out, err := testutil.RunBin(t, root, binDir, nil,
		"add", "gap", "--discovered-in", "M-0001", "--title", "Cache thrash",
		"--actor", "ai/claude", "--principal", "human/peter"); err != nil {
		t.Fatalf("agent add gap (in-scope creation): %v\n%s", err, out)
	}

	// Agent promotes the gap it just filed (--by names the in-scope
	// resolving milestone, satisfying the unrelated gap-addressed-has-
	// resolver precondition). Reachable ONLY via discovered_in reverse
	// (G-0001 has no parent) — the friction case.
	stdout, stderr, code := runSplit(t, root, bin,
		"promote", "G-0001", "addressed", "--by", "M-0001",
		"--actor", "ai/claude", "--principal", "human/peter")
	if code != 0 {
		t.Fatalf("agent promote of its own in-scope gap refused (exit %d); discovered_in-reverse reachability must allow it\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
}
