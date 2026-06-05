package check

import (
	"strings"
	"testing"

	codespkg "github.com/23min/aiwf/internal/codes"
)

// reflog_walk_test.go — M-0161/AC-5 (G-0205) unit-level
// coverage of the RunOrphanedAICommits rule consumer per
// CLAUDE.md §"Test the seam, not just the layer" + §"Test
// untested code paths before declaring code paths 'done'".
//
// The E2E coverage at
// `internal/cli/integration/isolation_escape_force_push_scenarios_test.go`
// exercises the production wire-up; the unit tests below pin
// the rule's input/output contract against in-memory fixtures
// (no git, no fixture trees). Branch coverage:
//
//   - empty orphan slice → nil findings (early-return path).
//   - single orphan → one finding with hint text containing SHA
//     + branch + date + the recovery commands.
//   - duplicate SHA across two branches → one finding (per-SHA
//     dedup at the rule layer — the M-0161 reviewer flagged
//     this as untested when only the E2E happy path exercises
//     the gather; AC-5 wrap follow-up).
//   - acknowledged SHA via ackedSHAs map → zero findings
//     (cell-5 carve-out: the AC-5 E2E deferred the ack
//     composition to G-0226 / D-0020, but the rule-side
//     exemption must still be unit-tested so the per-SHA
//     contract stays load-bearing).
//   - empty-SHA entry filtered (defensive path).
//
// The fakeOracle pattern from internal/check/isolation_escape_test.go
// is the precedent — in-memory fixtures pin the algorithm; the
// real-git seam tests pin the integration. Both layers stay
// honest about which assertions they prove.

// TestRunOrphanedAICommits_AC5_EmptyOrphans pins the
// early-return path: nil/empty orphan slice yields no
// findings.
func TestRunOrphanedAICommits_AC5_EmptyOrphans(t *testing.T) {
	t.Parallel()
	if got := RunOrphanedAICommits(nil, nil); got != nil {
		t.Errorf("RunOrphanedAICommits(nil, nil) = %v; want nil", got)
	}
	if got := RunOrphanedAICommits([]OrphanedAICommit{}, nil); got != nil {
		t.Errorf("RunOrphanedAICommits([], nil) = %v; want nil", got)
	}
}

// TestRunOrphanedAICommits_AC5_SingleOrphan_EmitsWarningWithHint
// pins the load-bearing happy path: one orphan → one finding
// with the expected Code, Severity, EntityID, and hint text
// shape (SHA + branch + acknowledge-illegal recovery hint).
func TestRunOrphanedAICommits_AC5_SingleOrphan_EmitsWarningWithHint(t *testing.T) {
	t.Parallel()
	const sha = "af1051d1a27c7f30986bff60521c8e82269442d0"
	const branch = "epic/E-0033-pin-legal-kernel-verb-workflows-mechanically"
	const date = "2026-05-18 14:51:31 +0200"
	orphans := []OrphanedAICommit{{
		SHA:        sha,
		Branch:     branch,
		ReflogDate: date,
		EntityID:   "M-0120",
		Actor:      "ai/claude",
	}}
	got := RunOrphanedAICommits(orphans, nil)
	if len(got) != 1 {
		t.Fatalf("RunOrphanedAICommits returned %d findings; want 1\nfindings: %+v", len(got), got)
	}
	f := got[0]
	if f.Code != CodeIsolationEscapeOrphanedAICommit.ID {
		t.Errorf("Code = %q; want %q", f.Code, CodeIsolationEscapeOrphanedAICommit.ID)
	}
	if f.Severity != SeverityWarning {
		t.Errorf("Severity = %q; want %q (AC-5 body / M-0125 ratchet)", f.Severity, SeverityWarning)
	}
	if f.EntityID != "M-0120" {
		t.Errorf("EntityID = %q; want %q", f.EntityID, "M-0120")
	}
	if !strings.Contains(f.Message, sha[:8]) {
		t.Errorf("Message %q does not contain shortened SHA %q", f.Message, sha[:8])
	}
	if !strings.Contains(f.Message, branch) {
		t.Errorf("Message %q does not contain branch %q", f.Message, branch)
	}
	if !strings.Contains(f.Message, date) {
		t.Errorf("Message %q does not contain reflog date %q", f.Message, date)
	}
	if !strings.Contains(f.Hint, "aiwf acknowledge-illegal") {
		t.Errorf("Hint %q does not name the acknowledge-illegal recovery path", f.Hint)
	}
	if !strings.Contains(f.Hint, "git update-ref") {
		t.Errorf("Hint %q does not name the git update-ref restore path", f.Hint)
	}
}

