package stresstest

import (
	"strings"
	"testing"
)

// archive_during_active_scope_classify_test.go pins
// classifyArchiveDuringActiveScope — the pure decision logic behind
// ArchiveDuringActiveScopeScenario (M-0243/AC-3) — against fabricated
// outcomes, so every branch is exercised deterministically rather
// than depending on a real archive sweep's exact behavior.

func TestClassifyArchiveDuringActiveScope(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                     string
		preScopeState            string
		postScopeState           string
		pauseStatus              string
		archivedNotTerminalFound bool
		wantSubstrings           []string // nil means no violations expected
	}{
		{
			name:                     "clean: scope stays active through the sweep, pause finds it, check catches the structural anomaly",
			preScopeState:            "active",
			postScopeState:           "active",
			pauseStatus:              "ok",
			archivedNotTerminalFound: true,
			wantSubstrings:           nil,
		},
		{
			name:                     "the scope was never actually active before the sweep — the scenario's premise did not hold",
			preScopeState:            "paused",
			postScopeState:           "active",
			pauseStatus:              "ok",
			archivedNotTerminalFound: true,
			wantSubstrings:           []string{"the scenario's premise did not hold"},
		},
		{
			name:                     "the scope's state changed or vanished after the sweep",
			preScopeState:            "active",
			postScopeState:           "ended",
			pauseStatus:              "ok",
			archivedNotTerminalFound: true,
			wantSubstrings:           []string{"state changed after the sweep"},
		},
		{
			name:                     "pause could not find the still-open scope after the sweep",
			preScopeState:            "active",
			postScopeState:           "active",
			pauseStatus:              "error",
			archivedNotTerminalFound: true,
			wantSubstrings:           []string{"could not act on the still-open scope"},
		},
		{
			name:                     "aiwf check silently failed to flag the non-terminal child riding along into archive/",
			preScopeState:            "active",
			postScopeState:           "active",
			pauseStatus:              "ok",
			archivedNotTerminalFound: false,
			wantSubstrings:           []string{"did not flag the non-terminal child"},
		},
		{
			name:                     "every check fails at once",
			preScopeState:            "paused",
			postScopeState:           "ended",
			pauseStatus:              "error",
			archivedNotTerminalFound: false,
			wantSubstrings: []string{
				"the scenario's premise did not hold",
				"state changed after the sweep",
				"could not act on the still-open scope",
				"did not flag the non-terminal child",
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyArchiveDuringActiveScope(tc.preScopeState, tc.postScopeState, tc.pauseStatus, tc.archivedNotTerminalFound)
			if len(got) != len(tc.wantSubstrings) {
				t.Fatalf("violations = %+v, want %d matching %v", got, len(tc.wantSubstrings), tc.wantSubstrings)
			}
			for i, want := range tc.wantSubstrings {
				if !strings.Contains(got[i].Message, want) {
					t.Errorf("violation[%d] = %q, want it to contain %q", i, got[i].Message, want)
				}
			}
		})
	}
}
