package stresstest

import (
	"os"
	"path/filepath"
	"testing"
)

// verb_sequence_list_invariant_real_test.go — M-0250/AC-3: confirms
// checkListInvariant's real wiring (runAiwfListJSON's subprocess
// launch + JSON decode, tree.Load's real invocation) against an
// actual repo — the pure comparison logic itself is pinned
// exhaustively in verb_sequence_list_invariant_test.go against
// fabricated inputs. The real `aiwf` binary and tree.Load read the
// same on-disk state, so they can only disagree if there's a real
// bug — TestCheckListInvariant_RealBinary_DetectsAGenuineDivergence
// below manufactures exactly that disagreement with a stand-in
// "aiwf" that reports a wrong result, confirming checkListInvariant's
// wiring doesn't silently drop what classifyListInvariant finds (a
// wf-vacuity mutation probe on the un-instrumented wiring surfaced
// this as a real gap: a version of checkListInvariant that discarded
// classifyListInvariant's result and always returned no violations
// passed every other test in this file).
func TestCheckListInvariant_RealBinary_NoDivergenceOnARealRepo(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)

	if _, err := runAiwfJSON(bin, dir, "add", "epic", "--title", "epic a", "--body", "b"); err != nil {
		t.Fatalf("add epic: %v", err)
	}
	addEnv, err := runAiwfJSON(bin, dir, "add", "adr", "--title", "t", "--body", "b")
	if err != nil {
		t.Fatalf("add adr: %v", err)
	}
	if _, promErr := runAiwfJSON(bin, dir, "promote", addEnv.Metadata.EntityID, "rejected"); promErr != nil {
		t.Fatalf("promote: %v", promErr)
	}

	violations, err := checkListInvariant(bin, dir, "label")
	if err != nil {
		t.Fatalf("checkListInvariant: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("unexpected violations: %+v", violations)
	}
}

// TestCheckListInvariant_RealBinary_DetectsAGenuineDivergence points
// checkListInvariant at a stand-in "aiwf" that always reports an
// empty list, run against a directory tree.Load can still see the
// real entity in — a genuine, real-subprocess-observable divergence
// between the two independent sources checkListInvariant compares.
// Closes the vacuity gap TestCheckListInvariant_RealBinary_
// NoDivergenceOnARealRepo alone can't: that test's "no violations"
// assertion passes identically whether checkListInvariant is wired
// correctly or silently drops classifyListInvariant's result.
func TestCheckListInvariant_RealBinary_DetectsAGenuineDivergence(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	realBin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)

	if _, err := runAiwfJSON(realBin, dir, "add", "epic", "--title", "epic a", "--body", "b"); err != nil {
		t.Fatalf("add epic: %v", err)
	}

	fakeBin := writeFakeAiwfList(t, `{"status":"ok","findings":[],"result":[]}`)

	violations, err := checkListInvariant(fakeBin, dir, "label")
	if err != nil {
		t.Fatalf("checkListInvariant: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("violations = %+v, want exactly 1 (the epic missing from the fake list output)", violations)
	}
}

// writeFakeAiwfList writes an executable shell script that ignores
// its arguments and prints stdout to stdout, standing in for `aiwf`
// wherever a test needs to control exactly what `aiwf list
// --format=json` reports without a real subprocess's real answer.
func writeFakeAiwfList(t *testing.T, stdout string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "aiwf")
	script := "#!/bin/sh\ncat <<'EOF'\n" + stdout + "\nEOF\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil { //nolint:gosec // deliberately executable; a test-local stand-in binary, not attacker-controlled input
		t.Fatalf("writing fake aiwf binary: %v", err)
	}
	return path
}
