package stresstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/testsupport"
)

// readpath_agreement_real_test.go — M-0300/AC-1 and AC-3: the wiring
// behind classifyReadPathAgreement, driven against real subprocesses.
// The pure comparison core is pinned against constructed observations in
// readpath_agreement_test.go; these tests confirm the surfaces are
// actually run, actually decoded, and that what the core finds is not
// dropped on the way back.

func TestReadPathAgreementInvariant_RealBinary_NoContradictionOnAHealthyRepo(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)

	if _, err := runAiwfJSON(bin, dir, "add", "epic", "--title", "epic a", "--body", "b"); err != nil {
		t.Fatalf("add epic: %v", err)
	}
	if _, err := runAiwfJSON(bin, dir, "add", "gap", "--title", "gap a", "--body", "b"); err != nil {
		t.Fatalf("add gap: %v", err)
	}

	violations, err := readPathAgreementInvariant{}.Evaluate(bin, dir, "healthy")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("unexpected violations on a healthy repo: %+v", violations)
	}
}

// TestReadPathAgreementInvariant_RealBinary_ADeclinedJudgmentIsNotAContradiction
// is the property's load-bearing tolerance, measured rather than
// reasoned about. The tree cites an id carried only by an unpublished
// local branch, so the full check classifies it cross-branch-local-only
// at error severity while the ref-less surface reports
// unresolved-unverified — the same subject, a blocking verdict on one
// side and a declined judgment on the other. That is correct behavior on
// today's kernel, and a property comparing finding sets for identity
// would report it as a defect on every run.
func TestReadPathAgreementInvariant_RealBinary_ADeclinedJudgmentIsNotAContradiction(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newCrossBranchReferenceRepo(t, bin)

	gate, others, err := observeReadPaths(bin, dir)
	if err != nil {
		t.Fatalf("observeReadPaths: %v", err)
	}

	// Guard against the fixture silently ceasing to produce the shape
	// under test: without a blocking claim from the gate and silence from
	// the ref-less surface, the assertion below would pass vacuously.
	if gate.Blocking == 0 {
		t.Fatalf("fixture no longer produces a blocking gate verdict; gate claims: %+v", gate.Claims)
	}
	fast := findSurface(t, others, "aiwf check --fast")
	if len(fast.Claims) != 0 {
		t.Fatalf("fixture no longer produces a declined ref-less judgment; --fast claims: %+v", fast.Claims)
	}

	if violations := classifyReadPathAgreement("cross-branch", gate, others); len(violations) != 0 {
		t.Fatalf("a declined judgment was reported as a contradiction: %+v", violations)
	}
}

func TestObserveReadPaths_RealBinary_ObservesEveryVerdictRenderingSurface(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)

	if _, err := runAiwfJSON(bin, dir, "add", "epic", "--title", "epic a", "--body", "b"); err != nil {
		t.Fatalf("add epic: %v", err)
	}

	gate, others, err := observeReadPaths(bin, dir)
	if err != nil {
		t.Fatalf("observeReadPaths: %v", err)
	}

	if gate.Surface != "aiwf check" || !gate.Itemized {
		t.Errorf("gate = %q (itemized %v), want the full check, itemized", gate.Surface, gate.Itemized)
	}
	for _, want := range []string{"aiwf check --fast", "aiwf check --shape-only", "aiwf status"} {
		findSurface(t, others, want)
	}
	if status := findSurface(t, others, "aiwf status"); status.Itemized {
		t.Error("aiwf status was recorded as itemized; it states no per-finding subcode or severity")
	}
}

