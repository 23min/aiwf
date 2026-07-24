package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/gitops"
)

// E-0071 / M-0277: `aiwf milestone tdd <M-id> --policy
// none|advisory|required` is the post-creation mutator for a
// milestone's TDD policy — the `tdd:` slice of G-0168's
// verb-chokepoint hole. Tests in this file pin the verb's contract
// end-to-end through the in-process dispatcher.

// milestoneTDDSetup gives every test in this file a freshly-init'd repo
// with one epic and one milestone (M-0001, created `tdd: none`) so the
// post-creation tdd verb has a referent to flip.
func milestoneTDDSetup(t *testing.T) string {
	t.Helper()
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("add epic: %d", rc)
	}
	if rc := cli.Execute([]string{"add", "milestone", "--epic", "E-0001", "--tdd", "none", "--title", "First", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("add milestone: %d", rc)
	}
	return root
}

// milestoneOnePath is the on-disk path of the M-0001 fixture milestone.
func milestoneOnePath(root string) string {
	return filepath.Join(root, "work", "epics", "E-0001-foundations", "M-0001-first.md")
}

// TestMilestoneTDD_SetsPolicy_OneTrailered pins AC-1: the verb writes
// the milestone's `tdd:` field, produces exactly one commit, and stamps
// the standard aiwf-verb / aiwf-entity / aiwf-actor trailers.
func TestMilestoneTDD_SetsPolicy_OneTrailered(t *testing.T) {
	t.Parallel()
	root := milestoneTDDSetup(t)

	before := commitCountSafe(t, root)
	rc := cli.Execute([]string{
		"milestone", "tdd", "M-0001",
		"--policy", "required",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("milestone tdd M-0001 --policy required = %d, want %d", rc, cliutil.ExitOK)
	}

	// Frontmatter reflects the new policy.
	body, err := os.ReadFile(milestoneOnePath(root))
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	if !strings.Contains(string(body), "tdd: required") {
		t.Errorf("frontmatter missing `tdd: required`:\n%s", body)
	}

	// Exactly one commit landed (per-mutation atomicity).
	if after := commitCountSafe(t, root); after != before+1 {
		t.Errorf("commit count = %d, want %d (exactly one commit)", after, before+1)
	}

	// The commit carries the standard trailers.
	tr, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatalf("HeadTrailers: %v", err)
	}
	want := map[string]string{
		gitops.TrailerVerb:   "milestone-tdd",
		gitops.TrailerEntity: "M-0001",
		gitops.TrailerActor:  "human/test",
	}
	got := map[string]string{}
	for _, e := range tr {
		if _, ok := want[e.Key]; ok {
			got[e.Key] = e.Value
		}
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("trailer %s = %q, want %q (all trailers: %+v)", k, got[k], v, tr)
		}
	}
}

// TestMilestoneTDD_PolicyValidation pins AC-2: --policy is validated
// against the closed set {none, advisory, required}. An unknown value
// is a usage error that makes no mutation; each valid value succeeds.
func TestMilestoneTDD_PolicyValidation(t *testing.T) {
	t.Parallel()

	t.Run("unknown value is usage error and makes no mutation", func(t *testing.T) {
		t.Parallel()
		root := milestoneTDDSetup(t)

		before := commitCountSafe(t, root)
		rc := cli.Execute([]string{
			"milestone", "tdd", "M-0001",
			"--policy", "bogus",
			"--actor", "human/test",
			"--root", root,
		})
		if rc != cliutil.ExitUsage {
			t.Errorf("milestone tdd --policy bogus = %d, want %d", rc, cliutil.ExitUsage)
		}
		// No mutation: still `tdd: none` from setup, and no new commit.
		body, err := os.ReadFile(milestoneOnePath(root))
		if err != nil {
			t.Fatalf("read milestone: %v", err)
		}
		if !strings.Contains(string(body), "tdd: none") {
			t.Errorf("milestone mutated by a rejected --policy value:\n%s", body)
		}
		if after := commitCountSafe(t, root); after != before {
			t.Errorf("commit count = %d, want %d (rejected value must land no commit)", after, before)
		}
	})

	for _, val := range []string{"none", "advisory", "required"} {
		t.Run("valid value "+val, func(t *testing.T) {
			t.Parallel()
			root := milestoneTDDSetup(t)
			rc := cli.Execute([]string{
				"milestone", "tdd", "M-0001",
				"--policy", val,
				"--actor", "human/test",
				"--root", root,
			})
			if rc != cliutil.ExitOK {
				t.Errorf("milestone tdd --policy %s = %d, want %d", val, rc, cliutil.ExitOK)
			}
		})
	}
}

