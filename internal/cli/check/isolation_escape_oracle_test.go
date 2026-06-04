package check

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
)

// isolation_escape_oracle_test.go — M-0161/AC-3 (G-0203) RED:
// unit-level pinning of the typed-error / per-ref-tolerance
// contract for newGitBranchOracle.
//
// AC-3 contract (per body and D-0019):
//
//   - The oracle's BranchOracle interface grows OracleErrors()
//     []check.OracleErr, returning per-ref construction failures
//     keyed by ref name + capability tag + underlying error.
//   - newGitBranchOracle returns a non-nil oracle even when some
//     refs fail their first-parent walk; failed refs accumulate
//     into OracleErrors(); healthy refs continue to populate
//     branchesBySHA.
//   - Pre-AC-3 behavior was "any per-ref failure → nil oracle +
//     error", causing RunProvenanceCheck to silently swallow and
//     the whole isolation-escape rule to skip for the repo.
//     Post-AC-3 the rule still runs against every healthy ref.
//
// These tests fail in RED because OracleErrors() is not a method
// on the gitBranchOracle (nor on the check.BranchOracle interface)
// today, and newGitBranchOracle still returns nil on per-ref
// failure.

// TestNewGitBranchOracle_AC3_AllHealthy_NoErrors pins the happy
// path: a repo with only healthy ritual refs returns an oracle
// whose OracleErrors() is empty. This is the baseline the
// per-ref tolerance contract degrades from.
func TestNewGitBranchOracle_AC3_AllHealthy_NoErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := setupAC3RepoAllHealthy(t)

	oracle, err := newGitBranchOracle(ctx, root)
	if err != nil {
		t.Fatalf("newGitBranchOracle on healthy repo: %v", err)
	}
	if oracle == nil {
		t.Fatal("newGitBranchOracle returned nil oracle on healthy repo")
	}
	errs := oracle.OracleErrors()
	if len(errs) != 0 {
		t.Errorf("OracleErrors() = %d entries on healthy repo; want 0 (no per-ref failures expected)\nentries: %+v", len(errs), errs)
	}
}

// TestNewGitBranchOracle_AC3_PerRefTolerance_OneCorruptedRef
// pins the load-bearing AC-3 claim: a single ref whose
// first-parent walk fails does NOT disable the oracle for the
// whole repo. The healthy ref(s) continue to populate the
// per-SHA index; the corrupt ref surfaces as one OracleErr
// entry naming the ref and wrapping the underlying error.
//
// Fixture: corrupt the tip object file of the stale ref so
//   - `git for-each-ref refs/heads/` still emits the ref (the
//     ref file on disk is valid; for-each-ref reads ref files,
//     not the pointed-to object)
//   - `git rev-list --first-parent <stale ref>` fails because
//     the object file is unreadable
//
// Pre-AC-3 the per-ref failure aborts the entire indexing loop
// at internal/cli/check/isolation_escape_oracle.go:64-67 and
// returns (nil, error). Post-AC-3 the failure is captured into
// the typed slice and indexing continues for every other ref.
func TestNewGitBranchOracle_AC3_PerRefTolerance_OneCorruptedRef(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root, healthyHeadSHA, corruptRef := setupAC3RepoWithCorruptRef(t)

	oracle, err := newGitBranchOracle(ctx, root)
	if err != nil {
		t.Fatalf("newGitBranchOracle returned error %q; AC-3 contract: per-ref failures accumulate into OracleErrors() and construction succeeds when at least one ref enumerated cleanly", err)
	}
	if oracle == nil {
		t.Fatal("newGitBranchOracle returned nil oracle; AC-3 contract: non-nil oracle even with per-ref failures")
	}

	branches := oracle.FirstParentBranches(healthyHeadSHA)
	if len(branches) == 0 {
		t.Errorf("FirstParentBranches(healthyHeadSHA=%s) returned empty; AC-3 contract: healthy refs continue to populate branchesBySHA even when sibling refs fail", healthyHeadSHA[:7])
	}

	errs := oracle.OracleErrors()
	if len(errs) == 0 {
		t.Fatal("OracleErrors() returned empty slice; AC-3 contract: corrupt ref surfaces as a typed entry, not silent skip")
	}
	var found bool
	for _, e := range errs {
		if e.Ref == corruptRef {
			found = true
			if e.Err == nil {
				t.Errorf("OracleErr for ref %q has nil Err; AC-3 contract: Err wraps the underlying git failure for operator hint text", corruptRef)
			}
		}
	}
	if !found {
		t.Errorf("OracleErrors() did not contain entry for corrupt ref %q\nentries: %+v", corruptRef, errs)
	}
}

