package verb

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/entity"
)

// TestTerminalStatusesForKind pins terminalStatusesForKind's per-kind
// terminal-status enumeration — the reverse lookup CancelAuditOnly uses
// to decide which predecessors could have reached the current terminal.
// The function is hardcoded (deliberately, to avoid reflecting the
// transitions map) and its own doc requires it stay in lock-step with
// entity/transition.go; this exhaustive table is that mechanical
// backstop. It also covers every kind arm exercised only via the
// audit-only cancel path for the non-epic/milestone kinds.
func TestTerminalStatusesForKind(t *testing.T) {
	t.Parallel()
	tests := []struct {
		kind entity.Kind
		want []entity.Status
	}{
		{entity.KindEpic, []entity.Status{entity.StatusDone, entity.StatusCancelled}},
		{entity.KindMilestone, []entity.Status{entity.StatusDone, entity.StatusCancelled}},
		{entity.KindADR, []entity.Status{entity.StatusSuperseded, entity.StatusRejected}},
		{entity.KindDecision, []entity.Status{entity.StatusSuperseded, entity.StatusRejected}},
		{entity.KindGap, []entity.Status{entity.StatusAddressed, entity.StatusWontfix}},
		{entity.KindContract, []entity.Status{entity.StatusRetired, entity.StatusRejected}},
		{entity.Kind("bogus"), nil},
	}
	for _, tc := range tests {
		t.Run(string(tc.kind), func(t *testing.T) {
			t.Parallel()
			if diff := cmp.Diff(tc.want, terminalStatusesForKind(tc.kind)); diff != "" {
				t.Errorf("terminalStatusesForKind(%q) mismatch (-want +got):\n%s", tc.kind, diff)
			}
		})
	}
}
