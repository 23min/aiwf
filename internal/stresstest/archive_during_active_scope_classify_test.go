package stresstest

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/verb"
)

// archive_during_active_scope_classify_test.go pins
// classifyArchiveDuringActiveScope — the pure decision logic behind
// ArchiveDuringActiveScopeScenario (M-0243/AC-3; updated by
// M-0244/AC-2's G-0393 sweep) — against fabricated outcomes, so every
// branch is exercised deterministically rather than depending on a
// real promote attempt's exact behavior.

func TestClassifyArchiveDuringActiveScope(t *testing.T) {
	t.Parallel()
	const wrongCode = "some-other-code"
	tests := []struct {
		name             string
		preScopeState    string
		promoteStatus    string
		promoteErrorCode string
		epicStatusAfter  string
		postScopeState   string
		wantSubstrings   []string // nil means no violations expected
	}{
		{
			name:             "clean: scope active beforehand, promote refused with the right code, nothing else moved",
			preScopeState:    "active",
			promoteStatus:    "error",
			promoteErrorCode: verb.CodeEpicPromoteNonTerminalChildren.ID,
			epicStatusAfter:  "active",
			postScopeState:   "active",
			wantSubstrings:   nil,
		},
		{
			name:             "the scope was never actually active before the attempt — the scenario's premise did not hold",
			preScopeState:    "paused",
			promoteStatus:    "error",
			promoteErrorCode: verb.CodeEpicPromoteNonTerminalChildren.ID,
			epicStatusAfter:  "active",
			postScopeState:   "active",
			wantSubstrings:   []string{"the scenario's premise did not hold"},
		},
		{
			name:             "the promote unexpectedly succeeded — G-0393's guard did not fire",
			preScopeState:    "active",
			promoteStatus:    "ok",
			promoteErrorCode: "",
			epicStatusAfter:  "done",
			postScopeState:   "active",
			wantSubstrings: []string{
				"G-0393's guard did not fire",
				"refused for the wrong reason",
				`the epic's status changed to "done"`,
			},
		},
		{
			name:             "the promote refused, but for a different reason than G-0393's guard",
			preScopeState:    "active",
			promoteStatus:    "error",
			promoteErrorCode: wrongCode,
			epicStatusAfter:  "active",
			postScopeState:   "active",
			wantSubstrings:   []string{"refused for the wrong reason"},
		},
		{
			name:             "the epic's status changed despite the promote being refused",
			preScopeState:    "active",
			promoteStatus:    "error",
			promoteErrorCode: verb.CodeEpicPromoteNonTerminalChildren.ID,
			epicStatusAfter:  "done",
			postScopeState:   "active",
			wantSubstrings:   []string{`the epic's status changed to "done"`},
		},
		{
			name:             "the child's scope state changed despite the promote being refused",
			preScopeState:    "active",
			promoteStatus:    "error",
			promoteErrorCode: verb.CodeEpicPromoteNonTerminalChildren.ID,
			epicStatusAfter:  "active",
			postScopeState:   "ended",
			wantSubstrings:   []string{"scope state changed after the refused attempt"},
		},
		{
			name:             "every check fails at once",
			preScopeState:    "paused",
			promoteStatus:    "ok",
			promoteErrorCode: wrongCode,
			epicStatusAfter:  "done",
			postScopeState:   "ended",
			wantSubstrings: []string{
				"the scenario's premise did not hold",
				"G-0393's guard did not fire",
				"refused for the wrong reason",
				`the epic's status changed to "done"`,
				"scope state changed after the refused attempt",
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyArchiveDuringActiveScope(tc.preScopeState, tc.promoteStatus, tc.promoteErrorCode, tc.epicStatusAfter, tc.postScopeState)
			if len(got) != len(tc.wantSubstrings) {
				t.Fatalf("violations = %+v, want %d matching %v", got, len(tc.wantSubstrings), tc.wantSubstrings)
			}
			for _, want := range tc.wantSubstrings {
				found := false
				for _, v := range got {
					if strings.Contains(v.Message, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("no violation contained %q; got %+v", want, got)
				}
			}
		})
	}
}

// TestScopeState_ReturnsEmptyStringWhenNoScopesPresent pins scopeState's
// empty-scopes branch directly — a real "show" call before any
// authorize was ever opened returns no scopes array at all.
func TestScopeState_ReturnsEmptyStringWhenNoScopesPresent(t *testing.T) {
	t.Parallel()
	if got := scopeState(verbEnvelope{}); got != "" {
		t.Fatalf("scopeState on an empty envelope = %q, want empty string", got)
	}
}

// TestErrorCode_ReturnsEmptyStringWhenNoErrorPresent pins errorCode's
// nil-error branch directly — a status:"ok" envelope carries no error
// field at all.
func TestErrorCode_ReturnsEmptyStringWhenNoErrorPresent(t *testing.T) {
	t.Parallel()
	if got := errorCode(verbEnvelope{Status: "ok"}); got != "" {
		t.Fatalf("errorCode on an error-less envelope = %q, want empty string", got)
	}
}
