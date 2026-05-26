package verb

import (
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/codes"
)

// CodeEpicCancelNonTerminalChildren is the typed kernel-code descriptor
// carried by [EpicCancelNonTerminalChildrenError] when `aiwf cancel`
// refuses an epic that still owns one or more non-terminal child
// milestones (D-0003). It declares [codes.ClassLegality], the marker the
// closed legality set is enumerated from (D-0011). Consumers see its
// [codes.Code.ID] string via [EpicCancelNonTerminalChildrenError.Code]
// and in the message text.
var CodeEpicCancelNonTerminalChildren = codes.Code{ID: "epic-cancel-non-terminal-children", Class: codes.ClassLegality}

// CodeMilestoneCancelNonTerminalACs is the typed kernel-code descriptor
// carried by [MilestoneCancelNonTerminalACsError] when `aiwf cancel`
// refuses a milestone that still carries one or more `open` acceptance
// criteria (D-0004). It declares [codes.ClassLegality] (D-0011).
// Consumers see its [codes.Code.ID] string via
// [MilestoneCancelNonTerminalACsError.Code] and in the message text.
var CodeMilestoneCancelNonTerminalACs = codes.Code{ID: "milestone-cancel-non-terminal-acs", Class: codes.ClassLegality}

// EpicCancelNonTerminalChildrenError reports that `aiwf cancel` refused
// an epic because one or more of its child milestones are still
// non-terminal (D-0003: refuse-with-listing, no auto-cascade). The
// operator must dispose each listed milestone (cancel or done) before
// the epic can be cancelled. It implements [entity.Coded], carrying
// CodeEpicCancelNonTerminalChildren.
type EpicCancelNonTerminalChildrenError struct {
	// Epic is the id of the epic whose cancel was refused.
	Epic string
	// Children holds the sorted ids of the offending non-terminal child
	// milestones.
	Children []string
}

// Error implements error. It names the epic, the count of offending
// milestones, the sorted ids, instructs the operator to dispose each
// first, and includes CodeEpicCancelNonTerminalChildren.ID so
// message-matching consumers can recognize the refusal.
func (e *EpicCancelNonTerminalChildrenError) Error() string {
	return fmt.Sprintf(
		"cannot cancel %s: %d non-terminal child milestone(s) [%s] (%s); cancel or done each before cancelling the epic",
		e.Epic, len(e.Children), strings.Join(e.Children, ", "), CodeEpicCancelNonTerminalChildren.ID,
	)
}

// Code returns CodeEpicCancelNonTerminalChildren's ID, satisfying
// [entity.Coded].
func (e *EpicCancelNonTerminalChildrenError) Code() string {
	return CodeEpicCancelNonTerminalChildren.ID
}

// MilestoneCancelNonTerminalACsError reports that `aiwf cancel` refused
// a milestone because one or more of its acceptance criteria are still
// `open` (D-0004: refuse-with-listing, no auto-cascade). The operator
// must dispose each listed AC (met, deferred, or cancelled) before the
// milestone can be cancelled. It implements [entity.Coded], carrying
// CodeMilestoneCancelNonTerminalACs.
type MilestoneCancelNonTerminalACsError struct {
	// Milestone is the id of the milestone whose cancel was refused.
	Milestone string
	// ACs holds the composite ids (`M-NNNN/AC-N`) of the offending open
	// acceptance criteria.
	ACs []string
}

// Error implements error. It names the milestone, the count of offending
// ACs, the composite ids, instructs the operator to dispose each first,
// and includes CodeMilestoneCancelNonTerminalACs.ID so message-matching
// consumers can recognize the refusal.
func (e *MilestoneCancelNonTerminalACsError) Error() string {
	return fmt.Sprintf(
		"cannot cancel %s: %d open acceptance criterion(s) [%s] (%s); dispose each (met, deferred, or cancelled) before cancelling the milestone",
		e.Milestone, len(e.ACs), strings.Join(e.ACs, ", "), CodeMilestoneCancelNonTerminalACs.ID,
	)
}

// Code returns CodeMilestoneCancelNonTerminalACs's ID, satisfying
// [entity.Coded].
func (e *MilestoneCancelNonTerminalACsError) Code() string {
	return CodeMilestoneCancelNonTerminalACs.ID
}