// TestReadPathAgreementInvariant_RealBinary_ReportsAContradictionBetweenTwoSurfaces
// is AC-3's constructed violation at the wiring level. Both defects that
// motivated this property are repaired on main, so no repository makes
// the real surfaces contradict each other; a stand-in `aiwf` that
// answers each surface differently produces the divergence off a real
// subprocess, and proves Evaluate does not discard what the comparison
// core finds.
func TestReadPathAgreementInvariant_RealBinary_ReportsAContradictionBetweenTwoSurfaces(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)

	fake := writeFakeAiwfSurfaces(t, map[string]string{
		"check": findingsEnvelope(`{"code":"refs-resolve","subcode":"cross-branch-pending","severity":"warning","entity_id":"G-0001"}`),
		"check --fast": findingsEnvelope(
			`{"code":"refs-resolve","subcode":"unresolved","severity":"error","entity_id":"G-0001"}`),
		"check --shape-only": findingsEnvelope(),
		"status":             `{"status":"ok","result":{"health":{"errors":0}}}`,
	})

	violations, err := readPathAgreementInvariant{}.Evaluate(fake, t.TempDir(), "constructed")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("violations = %+v, want exactly 1", violations)
	}
	for _, want := range []string{"constructed", "aiwf check", "aiwf check --fast", "G-0001", "refs-resolve"} {
		if !strings.Contains(violations[0].Message, want) {
			t.Errorf("violation message %q does not name %q", violations[0].Message, want)
		}
	}
}

// TestReadPathAgreementInvariant_RealBinary_ReportsAggregateSurfaceBlockingMoreThanTheGate
// is AC-3's constructed violation for the surface that states only a
// count: `aiwf status` claiming errors the authoritative check does not
// raise sends a reader to the one surface that will tell them nothing is
// wrong.
func TestReadPathAgreementInvariant_RealBinary_ReportsAggregateSurfaceBlockingMoreThanTheGate(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)

	fake := writeFakeAiwfSurfaces(t, map[string]string{
		"check":              findingsEnvelope(),
		"check --fast":       findingsEnvelope(),
		"check --shape-only": findingsEnvelope(),
		"status":             `{"status":"ok","result":{"health":{"errors":3}}}`,
	})

	violations, err := readPathAgreementInvariant{}.Evaluate(fake, t.TempDir(), "constructed")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("violations = %+v, want exactly 1", violations)
	}
	if !strings.Contains(violations[0].Message, "aiwf status") || !strings.Contains(violations[0].Message, "3") {
		t.Errorf("violation message %q does not name the surface and its blocking count", violations[0].Message)
	}
}

func TestReadPathAgreementInvariant_ErrorsWhenTheBinaryCannotRun(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "no-such-aiwf-binary")
	if _, err := (readPathAgreementInvariant{}).Evaluate(missing, t.TempDir(), "label"); err == nil {
		t.Fatal("Evaluate() returned no error for a binary that cannot be launched")
	}
}

func TestRunAiwfStatusJSON_ErrorsWhenTheBinaryCannotRun(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "no-such-aiwf-binary")
	if _, err := runAiwfStatusJSON(missing, t.TempDir()); err == nil {
		t.Fatal("runAiwfStatusJSON() returned no error for a binary that cannot be launched")
	}
}

func TestParseStatusVerbEnvelope(t *testing.T) {
	t.Parallel()

	if _, err := parseStatusVerbEnvelope([]string{"status"}, []byte("not valid json")); err == nil {
		t.Error("parseStatusVerbEnvelope() accepted output that is not JSON")
	}

	env, err := parseStatusVerbEnvelope([]string{"status"}, []byte(`{"status":"ok","result":{"health":{"errors":4}}}`))
	if err != nil {
		t.Fatalf("parseStatusVerbEnvelope: %v", err)
	}
	if env.Status != "ok" || env.Result.Health.Errors != 4 {
		t.Errorf("parseStatusVerbEnvelope() = %+v, want status ok with 4 errors", env)
	}
}