// TestRunOrphanedAICommits_AC5_DuplicateSHADedup pins the
// per-SHA dedup at the rule layer: an orphan that appears on
// multiple ritual branches (e.g., the tip of two branches at
// different times) fires once, naming the first-seen branch.
// The doc comment at reflog_walk.go:70-74 names this as the
// rule-layer policy.
func TestRunOrphanedAICommits_AC5_DuplicateSHADedup(t *testing.T) {
	t.Parallel()
	const sha = "deadbeefcafe1234567890abcdef0987654321ab"
	orphans := []OrphanedAICommit{
		{SHA: sha, Branch: "epic/E-0001-engine", ReflogDate: "2026-06-01 10:00:00 +0000", EntityID: "E-0001", Actor: "ai/claude"},
		{SHA: sha, Branch: "milestone/M-0001-bootstrap", ReflogDate: "2026-06-02 11:00:00 +0000", EntityID: "E-0001", Actor: "ai/claude"},
	}
	got := RunOrphanedAICommits(orphans, nil)
	if len(got) != 1 {
		t.Fatalf("RunOrphanedAICommits returned %d findings; want 1 (per-SHA dedup)\nfindings: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "epic/E-0001-engine") {
		t.Errorf("first-seen-branch convention broken; Message %q does not contain first orphan's branch", got[0].Message)
	}
}

// TestRunOrphanedAICommits_AC5_AcknowledgedSHAExempted pins
// the per-SHA ack exemption — the cell-5 carve-out the AC-5
// E2E matrix defers (D-0020 / G-0226). Even though the verb-
// side path is unavailable, the rule-side exemption is the
// load-bearing contract: if ackedSHAs[sha] is true, the
// finding does not fire. A future verb extension landing
// G-0226 reuses this exemption rather than re-implementing
// the silence path.
func TestRunOrphanedAICommits_AC5_AcknowledgedSHAExempted(t *testing.T) {
	t.Parallel()
	const sha = "abc123def456789012345678901234567890abcd"
	orphans := []OrphanedAICommit{{
		SHA:        sha,
		Branch:     "epic/E-0001-engine",
		ReflogDate: "2026-06-01 12:00:00 +0000",
		EntityID:   "E-0001",
		Actor:      "ai/claude",
	}}
	acked := map[string]bool{sha: true}
	got := RunOrphanedAICommits(orphans, acked)
	if len(got) != 0 {
		t.Errorf("RunOrphanedAICommits with acked SHA returned %d findings; want 0\nfindings: %+v", len(got), got)
	}
}

// TestRunOrphanedAICommits_AC5_EmptyShaFiltered pins the
// defensive filter at reflog_walk.go:268: an orphan entry
// with an empty SHA does NOT produce a finding (would otherwise
// emit a meaningless "commit  was orphaned" message). Defends
// against a future gather-helper bug emitting empty entries.
func TestRunOrphanedAICommits_AC5_EmptyShaFiltered(t *testing.T) {
	t.Parallel()
	orphans := []OrphanedAICommit{
		{SHA: "", Branch: "epic/E-0001-engine", ReflogDate: "2026-06-01 12:00:00 +0000", EntityID: "E-0001", Actor: "ai/claude"},
		{SHA: "validSHA1234567890abcdef0987654321cafebabe", Branch: "epic/E-0001-engine", ReflogDate: "2026-06-02 13:00:00 +0000", EntityID: "E-0001", Actor: "ai/claude"},
	}
	got := RunOrphanedAICommits(orphans, nil)
	if len(got) != 1 {
		t.Fatalf("RunOrphanedAICommits returned %d findings; want 1 (empty-SHA filtered, valid SHA fires)\nfindings: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "validSHA") {
		t.Errorf("expected finding for valid SHA; Message: %q", got[0].Message)
	}
}

// TestRunOrphanedAICommits_AC5_CodeIsBranchChoreographyClass
// pins the code-class invariant per ADR-0011 / M-0123: the
// new code's Class is ClassBranchChoreography so the M-0158/
// AC-6 drift policy sees it.
func TestRunOrphanedAICommits_AC5_CodeIsBranchChoreographyClass(t *testing.T) {
	t.Parallel()
	if CodeIsolationEscapeOrphanedAICommit.Class != codespkg.ClassBranchChoreography {
		t.Errorf("CodeIsolationEscapeOrphanedAICommit.Class = %v; want ClassBranchChoreography (per AC-5 body + ADR-0011)", CodeIsolationEscapeOrphanedAICommit.Class)
	}
	if CodeIsolationEscapeOrphanedAICommit.ID != "isolation-escape-orphaned-ai-commit" {
		t.Errorf("CodeIsolationEscapeOrphanedAICommit.ID = %q; want %q", CodeIsolationEscapeOrphanedAICommit.ID, "isolation-escape-orphaned-ai-commit")
	}
}
