package stresstest

import "testing"

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
		wantViolations           int
	}{
		{
			name:                     "clean: scope stays active through the sweep, pause finds it, check catches the structural anomaly",
			preScopeState:            "active",
			postScopeState:           "active",
			pauseStatus:              "ok",
			archivedNotTerminalFound: true,
			wantViolations:           0,
		},
		{
			name:                     "the scope was never actually active before the sweep — the scenario's premise did not hold",
			preScopeState:            "paused",
			postScopeState:           "active",
			pauseStatus:              "ok",
			archivedNotTerminalFound: true,
			wantViolations:           1,
		},
		{
			name:                     "the scope's state changed or vanished after the sweep",
			preScopeState:            "active",
			postScopeState:           "ended",
			pauseStatus:              "ok",
			archivedNotTerminalFound: true,
			wantViolations:           1,
		},
		{
			name:                     "pause could not find the still-open scope after the sweep",
			preScopeState:            "active",
			postScopeState:           "active",
			pauseStatus:              "error",
			archivedNotTerminalFound: true,
			wantViolations:           1,
		},
		{
			name:                     "aiwf check silently failed to flag the non-terminal child riding along into archive/",
			preScopeState:            "active",
			postScopeState:           "active",
			pauseStatus:              "ok",
			archivedNotTerminalFound: false,
			wantViolations:           1,
		},
		{
			name:                     "every check fails at once",
			preScopeState:            "paused",
			postScopeState:           "ended",
			pauseStatus:              "error",
			archivedNotTerminalFound: false,
			wantViolations:           4,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyArchiveDuringActiveScope(tc.preScopeState, tc.postScopeState, tc.pauseStatus, tc.archivedNotTerminalFound)
			if len(got) != tc.wantViolations {
				t.Errorf("violations = %d (%+v), want %d", len(got), got, tc.wantViolations)
			}
		})
	}
}
