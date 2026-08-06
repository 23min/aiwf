package stresstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/testsupport"
)

// refless_stability_real_test.go — M-0300/AC-2 and AC-3: the wiring
// behind classifyRefLessStability, driven against real repositories and
// real subprocesses.

func TestRemovableRefs_ASingleBranchRepoHasNothingTheTreeDoesNotNeed(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)

	if _, err := runAiwfJSON(bin, dir, "add", "epic", "--title", "epic a", "--body", "b"); err != nil {
		t.Fatalf("add epic: %v", err)
	}

	got, err := removableRefs(dir)
	if err != nil {
		t.Fatalf("removableRefs: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("removableRefs() = %v, want none — the only ref is the branch HEAD is on", got)
	}
}

func TestRemovableRefs_KeepsTheCheckoutAndTrunkAndOffersAnUnrelatedBranch(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newCrossBranchReferenceRepo(t, bin)

	got, err := removableRefs(dir)
	if err != nil {
		t.Fatalf("removableRefs: %v", err)
	}

	// refs/heads/main is the checkout and refs/remotes/origin/main is the
	// configured trunk the uniqueness check reads; only the side branch is
	// a ref the tree does not need.
	if diff := cmp.Diff([]string{"refs/heads/side"}, got); diff != "" {
		t.Errorf("removableRefs() mismatch (-want +got):\n%s", diff)
	}
}

// TestRefLessStabilityInvariant_RealBinary_StableWhenTheSubcodeRefines is
// this property's load-bearing tolerance, measured rather than reasoned
// about. Stripping the branch that carries the cited id moves the full
// check's classification from cross-branch-local-only to unresolved.
// Both block, so the tree's verdict did not change its mind — and a
// property comparing finding sets for identity would report a defect
// here on every run.
func TestRefLessStabilityInvariant_RealBinary_StableWhenTheSubcodeRefines(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newCrossBranchReferenceRepo(t, bin)

	removable, err := removableRefs(dir)
	if err != nil {
		t.Fatalf("removableRefs: %v", err)
	}
	if len(removable) == 0 {
		t.Fatal("fixture no longer offers a removable ref; the property would short-circuit and assert nothing")
	}

	// Guard the other half of the same vacuity risk: the subject has to
	// be classified in both runs for the comparison to have happened.
	withRefs, err := runAiwfJSON(bin, dir, "check")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	subject := readPathSubject{EntityID: "E-0001", Code: "body-prose-id"} //enums:ignore reading the wire, matching what the fixture's own measured output carries
	if !blockingSubjectsFrom(withRefs.Findings)[subject] {
		t.Fatalf("fixture no longer produces a blocking cross-branch verdict; findings: %+v", withRefs.Findings)
	}

	violations, err := refLessStabilityInvariant{}.Evaluate(bin, dir, "cross-branch")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("a subcode refinement was reported as instability: %+v", violations)
	}
}

func TestRefLessStabilityInvariant_RealBinary_NothingToRemoveIsNothingToJudge(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)

	if _, err := runAiwfJSON(bin, dir, "add", "epic", "--title", "epic a", "--body", "b"); err != nil {
		t.Fatalf("add epic: %v", err)
	}

	violations, err := refLessStabilityInvariant{}.Evaluate(bin, dir, "single-branch")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if violations != nil {
		t.Errorf("Evaluate() = %+v, want nil when no ref can be taken away", violations)
	}
}

// TestRefLessStabilityInvariant_RealBinary_ReportsADispositionFlip is
// AC-3's constructed violation. G-0556's repair landed on main before
// this property was written, so no repository makes the real check flip
// a subject's disposition when an unneeded ref disappears; a stand-in
// `aiwf` that answers by which copy it was run in produces the flip
// across real subprocesses and a real ref-stripped copy, and proves
// Evaluate does not discard what the comparison core finds.
func TestRefLessStabilityInvariant_RealBinary_ReportsADispositionFlip(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	realBin := sharedTestBinary(t)
	dir := newCrossBranchReferenceRepo(t, realBin)

	fake := writeFakeAiwfByWorkingCopy(t,
		findingsEnvelope(`{"code":"refs-resolve","subcode":"cross-branch-pending","severity":"warning","entity_id":"G-0001"}`),
		findingsEnvelope(`{"code":"refs-resolve","subcode":"unresolved","severity":"error","entity_id":"G-0001"}`))

	violations, err := refLessStabilityInvariant{}.Evaluate(fake, dir, "constructed")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("violations = %+v, want exactly 1", violations)
	}
	for _, want := range []string{"constructed", "G-0001", "refs/heads/side", "blocking"} {
		if !strings.Contains(violations[0].Message, want) {
			t.Errorf("violation message %q does not name %q", violations[0].Message, want)
		}
	}
}

func TestRefLessStabilityInvariant_ErrorsWhenTheBinaryCannotRun(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	realBin := sharedTestBinary(t)
	dir := newCrossBranchReferenceRepo(t, realBin)

	missing := filepath.Join(t.TempDir(), "no-such-aiwf-binary")
	if _, err := (refLessStabilityInvariant{}).Evaluate(missing, dir, "label"); err == nil {
		t.Fatal("Evaluate() returned no error for a binary that cannot be launched")
	}
}

func TestRemovableRefs_ErrorsOnADirectoryThatIsNotARepository(t *testing.T) {
	t.Parallel()

	if _, err := removableRefs(t.TempDir()); err == nil {
		t.Fatal("removableRefs() returned no error for a directory with no .git")
	}
}

