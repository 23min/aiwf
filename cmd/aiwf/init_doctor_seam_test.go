package main

import (
	"strings"
	"testing"
)

// M-069 AC-5 — init then doctor --self-check seam in a fresh tempdir
// repo.
//
// `aiwf doctor --self-check` is the kernel's end-to-end smoke test: it
// spins up an internal throwaway repo, drives every verb through it,
// and reports pass/fail. Existing tests run it directly against a bare
// tempdir (`TestRun_DoctorSelfCheck_Passes`,
// `TestBinary_DoctorSelfCheck_Passes`) — they confirm doctor's
// *internal* sequence works. They do *not* exercise the consumer's
// natural quickstart flow: a user clones a fresh repo, runs
// `aiwf init` to scaffold it, and only then runs
// `aiwf doctor --self-check` to verify the install is healthy.
//
// The seam between `aiwf init` (consumer-tempdir scaffolding) and
// `aiwf doctor --self-check` (internal-tempdir verb matrix) is where
// state from one could silently break the other. A regression where
// init leaves state that doctor reads during its own scaffolding, or
// where the consumer's hooks fire against the doctor's throwaway
// commits, would fail this seam without showing up in either direct
// test. The existing tests pass even with such a regression because
// they skip the init step entirely.

// TestSeam_InitThenDoctorSelfCheck (M-069 AC-5) drives the consumer's
// natural quickstart flow as a subprocess sequence:
//
//  1. build the aiwf binary;
//  2. fresh tempdir + `git init`;
//  3. `aiwf init --actor human/test` (full install — hook + scaffolding);
//  4. `aiwf doctor --self-check` from that same tempdir;
//  5. assert both succeed and doctor reports `self-check passed`.
//
// Subprocess form (not in-process `run([]string{...})`) is required:
// `aiwf init` bakes the binary's path into the pre-push hook, and
// in-process dispatch would put the test binary's path there — any
// hook firing during the self-check's commits would deadlock by
// re-invoking the test binary.
func TestSeam_InitThenDoctorSelfCheck(t *testing.T) {
	t.Parallel()
	skipIfShortOrUnsupported(t)
	tmp := t.TempDir()
	bin := buildBinary(t, tmp /* no ldflags */)

	repo := t.TempDir()
	mustExec(t, repo, "git", "init", "-q")
	mustExec(t, repo, "git", "config", "user.email", "test@example.com")
	mustExec(t, repo, "git", "config", "user.name", "aiwf-test")

	// Step 3: aiwf init — consumer's quickstart first command. Full
	// install (no --skip-hook) so the seam covers the hook-installed
	// state, not a stripped-down setup.
	if out, err := runBinaryAt(repo, bin, "init", "--actor", "human/test"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}

	// Step 4: aiwf doctor --self-check — consumer's quickstart
	// validation command. Runs from inside the init'd repo so any
	// state init left behind that doctor implicitly reads surfaces
	// here.
	out, err := runBinaryAt(repo, bin, "doctor", "--self-check")
	if err != nil {
		t.Fatalf("aiwf doctor --self-check: %v\n%s", err, out)
	}
	if !strings.Contains(out, "self-check passed") {
		t.Errorf("doctor output missing pass marker after init→self-check seam:\n%s", out)
	}
}