// setupAC3RepoAllHealthy builds a fresh repo with main + one
// ritual ref, both healthy. Returns the repo root.
func setupAC3RepoAllHealthy(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	gitRun(t, root, "branch", "-M", "main")

	writeFile(t, root, "seed.md", "seed\n")
	gitRun(t, root, "add", ".")
	gitRun(t, root, "commit", "-m", "baseline")

	gitRun(t, root, "checkout", "-b", "epic/E-0001-engine")
	writeFile(t, root, "epic.md", "epic\n")
	gitRun(t, root, "add", ".")
	gitRun(t, root, "commit", "-m", "epic work")

	gitRun(t, root, "checkout", "main")
	return root
}

// setupAC3RepoWithCorruptRef builds a repo with main + a
// healthy ritual ref (epic/E-0001-engine) + a corrupt ritual
// ref (epic/E-9999-stale) whose tip object file is unreadable.
// Returns root, the healthy ref's HEAD SHA, and the corrupt
// ref's name.
//
// Fixture mechanic: after committing on the stale ref, overwrite
// the loose object file at .git/objects/<aa>/<bb...> with
// random bytes. for-each-ref still emits the ref (ref files are
// untouched); rev-list --first-parent fails when it tries to
// decode the zlib-compressed object.
func setupAC3RepoWithCorruptRef(t *testing.T) (root, healthyHeadSHA, corruptRef string) {
	t.Helper()
	ctx := context.Background()
	root = t.TempDir()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	gitRun(t, root, "branch", "-M", "main")

	writeFile(t, root, "seed.md", "seed\n")
	gitRun(t, root, "add", ".")
	gitRun(t, root, "commit", "-m", "baseline")

	gitRun(t, root, "checkout", "-b", "epic/E-0001-engine")
	writeFile(t, root, "epic.md", "epic\n")
	gitRun(t, root, "add", ".")
	gitRun(t, root, "commit", "-m", "epic work")
	healthyHeadSHA = gitOutput(t, root, "rev-parse", "HEAD")

	gitRun(t, root, "checkout", "-b", "epic/E-9999-stale", "main")
	writeFile(t, root, "stale.md", "stale\n")
	gitRun(t, root, "add", ".")
	gitRun(t, root, "commit", "-m", "stale work")
	staleHead := gitOutput(t, root, "rev-parse", "HEAD")

	// Corrupt the stale ref's tip object so rev-list --first-parent
	// fails while for-each-ref still emits the ref. The loose
	// object lives at .git/objects/<sha[:2]>/<sha[2:]>; overwriting
	// with raw bytes breaks zlib decoding. Git writes loose
	// objects mode 0o444 (read-only) so chmod is required first.
	objPath := filepath.Join(root, ".git", "objects", staleHead[:2], staleHead[2:])
	if err := os.Chmod(objPath, 0o644); err != nil {
		t.Fatalf("chmod object %s: %v", objPath, err)
	}
	if err := os.WriteFile(objPath, []byte("garbage-not-zlib\n"), 0o644); err != nil {
		t.Fatalf("corrupt object %s: %v", objPath, err)
	}

	gitRun(t, root, "checkout", "main")
	corruptRef = "epic/E-9999-stale"
	return root, healthyHeadSHA, corruptRef
}

// gitRun executes a git subcommand in root, t.Fatal on error.
func gitRun(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// gitOutput returns the trimmed stdout of a git subcommand in
// root, t.Fatal on error.
func gitOutput(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

// writeFile writes a file at the given repo-relative path.
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, rel)
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}

// AC-4 (G-0204) unit tests — shallow-clone detection.
//
// AC-4 contract (per body): newGitBranchOracle detects shallow
// state via `git rev-parse --is-shallow-repository`; on shallow
// the per-SHA map is left empty (no false positives from
// half-walked first-parent indexes) and a typed OracleErr with
// Capability="shallow-clone" surfaces so the consumer
// (RunProvenanceCheck) emits isolation-escape-shallow-clone at
// warning severity. Composes with AC-3's typed-error contract.
//
// Fixture mechanic: write any valid SHA into .git/shallow; git
// treats the presence of a non-empty .git/shallow file as making
// the repo shallow. Faster + more deterministic than spinning up
// a `git clone --depth=N` from a richer source.