// newCrossBranchReferenceRepo builds a repo whose tree cites an id that
// exists only on an unpublished local branch. A remote is configured and
// main is pushed to it, because the published-versus-local split reads
// remote-tracking refs — without any, the classification falls back to
// cross-branch-pending at warning severity and the blocking-versus-
// declined asymmetry under test never arises.
//
// The citing commit is written with plain git rather than through a
// verb: `aiwf edit-body` projects the same resolution rules and refuses
// the body this fixture needs.
func newCrossBranchReferenceRepo(t *testing.T, bin string) string {
	t.Helper()
	root := t.TempDir()
	remote := filepath.Join(root, "remote.git")
	dir := filepath.Join(root, "work")

	runGitOrFatal(t, root, "init", "-q", "--bare", remote)
	runGitOrFatal(t, root, "init", "-q", "--initial-branch=main", dir)
	runGitOrFatal(t, dir, "config", "user.email", "stresstest@example.com")
	runGitOrFatal(t, dir, "config", "user.name", "stresstest")
	runGitOrFatal(t, dir, "remote", "add", "origin", remote)

	epicEnv, err := runAiwfJSON(bin, dir, "add", "epic", "--title", "host epic", "--body", "b")
	if err != nil {
		t.Fatalf("add epic: %v", err)
	}
	runGitOrFatal(t, dir, "push", "-q", "origin", "main")

	runGitOrFatal(t, dir, "checkout", "-q", "-b", "side")
	gapEnv, err := runAiwfJSON(bin, dir, "add", "gap", "--title", "side only gap", "--body", "b")
	if err != nil {
		t.Fatalf("add gap: %v", err)
	}
	runGitOrFatal(t, dir, "checkout", "-q", "main")

	showEnv, err := runAiwfJSON(bin, dir, "show", epicEnv.Metadata.EntityID)
	if err != nil {
		t.Fatalf("show epic: %v", err)
	}
	epicPath := filepath.Join(dir, showEnv.Result.Path)
	appendToFileOrFatal(t, epicPath, "\nThis body cites "+gapEnv.Metadata.EntityID+", which lives only on the side branch.\n")
	runGitOrFatal(t, dir, "add", "-A")
	runGitOrFatal(t, dir, "commit", "-q", "-m", "cite a side-branch id")

	return dir
}

// appendToFileOrFatal appends text to an existing file.
func appendToFileOrFatal(t *testing.T, path, text string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("opening %s: %v", path, err)
	}
	if _, err := f.WriteString(text); err != nil {
		t.Fatalf("appending to %s: %v", path, err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("closing %s: %v", path, err)
	}
}

// findSurface returns the observation for the named surface, failing the
// test when it is absent.
func findSurface(t *testing.T, obs []readPathObservation, surface string) readPathObservation {
	t.Helper()
	for _, o := range obs {
		if o.Surface == surface {
			return o
		}
	}
	t.Fatalf("observeReadPaths() did not observe %q; got %+v", surface, obs)
	return readPathObservation{}
}

// findingsEnvelope renders a findings envelope carrying the given raw
// finding objects, so a stand-in surface's answer reads as the findings
// it states.
func findingsEnvelope(findings ...string) string {
	return `{"status":"ok","findings":[` + strings.Join(findings, ",") + `]}`
}

// writeFakeAiwfSurfaces writes an executable shell script standing in
// for `aiwf`, answering each surface with the stdout keyed by that
// surface's arguments (without the --format=json every call carries).
func writeFakeAiwfSurfaces(t *testing.T, byArgs map[string]string) string {
	t.Helper()
	var script strings.Builder
	script.WriteString("#!/bin/sh\ncase \"$*\" in\n")
	for args, stdout := range byArgs {
		script.WriteString("  '" + args + " --format=json') cat <<'EOF'\n" + stdout + "\nEOF\n  ;;\n")
	}
	script.WriteString("  *) echo \"unexpected surface: $*\" >&2; exit 3 ;;\nesac\n")

	path := filepath.Join(t.TempDir(), "aiwf")
	if err := testsupport.WriteExecutable(path, []byte(script.String())); err != nil {
		t.Fatalf("writing fake aiwf binary: %v", err)
	}
	return path
}
