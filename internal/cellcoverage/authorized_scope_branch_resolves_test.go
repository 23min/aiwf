package cellcoverage

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/verb"
)

// authorized_scope_branch_resolves_test.go — M-0159/AC-7: pin the
// G-0213 invariant that the cellcoverage AuthorizeScope fixture's
// stamped `aiwf-branch:` trailer value MUST resolve to a real
// branch ref in the fixture's tmp git repo.
//
// G-0213's failure mode (pre-fix):
//
//   The fixture at authorized_scope.go stamped a fictional
//   `epic/E-NNNN-cellcoverage-fixture` branch name into the
//   trailer purely to satisfy the verb.Authorize preflight's
//   "non-empty when actor is ai/" requirement. The branch did
//   NOT actually exist in the fixture's git repo. Every kernel
//   rule that reads `aiwf-branch:` against a "must resolve"
//   check would have silently broken all M-0125 positive cell
//   tests the moment such a rule landed — M-0159 / M-0161 future
//   work was sequencing-blocked behind addressing this.
//
// The fix per G-0213 Option 1 (chosen at AC-7 design call):
// AuthorizeScope creates the branch via `git branch <name>` in
// the fixture's tmp repo BEFORE invoking verb.Authorize, so the
// trailer value resolves end-to-end. Option 2 (sentinel trailer)
// was rejected for coupling production rule code to a fixture
// marker; Option 3 (rule fail-open on empty BranchOracle) was
// rejected for trading a real safety property for fixture
// convenience.
//
// This test pins the invariant at the fixture-contract level:
// any change that breaks branch resolvability of the fixture's
// aiwf-branch trailer value fails CI.

// TestCellFixture_AuthorizeScope_AIWFBranchTrailerResolves drives
// AuthorizeScope end-to-end and asserts that the named branch
// resolves via `git rev-parse --verify refs/heads/<name>`. RED
// today (fixture stamps the trailer but does not create the
// branch); GREEN after AuthorizeScope is modified to create the
// branch in the fixture's git repo.
func TestCellFixture_AuthorizeScope_AIWFBranchTrailerResolves(t *testing.T) {
	t.Parallel()
	f := NewCellFixture(t)

	// Bring an entity up to active so AuthorizeScope's verb path
	// passes the entity-state preflight.
	f.Must(verb.Add(f.ctx, f.Tree(), entity.KindEpic, "Branch-Resolve Epic", testActor, verb.AddOptions{}))
	f.Must(verb.Promote(f.ctx, f.Tree(), "E-0001", entity.StatusActive, testActor, "", false, verb.PromoteOptions{}))

	s := f.AuthorizeScope(t, "E-0001", "ai/claude")
	if s == nil {
		t.Fatal("AuthorizeScope returned nil scope (smoke check)")
	}

	// Read the aiwf-branch trailer value from the authorize commit.
	// This is the value a branch-resolution kernel rule would lift
	// from the scope opener — we pin the invariant against the
	// actual trailer value, not the fixture's input template
	// (cosmetic identity but worth keeping precise: the load-bearing
	// value is what's RECORDED in the commit, not what was passed
	// to the verb).
	branchValue := readAIWFBranchTrailer(t, f.ctx, f.Root, s.AuthSHA)
	if branchValue == "" {
		t.Fatalf("authorize commit %s has empty or absent aiwf-branch trailer; G-0213 invariant cannot be evaluated", s.AuthSHA)
	}

	// G-0213 invariant: the trailer value MUST resolve to a real
	// branch ref in the fixture's git repo. A failure here means
	// the fixture is stamping a fictional branch name, which would
	// silently break every M-0125 cell test the moment any
	// branch-resolution rule lands.
	cmd := exec.CommandContext(f.ctx, "git", "rev-parse", "--verify", "refs/heads/"+branchValue)
	cmd.Dir = f.Root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("aiwf-branch trailer value %q does not resolve to a real branch ref in the fixture repo: %v\noutput: %s\n\nG-0213 / M-0159/AC-7: AuthorizeScope must create the branch (e.g., via `git branch <name>`) before stamping the trailer so any future branch-resolution rule does not silently break every M-0125 cell test.",
			branchValue, err, strings.TrimSpace(string(out)))
	}
}

// readAIWFBranchTrailer extracts the aiwf-branch trailer value from
// the named commit via `git log -1 --pretty=%(trailers:...)`. Returns
// the empty string if the trailer is absent or empty; the caller
// distinguishes that from "trailer present with value X" so the
// invariant test can report the precise failure mode.
//
// Uses the same `%(trailers:key=<k>,valueonly=true,unfold=true)`
// format the AC-4 and AC-5 scenarios established for structural
// trailer queries — keeps the assertion shape consistent with the
// branch_scenarios_*_test.go convention.
func readAIWFBranchTrailer(t *testing.T, ctx context.Context, root, sha string) string {
	t.Helper()
	cmd := exec.CommandContext(ctx, "git", "log", "-1",
		"--pretty=%(trailers:key="+gitops.TrailerBranch+",valueonly=true,unfold=true)",
		sha)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("read aiwf-branch trailer from %s: %v", sha, err)
	}
	return strings.TrimSpace(string(out))
}
