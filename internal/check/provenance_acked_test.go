package check

import (
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/scope"
	"github.com/23min/aiwf/internal/tree"
)

// provenance_acked_test.go pins M-0292's scoping claim at the unit
// boundary: an acknowledged commit reports no provenance finding, and
// an unacknowledged commit is untouched by the acknowledgment of
// another.
//
// The integration tests drive the same property through the real verb
// and the real gather layer, which is what proves the wiring. These
// reach RunProvenance directly, because that is where the scoping
// decision lives, and they enumerate its codes so the boundary is read
// off a list rather than inferred.
//
// The enumeration is a reader's aid, not the guarantee. The skip
// precedes all three rule groups, so a code is cleared by construction
// rather than by appearing below — which is why a code added later
// needs no row here to be ratifiable.

// offendingTrailerSets names each provenance code raised by a
// single commit's own trailers, together with a trailer set that
// raises it. The codes needing cross-commit state — an authorize
// opener, a tree to resolve reachability against — take a companion
// commit and so are covered by
// TestRunProvenance_AcknowledgedCommitClearsCrossCommitRules below
// rather than forced into this table.
var offendingTrailerSets = []struct {
	code     string
	trailers []gitops.Trailer
}{
	{
		code: CodeProvenanceActorMalformed,
		trailers: []gitops.Trailer{
			{Key: gitops.TrailerActor, Value: "notaroleid"},
			{Key: gitops.TrailerPrincipal, Value: "human/p"},
		},
	},
	{
		code: CodeProvenancePrincipalNonHuman,
		trailers: []gitops.Trailer{
			{Key: gitops.TrailerActor, Value: "ai/claude"},
			{Key: gitops.TrailerPrincipal, Value: "ai/claude"},
		},
	},
	{
		code: CodeProvenanceOnBehalfOfNonHuman,
		trailers: []gitops.Trailer{
			{Key: gitops.TrailerActor, Value: "human/p"},
			{Key: gitops.TrailerOnBehalfOf, Value: "ai/claude"},
			{Key: gitops.TrailerAuthorizedBy, Value: "abcdef1234567"},
		},
	},
	{
		code: CodeProvenanceAuthorizedByMalformed,
		trailers: []gitops.Trailer{
			{Key: gitops.TrailerActor, Value: "human/p"},
			{Key: gitops.TrailerOnBehalfOf, Value: "human/p"},
			{Key: gitops.TrailerAuthorizedBy, Value: "nothex"},
		},
	},
	{
		code: CodeProvenanceForceNonHuman,
		trailers: []gitops.Trailer{
			{Key: gitops.TrailerActor, Value: "ai/claude"},
			{Key: gitops.TrailerPrincipal, Value: "human/p"},
			{Key: gitops.TrailerForce, Value: "reason"},
			{Key: gitops.TrailerVerb, Value: "promote"},
			{Key: gitops.TrailerOnBehalfOf, Value: "human/p"},
			{Key: gitops.TrailerAuthorizedBy, Value: "abcdef1234567"},
		},
	},
	{
		code: CodeProvenanceAuditOnlyNonHuman,
		trailers: []gitops.Trailer{
			{Key: gitops.TrailerActor, Value: "ai/claude"},
			{Key: gitops.TrailerPrincipal, Value: "human/p"},
			{Key: gitops.TrailerAuditOnly, Value: "reason"},
			{Key: gitops.TrailerVerb, Value: "promote"},
			{Key: gitops.TrailerOnBehalfOf, Value: "human/p"},
			{Key: gitops.TrailerAuthorizedBy, Value: "abcdef1234567"},
		},
	},
	{
		code: CodeProvenanceTrailerIncoherent,
		trailers: []gitops.Trailer{
			{Key: gitops.TrailerActor, Value: "human/p"},
			{Key: gitops.TrailerPrincipal, Value: "human/p"},
		},
	},
	{
		code: CodeProvenanceNoActiveScope,
		trailers: []gitops.Trailer{
			{Key: gitops.TrailerActor, Value: "ai/claude"},
			{Key: gitops.TrailerPrincipal, Value: "human/p"},
			{Key: gitops.TrailerVerb, Value: "promote"},
		},
	},
	{
		code: CodeProvenanceAuthorizationMissing,
		trailers: []gitops.Trailer{
			{Key: gitops.TrailerActor, Value: "human/p"},
			{Key: gitops.TrailerOnBehalfOf, Value: "human/p"},
			{Key: gitops.TrailerAuthorizedBy, Value: "abcdef1234567"},
		},
	},
}

// crossCommitOffender is a fixture for a rule that judges a commit
// against the history around it, so it needs a companion commit and a
// tree rather than one commit's trailers.
type crossCommitOffender struct {
	code     string
	offender string
	commits  []scope.Commit
	tree     *tree.Tree
}

