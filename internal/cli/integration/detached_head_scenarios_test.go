package integration

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// detached_head_scenarios_test.go — M-0161/AC-7 (G-0207)
// real-git E2E scenarios for detached HEAD behavior across
// the four surfaces (preflight, oracle, check, doctor).
//
// AC-7 contract (per body):
//
//   - Preflight: `aiwf authorize --to ai/<agent>` refuses
//     with stderr containing "detached HEAD has no ritual
//     context" (substring exception per AC-7 body line 498
//     — verb-time errors don't carry structured codes).
//   - Oracle + check: detached HEAD doesn't degrade
//     functionality; AI commit on dangling HEAD is silent
//     per AC-3's KNOWN-GOOD-empty path.
//   - Doctor: surfaces an advisory `head: detached-head:
//     advisory ...` line on detached HEAD; substring marker
//     is `detached-head` (the canonical AC-7 token).
//
// AC-7 body line 483 calls for a JSON envelope from `aiwf
// doctor`, but the doctor verb does not currently support
// `--format=json` (deferred to a follow-up gap — adding
// JSON output to doctor is a separate scope). The tests
// below use substring matches against doctor's text output;
// this matches AC-7 body line 498's substring-against-stderr
// exception extended to substring-against-stdout for the
// doctor surface. The trade-off is documented; the load-
// bearing contract (the rule surfaces the state) is pinned
// at the right tightness for today's output.

// TestDetachedHEAD_AC7_PreflightRefusesWithRefinedMessage
// pins the preflight refinement at lines 460-464 of the
// AC-7 body: detached HEAD + `aiwf authorize --to ai/...`
// refuses with stderr containing the canonical substring.
func TestDetachedHEAD_AC7_PreflightRefusesWithRefinedMessage(t *testing.T) {
	t.Parallel()
	pinCell("branch-cell-m0161-ac7-c1", t.Name())
	env := newScenarioEnv(t)
	env.MustRunBin("add", "epic", "--title", "Engine")
	// Detach HEAD at main's tip.
	mainSHA := strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD"))
	env.MustRunGit("checkout", mainSHA)

	// Run authorize WITHOUT --force; expect refusal with the
	// refined "detached HEAD has no ritual context" substring.
	out, err := testutil.RunBin(t, env.Root, env.BinDir, nil,
		"authorize", "E-0001",
		"--to", "ai/claude",
		"--branch", "epic/E-0001-engine",
	)
	if err == nil {
		t.Fatalf("expected aiwf authorize to fail on detached HEAD; got success\n%s", out)
	}
	if !strings.Contains(out, "detached HEAD has no ritual context") {
		t.Errorf("expected stderr to contain %q; got:\n%s", "detached HEAD has no ritual context", out)
	}
	// Also verify the override path is named.
	if !strings.Contains(out, "--force") {
		t.Errorf("expected refusal text to name `--force` override; got:\n%s", out)
	}
}

// TestDetachedHEAD_AC7_PreflightNoBranchRefuses pins AC-7
// matrix row 2: detached HEAD + `aiwf authorize --to ai/...`
// (NO --branch) hits PreflightBranchContextRequiredError's
// refined text (the rung-pair check is gated on --branch; with
// no --branch we drop into the legacy AI-target preflight at
// internal/verb/authorize.go:391-395 which emits
// PreflightBranchContextRequiredError when CurrentBranch is
// non-ritual). The detached-HEAD refinement at
// internal/verb/authorize.go:87-93 is the load-bearing branch
// here — without this test it is dead-letter code (M-0161/
// AC-7 reviewer B1).
func TestDetachedHEAD_AC7_PreflightNoBranchRefuses(t *testing.T) {
	t.Parallel()
	pinCell("branch-cell-m0161-ac7-c2", t.Name())
	env := newScenarioEnv(t)
	env.MustRunBin("add", "epic", "--title", "Engine")
	mainSHA := strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD"))
	env.MustRunGit("checkout", mainSHA)

	out, err := testutil.RunBin(t, env.Root, env.BinDir, nil,
		"authorize", "E-0001",
		"--to", "ai/claude",
		// no --branch — hits the branch-context-required path
	)
	if err == nil {
		t.Fatalf("expected aiwf authorize to fail on detached HEAD without --branch; got success\n%s", out)
	}
	if !strings.Contains(out, "detached HEAD has no ritual context") {
		t.Errorf("expected stderr to contain %q; got:\n%s", "detached HEAD has no ritual context", out)
	}
	if !strings.Contains(out, "branch-context-required") {
		t.Errorf("expected stderr to name the branch-context-required code (the AC-7-refined error path); got:\n%s", out)
	}
}

