package check

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/scope"
)

// scopeStateCommit is `aiwf authorize <entityID> --pause` or `--resume`
// (state "paused" or "resumed"): an authorize commit that names no scope.
func scopeStateCommit(sha, entityID, state string) scope.Commit {
	return humanCommit(sha, "authorize", entityID, "human/peter", []gitops.Trailer{
		{Key: gitops.TrailerScope, Value: state},
		{Key: gitops.TrailerReason, Value: "hold"},
	})
}

// TestRunProvenance_AuthorizationPaused pins when an agent commit draws
// provenance-authorization-paused: when the scope its aiwf-authorized-by
// names is paused at that commit, as scope.Replay derives the scope's
// state from its entity's commits up to there. Each history holds one
// agent commit, the one carrying aiwf-authorized-by. A finding is an
// error against that commit's entity, reported once.
func TestRunProvenance_AuthorizationPaused(t *testing.T) {
	t.Parallel()
	tr := buildProvenanceTree(t)
	scopeA := strings.Repeat("a", 40)
	scopeB := strings.Repeat("b", 40)
	agentUnder := func(sha, auth string) scope.Commit {
		return agentCommit(sha, "promote", "M-0001", "ai/claude", "human/peter", auth, nil)
	}
	tests := []struct {
		name       string
		history    []scope.Commit
		wantPaused bool
	}{
		{
			// The resume after the agent commit leaves the scope active
			// at the end of history; the commit still landed during the pause.
			name: "a commit made while its scope is paused draws the finding",
			history: []scope.Commit{
				authorizeOpenedCommit(scopeA, "E-0001", "human/peter", "ai/claude"),
				scopeStateCommit("p000001", "E-0001", "paused"),
				agentUnder("c000001", scopeA),
				scopeStateCommit("r000001", "E-0001", "resumed"),
			},
			wantPaused: true,
		},
		{
			name: "a resume before the commit lifts the pause",
			history: []scope.Commit{
				authorizeOpenedCommit(scopeA, "E-0001", "human/peter", "ai/claude"),
				scopeStateCommit("p000001", "E-0001", "paused"),
				scopeStateCommit("r000001", "E-0001", "resumed"),
				agentUnder("c000001", scopeA),
			},
		},
		{
			// The agent's closing promote of E-0001 ends the scope. The
			// commit's own transition is not yet in force when it lands, so
			// it was made during the pause.
			name: "a commit that ends its own paused scope draws the finding",
			history: []scope.Commit{
				authorizeOpenedCommit(scopeA, "E-0001", "human/peter", "ai/claude"),
				scopeStateCommit("p000001", "E-0001", "paused"),
				agentCommit("c000001", "promote", "E-0001", "ai/claude", "human/peter", scopeA,
					[]gitops.Trailer{{Key: gitops.TrailerScopeEnds, Value: scopeA}}),
			},
			wantPaused: true,
		},
		{
			// The pause moves B, the most recently opened active scope.
			name: "a pause holds on the scope the replay moves",
			history: []scope.Commit{
				authorizeOpenedCommit(scopeA, "E-0001", "human/peter", "ai/claude"),
				authorizeOpenedCommit(scopeB, "E-0001", "human/peter", "ai/claude"),
				scopeStateCommit("p000001", "E-0001", "paused"),
				agentUnder("c000001", scopeB),
			},
			wantPaused: true,
		},
		{
			name: "a pause leaves the entity's other scopes active",
			history: []scope.Commit{
				authorizeOpenedCommit(scopeA, "E-0001", "human/peter", "ai/claude"),
				authorizeOpenedCommit(scopeB, "E-0001", "human/peter", "ai/claude"),
				scopeStateCommit("p000001", "E-0001", "paused"),
				agentUnder("c000001", scopeA),
			},
		},
		{
			// B, on M-0001, is opened after A; a pause of E-0001 that could
			// reach scopes on other entities would move B.
			name: "a pause leaves another entity's scopes alone",
			history: []scope.Commit{
				authorizeOpenedCommit(scopeA, "E-0001", "human/peter", "ai/claude"),
				authorizeOpenedCommit(scopeB, "M-0001", "human/peter", "ai/claude"),
				scopeStateCommit("p000001", "E-0001", "paused"),
				agentUnder("c000001", scopeB),
			},
		},
		{
			// The scope was opened before ids reached canonical width and
			// paused after; the replay reads both as the same entity's.
			name: "an entity's commits replay together whatever width their ids are written at",
			history: []scope.Commit{
				authorizeOpenedCommit(scopeA, "E-01", "human/peter", "ai/claude"),
				scopeStateCommit("p000001", "E-0001", "paused"),
				agentUnder("c000001", scopeA),
			},
			wantPaused: true,
		},
		{
			name: "a short aiwf-authorized-by is judged against the scope it resolves to",
			history: []scope.Commit{
				authorizeOpenedCommit(scopeA, "E-0001", "human/peter", "ai/claude"),
				scopeStateCommit("p000001", "E-0001", "paused"),
				agentUnder("c000001", scopeA[:7]),
			},
			wantPaused: true,
		},
		{
			// The end rides a commit on another entity, which the replay of
			// E-0001 does not read, so it still holds the scope paused.
			name: "a commit after its scope ended draws no paused finding",
			history: []scope.Commit{
				authorizeOpenedCommit(scopeA, "E-0001", "human/peter", "ai/claude"),
				scopeStateCommit("p000001", "E-0001", "paused"),
				humanCommit("e000001", "edit-body", "M-0002", "human/peter", []gitops.Trailer{
					{Key: gitops.TrailerScopeEnds, Value: scopeA},
				}),
				agentUnder("c000001", scopeA),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var paused []Finding
			for _, f := range RunProvenance(tt.history, tr, nil) {
				if f.Code == CodeProvenanceAuthorizationPaused {
					paused = append(paused, f)
				}
			}
			want := 0
			if tt.wantPaused {
				want = 1
			}
			if len(paused) != want {
				t.Fatalf("%s findings = %+v, want %d", CodeProvenanceAuthorizationPaused, paused, want)
			}
			var wantEntity string
			for _, c := range tt.history {
				if idx := indexCommitTrailersForProvenance(c.Trailers); idx[gitops.TrailerAuthorizedBy] != "" {
					wantEntity = idx[gitops.TrailerEntity]
				}
			}
			for _, f := range paused {
				if f.Severity != SeverityError || f.EntityID != wantEntity {
					t.Errorf("finding severity, entity = %s, %q; want %s, %q", f.Severity, f.EntityID, SeverityError, wantEntity)
				}
			}
		})
	}
}