// crossCommitOffenders builds the fixtures for the two codes
// provenanceAuthorizationFindings raises. Shared by the scoping test
// and by the predicate-to-rules tie in hint_ratification_test.go, so
// the emitted set is derived from one definition rather than two that
// can drift.
func crossCommitOffenders(t *testing.T) []crossCommitOffender {
	t.Helper()
	authSHA := strings.Repeat("a", 40)
	return []crossCommitOffender{
		{
			code:     CodeProvenanceAuthorizationEnded,
			offender: "dddd444",
			tree:     buildProvenanceTree(t),
			commits: []scope.Commit{
				authorizeOpenedCommit(authSHA, "E-0001", "human/peter", "ai/claude"),
				agentCommit("bbbb222", "promote", "M-0001", "ai/claude", "human/peter", authSHA, nil),
				agentCommit("cccc333", "promote", "E-0001", "ai/claude", "human/peter", authSHA, []gitops.Trailer{
					{Key: gitops.TrailerScopeEnds, Value: authSHA},
				}),
				agentCommit("dddd444", "promote", "M-0002", "ai/claude", "human/peter", authSHA, nil),
			},
		},
		{
			code:     CodeProvenanceAuthorizationOutOfScope.ID,
			offender: "bbbb222",
			tree:     buildProvenanceTree(t),
			commits: []scope.Commit{
				authorizeOpenedCommit(authSHA, "E-0009", "human/peter", "ai/claude"),
				agentCommit("bbbb222", "promote", "M-0001", "ai/claude", "human/peter", authSHA, nil),
			},
		},
	}
}

// codesFrom collapses findings to their sorted, de-duplicated code set.
func codesFrom(findings []Finding) []string {
	seen := map[string]bool{}
	var out []string
	for i := range findings {
		if !seen[findings[i].Code] {
			seen[findings[i].Code] = true
			out = append(out, findings[i].Code)
		}
	}
	sort.Strings(out)
	return out
}

// TestRunProvenance_AcknowledgedCommitReportsNothing walks every
// offending trailer set and asserts two things per set: unacknowledged
// it raises the named code, and acknowledged it raises nothing at all.
//
// The first half is the positive control — without it a passing test
// cannot distinguish "the acknowledgment cleared the finding" from "the
// fixture never raised one". The second half is the scoping claim, and
// it is deliberately "nothing" rather than "not this code": an
// acknowledgment that cleared the named code while another still fired
// on the same commit would leave the push blocked, which is the defect
// this milestone exists to remove.
func TestRunProvenance_AcknowledgedCommitReportsNothing(t *testing.T) {
	t.Parallel()
	for _, tc := range offendingTrailerSets {
		t.Run(tc.code, func(t *testing.T) {
			t.Parallel()
			const sha = "1111111111111111111111111111111111111111"
			commits := []scope.Commit{{SHA: sha, Trailers: tc.trailers}}

			unacked := codesFrom(RunProvenance(commits, nil, nil))
			if !slices.Contains(unacked, tc.code) {
				t.Fatalf("fixture for %s raised %v; it must raise the code it names or the "+
					"acknowledged half below asserts nothing", tc.code, unacked)
			}

			acked := RunProvenance(commits, nil, map[string]bool{sha: true})
			if len(acked) != 0 {
				t.Errorf("acknowledged commit still reports %v; an acknowledgment is a judgment "+
					"about the commit, so every provenance finding against it clears", codesFrom(acked))
			}
		})
	}
}

// TestRunProvenance_AcknowledgedCommitClearsCrossCommitRules covers
// the two codes the table above cannot reach: they are raised by
// provenanceAuthorizationFindings against cross-commit state — a scope
// that has ended, a target outside the scope-entity's reach — so each
// needs an opener commit and a tree rather than one commit's trailers.
//
// The fixtures are the ones the baseline rule tests use, re-run with
// the offending commit acknowledged.
func TestRunProvenance_AcknowledgedCommitClearsCrossCommitRules(t *testing.T) {
	t.Parallel()
	for _, tc := range crossCommitOffenders(t) {
		t.Run(tc.code, func(t *testing.T) {
			t.Parallel()
			tr, commits := tc.tree, tc.commits

			if got := RunProvenance(commits, tr, nil); !hasFinding(got, tc.code) {
				t.Fatalf("fixture raised %v, want it to include %s", findingCodes(got), tc.code)
			}

			acked := RunProvenance(commits, tr, map[string]bool{tc.offender: true})
			for i := range acked {
				if acked[i].Code == tc.code {
					t.Errorf("%s still fires against the acknowledged commit %s: %s",
						tc.code, tc.offender, acked[i].Message)
				}
			}
		})
	}
}

// TestRunProvenance_AcknowledgmentDoesNotReachAnotherCommit is the
// closed-set half: acknowledging one SHA must not silence an
// identically-shaped sibling. Without it, a guard that ignored the map
// key — silencing whenever any acknowledgment exists — would pass every
// assertion above.
func TestRunProvenance_AcknowledgmentDoesNotReachAnotherCommit(t *testing.T) {
	t.Parallel()
	for _, tc := range offendingTrailerSets {
		t.Run(tc.code, func(t *testing.T) {
			t.Parallel()
			const ackedSHA = "1111111111111111111111111111111111111111"
			const siblingSHA = "2222222222222222222222222222222222222222"
			commits := []scope.Commit{
				{SHA: ackedSHA, Trailers: tc.trailers},
				{SHA: siblingSHA, Trailers: tc.trailers},
			}

			got := RunProvenance(commits, nil, map[string]bool{ackedSHA: true})
			if !slices.Contains(codesFrom(got), tc.code) {
				t.Errorf("acknowledging %s silenced the sibling %s too; got %v, want %s still reported",
					ackedSHA[:7], siblingSHA[:7], codesFrom(got), tc.code)
			}
			for i := range got {
				if strings.Contains(got[i].Message, ackedSHA[:7]) {
					t.Errorf("a finding still names the acknowledged commit %s: %s", ackedSHA[:7], got[i].Message)
				}
			}
		})
	}
}