// TestNewGitBranchOracle_AC4_ShallowDetection_EmptyMapPlusTypedError
// pins the AC-4 load-bearing claim: when the repo is shallow,
// the oracle leaves branchesBySHA empty AND surfaces a typed
// OracleErr with Capability="shallow-clone".
//
// RED: newGitBranchOracle does not check is-shallow-repository
// today; it walks rev-list normally and populates the index from
// whatever the shallow boundary lets it see. Tests fail because
// (a) no shallow-clone OracleErr is emitted and (b) the per-SHA
// map is populated from the truncated walk.
func TestNewGitBranchOracle_AC4_ShallowDetection_EmptyMapPlusTypedError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := setupAC4ShallowRepo(t)

	oracle, err := newGitBranchOracle(ctx, root)
	if err != nil {
		t.Fatalf("newGitBranchOracle returned error %q; AC-4 contract: shallow detection accumulates a typed OracleErr and construction succeeds", err)
	}
	if oracle == nil {
		t.Fatal("newGitBranchOracle returned nil; AC-4 contract: non-nil oracle even on shallow repos")
	}

	// The per-SHA map must be empty — shallow truncation would
	// otherwise yield silently-incomplete classifications.
	if len(oracle.branchesBySHA) != 0 {
		t.Errorf("AC-4: branchesBySHA has %d entries on shallow repo; want 0 (fail-shut on shallow per AC-4 body)", len(oracle.branchesBySHA))
	}

	errs := oracle.OracleErrors()
	if len(errs) == 0 {
		t.Fatal("AC-4: OracleErrors() empty on shallow repo; want >= 1 entry with Capability=\"shallow-clone\"")
	}
	var foundShallow bool
	for _, e := range errs {
		if e.Capability == "shallow-clone" {
			foundShallow = true
			if e.Err == nil {
				t.Errorf("AC-4: shallow-clone OracleErr has nil Err; want non-nil for diagnostic surface")
			}
		}
	}
	if !foundShallow {
		t.Errorf("AC-4: OracleErrors() does not contain entry with Capability=\"shallow-clone\"\nentries: %+v", errs)
	}
}

// TestNewGitBranchOracle_AC4_NonShallow_NoShallowEntry pins the
// symmetric path: a non-shallow repo produces NO shallow-clone
// OracleErr regardless of other state.
func TestNewGitBranchOracle_AC4_NonShallow_NoShallowEntry(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := setupAC3RepoAllHealthy(t)

	oracle, err := newGitBranchOracle(ctx, root)
	if err != nil {
		t.Fatalf("newGitBranchOracle: %v", err)
	}
	if oracle == nil {
		t.Fatal("oracle is nil")
	}
	for _, e := range oracle.OracleErrors() {
		if e.Capability == "shallow-clone" {
			t.Errorf("AC-4: shallow-clone OracleErr appeared on non-shallow repo (entry: %+v)", e)
		}
	}
}

// setupAC4ShallowRepo builds a healthy repo (main + one ritual
// branch with commits) then forces shallow state by writing the
// HEAD SHA into .git/shallow. Returns root.
//
// Git's contract for shallow detection: .git/shallow is a list
// of SHAs marking the "shallow boundary" — commits beyond these
// are excluded from the local object store. The presence of a
// non-empty .git/shallow file is what
// `git rev-parse --is-shallow-repository` reports on. The
// content's exact correctness doesn't matter for the test
// (we're not exercising shallow rev-list logic, just the
// is-shallow boolean).
func setupAC4ShallowRepo(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	gitRun(t, root, "branch", "-M", "main")

	writeFile(t, root, "seed.md", "seed\n")
	gitRun(t, root, "add", ".")
	gitRun(t, root, "commit", "-m", "baseline")

	gitRun(t, root, "checkout", "-b", "epic/E-0001-engine")
	writeFile(t, root, "epic.md", "epic\n")
	gitRun(t, root, "add", ".")
	gitRun(t, root, "commit", "-m", "epic work")

	headSHA := gitOutput(t, root, "rev-parse", "HEAD")
	gitRun(t, root, "checkout", "main")

	shallowPath := filepath.Join(root, ".git", "shallow")
	if err := os.WriteFile(shallowPath, []byte(headSHA+"\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", shallowPath, err)
	}
	return root
}