// TestDetachedHEAD_AC7_PreflightForceReasonBypasses pins
// the override path (AC-7 matrix row 3): detached HEAD +
// `--force --reason "..."` succeeds and the commit carries
// the aiwf-force trailer.
func TestDetachedHEAD_AC7_PreflightForceReasonBypasses(t *testing.T) {
	t.Parallel()
	pinCell("branch-cell-m0161-ac7-c3", t.Name())
	env := newScenarioEnv(t)
	env.MustRunBin("add", "epic", "--title", "Engine")
	mainSHA := strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD"))
	env.MustRunGit("checkout", mainSHA)

	// --force --reason should succeed; the verb-time refusal
	// path is bypassed (M-0103 sovereign override).
	env.MustRunBin("authorize", "E-0001",
		"--to", "ai/claude",
		"--branch", "epic/E-0001-engine",
		"--force",
		"--reason", "AC-7 fixture: intentional sovereign override from detached HEAD",
	)
	// Verify the authorize commit carries the aiwf-force trailer.
	body := env.MustRunGit("log", "-1", "--pretty=%B")
	if !strings.Contains(body, "aiwf-force:") {
		t.Errorf("expected authorize commit to carry aiwf-force trailer; commit body:\n%s", body)
	}
}

// TestDetachedHEAD_AC7_CheckSucceedsNoFalseFindings pins
// AC-7 matrix row 4: detached HEAD + no AI commits → `aiwf
// check` succeeds without false positives. The rule polices
// what it sees; detached state alone is not a finding.
func TestDetachedHEAD_AC7_CheckSucceedsNoFalseFindings(t *testing.T) {
	t.Parallel()
	pinCell("branch-cell-m0161-ac7-c4", t.Name())
	env := newScenarioEnv(t)
	env.MustRunBin("add", "epic", "--title", "Engine")
	mainSHA := strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD"))
	env.MustRunGit("checkout", mainSHA)
	// `aiwf check` runs without isolation-escape findings —
	// no AI commits to police.
	assertExpectation(t, env, Expectation{NoFindingWithCode: "isolation-escape"})
}

// TestDetachedHEAD_AC7_DoctorSurfacesAdvisory pins AC-7
// matrix row 6: `aiwf doctor` on a detached HEAD surfaces
// an advisory line containing the substring `detached-head`.
// Substring is the load-bearing signal for today's text-only
// doctor output (deferred JSON envelope per file header).
func TestDetachedHEAD_AC7_DoctorSurfacesAdvisory(t *testing.T) {
	t.Parallel()
	pinCell("branch-cell-m0161-ac7-c5", t.Name())
	env := newScenarioEnv(t)
	env.MustRunBin("add", "epic", "--title", "Engine")
	mainSHA := strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD"))
	env.MustRunGit("checkout", mainSHA)

	out := env.MustRunBin("doctor")
	if !strings.Contains(out, "detached-head") {
		t.Errorf("expected doctor output to contain %q on detached HEAD; got:\n%s", "detached-head", out)
	}
	if !strings.Contains(out, "advisory") {
		t.Errorf("expected doctor output to mark severity as advisory; got:\n%s", out)
	}
}

// TestDetachedHEAD_AC7_DanglingAICommitSilent pins AC-7
// matrix row 5: an AI-actor commit made FROM detached HEAD
// produces a dangling commit (no ref points at it after re-
// attach, but it remains reachable from HEAD while detached).
// The oracle's first-parent index doesn't include the
// dangling commit's branch (because no branch points at it),
// so FirstParentBranches returns empty → KNOWN-GOOD empty
// per AC-3's typed-error contract → isolation-escape stays
// silent.
func TestDetachedHEAD_AC7_DanglingAICommitSilent(t *testing.T) {
	t.Parallel()
	pinCell("branch-cell-m0161-ac7-c6", t.Name())
	env := newScenarioEnv(t)
	env.MustRunBin("add", "epic", "--title", "Engine")
	OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
	// Detach HEAD by checking out the SHA.
	headSHA := strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD"))
	env.MustRunGit("checkout", headSHA)
	// Simulate an AI escape (raw git commit) ON the detached
	// HEAD. The commit becomes a dangling commit — reachable
	// from HEAD but not from any ritual ref.
	SimulateAIEscape(t, env, "E-0001", "AI commit from detached HEAD (dangling)")
	// aiwf check sees the AI commit (HEAD-reachable) but its
	// branch lookup returns empty (no ritual ref points at
	// the dangling chain) → rule silent.
	assertExpectation(t, env, Expectation{NoFindingWithCode: "isolation-escape"})
}

// TestDetachedHEAD_AC7_DoctorSilentOnAttachedHEAD pins AC-7
// matrix row 7: when HEAD is on a real branch, the doctor
// detached-head advisory does NOT appear.
func TestDetachedHEAD_AC7_DoctorSilentOnAttachedHEAD(t *testing.T) {
	t.Parallel()
	pinCell("branch-cell-m0161-ac7-c7", t.Name())
	env := newScenarioEnv(t)
	env.MustRunBin("add", "epic", "--title", "Engine")
	// HEAD is on main (attached state); doctor should not
	// emit the detached-head advisory.
	out := env.MustRunBin("doctor")
	if strings.Contains(out, "detached-head:") {
		t.Errorf("doctor emitted detached-head advisory on attached HEAD; should be silent:\n%s", out)
	}
}