func TestRefLessStabilityInvariant_ErrorsWhenTheRepositoryCannotBeRead(t *testing.T) {
	t.Parallel()

	if _, err := (refLessStabilityInvariant{}).Evaluate("bin", t.TempDir(), "label"); err == nil {
		t.Fatal("Evaluate() returned no error for a directory with no .git")
	}
}

func TestRemovableRefs_ErrorsWhenTheRefsCannotBeListed(t *testing.T) {
	t.Parallel()

	// A `.git` that exists but is not a repository: the needed-ref
	// queries all degrade quietly, and listing the refs is where it
	// surfaces.
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("creating a bare .git directory: %v", err)
	}

	if _, err := removableRefs(dir); err == nil {
		t.Fatal("removableRefs() returned no error for a .git that is not a repository")
	}
}

func TestRemovableRefs_ErrorsOnAnUnreadableConfig(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	dir := newVerbSequenceTestRepo(t)

	if err := os.WriteFile(filepath.Join(dir, "aiwf.yaml"), []byte("allocate: [not, a, mapping\n"), 0o600); err != nil {
		t.Fatalf("writing aiwf.yaml: %v", err)
	}

	if _, err := removableRefs(dir); err == nil {
		t.Fatal("removableRefs() returned no error for an aiwf.yaml that cannot be parsed")
	}
}

func TestRemovableRefs_AnUnbornRepositoryHasNoRefsAtAll(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)

	got, err := removableRefs(newVerbSequenceTestRepo(t))
	if err != nil {
		t.Fatalf("removableRefs: %v", err)
	}
	if got != nil {
		t.Errorf("removableRefs() = %v, want none — the repository has no refs yet", got)
	}
}

func TestRemovableRefs_KeepsTheUpstreamOfTheCheckedOutBranch(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newCrossBranchReferenceRepo(t, bin)

	// The fixture pushes without tracking, so the upstream query fails
	// there and the trunk ref alone keeps refs/remotes/origin/main. Set
	// tracking and it is kept for its own reason too — the provenance
	// audit range is defined by it.
	runGitOrFatal(t, dir, "branch", "--set-upstream-to", "origin/main", "main")

	needed, err := neededRefs(dir)
	if err != nil {
		t.Fatalf("neededRefs: %v", err)
	}
	if !needed["refs/remotes/origin/main"] {
		t.Errorf("neededRefs() = %v, want it to keep the upstream", needed)
	}
}

func TestCopyRepoWithoutRefs_ErrorsWhenTheRepositoryCannotBeCopied(t *testing.T) {
	t.Parallel()

	if _, _, err := copyRepoWithoutRefs(filepath.Join(t.TempDir(), "no-such-repo"), nil); err == nil {
		t.Fatal("copyRepoWithoutRefs() returned no error for a directory that does not exist")
	}
}

// TestCopyRepoWithoutRefs_ErrorsWhenARefCannotBeDeleted covers the branch
// that keeps this property from going quietly vacuous: a deletion that
// fails and is swallowed leaves the copy carrying the ref, and the two
// runs then agree because they read the same repository twice.
//
// `git update-ref -d` treats an already-absent ref as done, so a missing
// name is not a failure; a single-level name, which git refuses to
// resolve as a ref at all, is.
func TestCopyRepoWithoutRefs_ErrorsWhenARefCannotBeDeleted(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	dir := newVerbSequenceTestRepo(t)
	runGitOrFatal(t, dir, "commit", "-q", "--allow-empty", "-m", "seed")

	if _, _, err := copyRepoWithoutRefs(dir, []string{"not-a-ref"}); err == nil {
		t.Fatal("copyRepoWithoutRefs() returned no error for a ref git refuses to delete")
	}
}

func TestCopyRepoWithoutRefs_LeavesTheOriginalRepositoryUntouched(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newCrossBranchReferenceRepo(t, bin)

	stripped, cleanup, err := copyRepoWithoutRefs(dir, []string{"refs/heads/side"})
	if err != nil {
		t.Fatalf("copyRepoWithoutRefs: %v", err)
	}
	defer cleanup()

	if strippedRefs, refErr := gitRefNames(stripped); refErr != nil || contains(strippedRefs, "refs/heads/side") {
		t.Errorf("the copy still carries refs/heads/side: %v (%v)", strippedRefs, refErr)
	}
	originalRefs, err := gitRefNames(dir)
	if err != nil {
		t.Fatalf("gitRefNames: %v", err)
	}
	if !contains(originalRefs, "refs/heads/side") {
		t.Errorf("the scenario's own repository lost refs/heads/side: %v", originalRefs)
	}
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// writeFakeAiwfByWorkingCopy writes an executable shell script standing
// in for `aiwf`, answering with strippedStdout when it is run inside the
// ref-stripped copy and originalStdout otherwise. The copy's path is
// what tells the two apart, which is the only signal a surface run twice
// over the same arguments has.
func writeFakeAiwfByWorkingCopy(t *testing.T, originalStdout, strippedStdout string) string {
	t.Helper()
	script := "#!/bin/sh\ncase \"$PWD\" in\n" +
		"  *refless-*) cat <<'EOF'\n" + strippedStdout + "\nEOF\n  ;;\n" +
		"  *) cat <<'EOF'\n" + originalStdout + "\nEOF\n  ;;\nesac\n"

	path := filepath.Join(t.TempDir(), "aiwf")
	if err := testsupport.WriteExecutable(path, []byte(script)); err != nil {
		t.Fatalf("writing fake aiwf binary: %v", err)
	}
	return path
}
