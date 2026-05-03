package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
)

// TestRun_PromotePhaseWithTestsFlag is the seam test for I3 step 2:
// the dispatcher's --tests flag must end up as a real aiwf-tests
// trailer on the resulting commit. Drives runPromote through run()
// with the flag set, then reads HEAD trailers via gitops.
//
// Per the testing-policy rule from G27: a unit test of
// gitops.ParseStrictTestMetrics alone would not catch the bug class
// where the dispatcher parses the flag but forgets to plumb the
// metrics into the verb (a parallel-source-of-truth regression).
// This test exercises the full dispatcher → verb → commit path.
func TestRun_PromotePhaseWithTestsFlag(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test")
	mustRun(t, "add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-01", "--title", "First", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "ac", "--actor", "human/test", "--root", root, "M-001", "--title", "Engine starts")
	mustRun(t, "promote", "--actor", "human/test", "--root", root, "M-001/AC-1", "--phase", "red")
	mustRun(t, "promote", "--actor", "human/test", "--root", root, "M-001/AC-1", "--phase", "green",
		"--tests", "pass=12 fail=0 skip=1")

	trailers, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatalf("HeadTrailers: %v", err)
	}
	var sawTests bool
	for _, tr := range trailers {
		if tr.Key == "aiwf-tests" {
			sawTests = true
			if tr.Value != "pass=12 fail=0 skip=1" {
				t.Errorf("aiwf-tests value = %q, want %q", tr.Value, "pass=12 fail=0 skip=1")
			}
		}
	}
	if !sawTests {
		t.Errorf("expected aiwf-tests trailer on HEAD; got %+v", trailers)
	}
}

// TestRun_PromotePhase_TestsRejectsBadInput: the dispatcher must
// emit a usage error and exit non-zero when --tests carries an
// unknown key, malformed token, or negative value. The verb is not
// invoked and no commit lands.
func TestRun_PromotePhase_TestsRejectsBadInput(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test")
	mustRun(t, "add", "epic", "--title", "F", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-01", "--title", "M", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "ac", "--actor", "human/test", "--root", root, "M-001", "--title", "AC")
	mustRun(t, "promote", "--actor", "human/test", "--root", root, "M-001/AC-1", "--phase", "red")

	cases := []string{
		"duration=120ms",
		"pass",
		"pass=oops",
		"pass=-1",
	}
	for _, bad := range cases {
		t.Run(bad, func(t *testing.T) {
			rc := run([]string{
				"promote", "--actor", "human/test", "--root", root,
				"M-001/AC-1", "--phase", "green", "--tests", bad,
			})
			if rc == exitOK {
				t.Errorf("--tests %q should be a usage error; got exitOK", bad)
			}
		})
	}
}

// TestRun_AddACWithTestsFlag: --tests on add ac under tdd: required
// lands the trailer; on a non-tdd-required parent the dispatcher
// surfaces the verb's refusal as a non-zero exit.
func TestRun_AddACWithTestsFlag(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test")
	mustRun(t, "add", "epic", "--title", "F", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-01", "--title", "Required", "--actor", "human/test", "--root", root)

	// Hand-edit M-001 to tdd: required; the next add ac will seed red.
	mPath := filepath.Join(root, "work", "epics", "E-01-f", "M-001-required.md")
	raw, readErr := os.ReadFile(mPath)
	if readErr != nil {
		t.Fatalf("read M-001: %v", readErr)
	}
	patched := strings.Replace(string(raw), "status: draft\n", "status: draft\ntdd: required\n", 1)
	if writeErr := os.WriteFile(mPath, []byte(patched), 0o644); writeErr != nil {
		t.Fatalf("write M-001: %v", writeErr)
	}

	// The hand-edit is the test's premise: the user puts the
	// milestone into tdd: required state, then runs `aiwf add ac
	// --tests`. The verb must succeed on this — including the
	// case where the hand-edit is uncommitted. Earlier iterations
	// of this test wrapped the edit in a manual `git commit`
	// trailer block to dodge a perceived projection issue; the
	// I3 audit found that to be papering over rather than testing.
	// The verb's own commit is what carries the aiwf-tests
	// trailer; what we read at HEAD afterwards is that commit.
	mustRun(t, "add", "ac", "--actor", "human/test", "--root", root, "M-001", "--title", "Engine",
		"--tests", "pass=0 fail=1 skip=0")

	trailers, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatalf("HeadTrailers: %v", err)
	}
	var sawTests bool
	for _, tr := range trailers {
		if tr.Key == "aiwf-tests" && tr.Value == "pass=0 fail=1 skip=0" {
			sawTests = true
		}
	}
	if !sawTests {
		t.Errorf("expected aiwf-tests trailer on add-ac under tdd: required; got %+v", trailers)
	}

	// Non-tdd milestone: --tests must fail.
	mustRun(t, "add", "milestone", "--epic", "E-01", "--title", "Optional", "--actor", "human/test", "--root", root)
	rc := run([]string{
		"add", "ac", "--actor", "human/test", "--root", root,
		"M-002", "--title", "Pointless", "--tests", "pass=1",
	})
	if rc == exitOK {
		t.Error("--tests on non-tdd milestone should fail; got exitOK")
	}
}

// TestRun_PromoteStatusModeRejectsTests: --tests is only meaningful
// in phase mode; passing it with a positional new-status (status
// mode) is a usage error.
func TestRun_PromoteStatusModeRejectsTests(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test")
	mustRun(t, "add", "epic", "--title", "F", "--actor", "human/test", "--root", root)

	rc := run([]string{
		"promote", "--actor", "human/test", "--root", root,
		"E-01", "active", "--tests", "pass=1",
	})
	if rc == exitOK {
		t.Error("--tests in status mode should be a usage error; got exitOK")
	}
}

// mustRun runs the dispatcher and fatals on non-zero. Centralises the
// boilerplate t.Fatalf wrapping.
func mustRun(t *testing.T, args ...string) {
	t.Helper()
	if rc := run(args); rc != exitOK {
		t.Fatalf("run(%s) = %d, want exitOK", strings.Join(args, " "), rc)
	}
}