// TestMilestoneTDD_UniformOrdinaryGating pins AC-3 (embodies D-0048):
// gating is uniform-ordinary. An `ai/` actor operating within an active
// authorize scope, acting for a principal, may flip the policy in
// either direction — including the weakening `required -> none` — with
// no `--force`. Weakening and strengthening take the identical path;
// there is no directional or sovereign carve-out. This is a
// binary-level test because the authorize scope is branch-bound.
func TestMilestoneTDD_UniformOrdinaryGating(t *testing.T) {
	t.Parallel()
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
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
	for _, args := range [][]string{
		{"init"},
		{"add", "epic", "--title", "Engine"},
		{"promote", "E-0001", "active"},
		{"add", "milestone", "--tdd", "none", "--title", "Cache", "--epic", "E-0001"},
	} {
		if out, err := testutil.RunBin(t, root, binDir, nil, args...); err != nil {
			t.Fatalf("aiwf %v: %v\n%s", args, err, out)
		}
	}
	// A ritual branch satisfies the authorize AI-target preflight and
	// carries the ai/ actor's scope.
	if out, err := testutil.RunGit(root, "checkout", "-b", "epic/E-0001-engine"); err != nil {
		t.Fatalf("git checkout -b: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil,
		"authorize", "E-0001", "--to", "ai/claude", "--reason", "scoped to E-01"); err != nil {
		t.Fatalf("aiwf authorize: %v\n%s", err, out)
	}

	// Strengthen: none -> required, authorized ai/ actor, no --force.
	if out, err := testutil.RunBin(t, root, binDir, nil,
		"milestone", "tdd", "M-0001", "--policy", "required",
		"--actor", "ai/claude", "--principal", "human/peter"); err != nil {
		t.Fatalf("strengthen (authorized ai/ actor, no --force): %v\n%s", err, out)
	}
	// Weaken: required -> none, same actor, no --force. The weakening
	// direction is not specially gated — no sovereign carve-out.
	if out, err := testutil.RunBin(t, root, binDir, nil,
		"milestone", "tdd", "M-0001", "--policy", "none",
		"--actor", "ai/claude", "--principal", "human/peter"); err != nil {
		t.Fatalf("weaken (authorized ai/ actor, no --force): %v\n%s", err, out)
	}
}

// TestMilestoneTDD_UnauthorizedAgentRefused pins the other half of
// uniform-ordinary gating: an `ai/` actor with NO active scope is
// refused by the ordinary provenance gate — the verb inherits the
// standard entity-scoped gate rather than opening its own hole. Uses
// the in-process dispatcher (no scope is ever opened).
func TestMilestoneTDD_UnauthorizedAgentRefused(t *testing.T) {
	t.Parallel()
	root := milestoneTDDSetup(t)

	before := commitCountSafe(t, root)
	rc := cli.Execute([]string{
		"milestone", "tdd", "M-0001", "--policy", "required",
		"--actor", "ai/claude", "--principal", "human/test", "--root", root,
	})
	if rc == cliutil.ExitOK {
		t.Errorf("unauthorized ai/ actor unexpectedly succeeded; want refusal (provenance gate)")
	}
	if after := commitCountSafe(t, root); after != before {
		t.Errorf("refused mutation landed a commit: count %d, want %d", after, before)
	}
}

// TestMilestoneTDD_CompositeIDRejected pins AC-1's verb-level guard:
// tdd is a milestone-level field, so a composite id (M-NNNN/AC-N) is
// rejected before any mutation.
func TestMilestoneTDD_CompositeIDRejected(t *testing.T) {
	t.Parallel()
	root := milestoneTDDSetup(t)
	if rc := cli.Execute([]string{"add", "ac", "M-0001", "--title", "first ac", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("add ac: %d", rc)
	}

	rc := cli.Execute([]string{
		"milestone", "tdd", "M-0001/AC-1",
		"--policy", "required",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != cliutil.ExitUsage {
		t.Errorf("milestone tdd M-0001/AC-1 = %d, want %d (composite ids rejected)", rc, cliutil.ExitUsage)
	}
}

// TestMilestoneTDD_TargetNotMilestone pins AC-1's kind guard: the
// positional id must resolve to a milestone, not any other kind.
func TestMilestoneTDD_TargetNotMilestone(t *testing.T) {
	t.Parallel()
	root := milestoneTDDSetup(t)

	rc := cli.Execute([]string{
		"milestone", "tdd", "E-0001",
		"--policy", "required",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != cliutil.ExitUsage {
		t.Errorf("milestone tdd E-0001 = %d, want %d (E-0001 is not a milestone)", rc, cliutil.ExitUsage)
	}
}

// TestMilestoneTDD_TargetUnknown pins AC-1's not-found guard.
func TestMilestoneTDD_TargetUnknown(t *testing.T) {
	t.Parallel()
	root := milestoneTDDSetup(t)

	rc := cli.Execute([]string{
		"milestone", "tdd", "M-0999",
		"--policy", "required",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != cliutil.ExitUsage {
		t.Errorf("milestone tdd M-0999 = %d, want %d (M-0999 doesn't exist)", rc, cliutil.ExitUsage)
	}
}

// TestMilestoneTDD_NoPolicyIsUsage pins the verb's contract: --policy is
// required; a bare invocation is a usage error.
func TestMilestoneTDD_NoPolicyIsUsage(t *testing.T) {
	t.Parallel()
	root := milestoneTDDSetup(t)

	rc := cli.Execute([]string{
		"milestone", "tdd", "M-0001",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != cliutil.ExitUsage {
		t.Errorf("milestone tdd with no --policy = %d, want %d", rc, cliutil.ExitUsage)
	}
}

// TestMilestoneTDD_FlipsEitherDirection pins AC-1's either-direction
// contract at the happy-path level: none -> required -> none both land
// as clean mutations.
func TestMilestoneTDD_FlipsEitherDirection(t *testing.T) {
	t.Parallel()
	root := milestoneTDDSetup(t)

	if rc := cli.Execute([]string{"milestone", "tdd", "M-0001", "--policy", "required", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("flip to required: %d", rc)
	}
	if rc := cli.Execute([]string{"milestone", "tdd", "M-0001", "--policy", "none", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("flip back to none: %d", rc)
	}

	body, err := os.ReadFile(milestoneOnePath(root))
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	if !strings.Contains(string(body), "tdd: none") {
		t.Errorf("frontmatter missing `tdd: none` after flip-back:\n%s", body)
	}
}
