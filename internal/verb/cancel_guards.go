package verb

import (
	"fmt"
	"sort"
	"strings"

	"github.com/23min/aiwf/internal/codes"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// nonTerminalEpicChildren returns the sorted ids of epicID's child
// milestones that are not yet at a terminal status. Shared by three
// call sites that each refuse to leave a non-terminal milestone
// stranded under a terminal (or terminal-bound) parent epic: Cancel's
// epic-cancel guard (D-0003, below), Promote's epic-terminal guard
// (G-0393 / G-0394, promote.go — both `done` and `cancelled`), and
// Archive's independent subtree-terminality guard (G-0394, archive.go).
func nonTerminalEpicChildren(t *tree.Tree, epicID string) []string {
	var nonTerminal []string
	for _, m := range t.ByKind(entity.KindMilestone) {
		if m.Parent == epicID && !entity.IsTerminal(entity.KindMilestone, m.Status) {
			nonTerminal = append(nonTerminal, m.ID)
		}
	}
	sort.Strings(nonTerminal)
	return nonTerminal
}

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

// CodeEpicPromoteNonTerminalChildren is the typed kernel-code
// descriptor carried by [EpicPromoteNonTerminalChildrenError] when
// `aiwf promote` refuses to move an epic straight to a terminal status
// while it still owns one or more non-terminal child milestones
// (G-0393 / G-0394: two independently-filed gaps converging on the
// same guard). Mirrors CodeEpicCancelNonTerminalChildren's D-0003
// guard onto both of Promote's terminal targets for KindEpic — `done`
// and `cancelled` — so a done epic with an in-progress milestone is
// exactly as incoherent as a cancelled one, and `aiwf promote <epic>
// cancelled` can't bypass Cancel's own dedicated guard by going
// through Promote instead. Declares [codes.ClassLegality] (D-0011).
var CodeEpicPromoteNonTerminalChildren = codes.Code{ID: "epic-promote-non-terminal-children", Class: codes.ClassLegality}

// EpicPromoteNonTerminalChildrenError reports that `aiwf promote`
// refused to move an epic to a terminal status (NewStatus) because one
// or more of its child milestones are still non-terminal
// (refuse-with-listing, no auto-cascade, mirroring D-0003's cancel
// guard). The operator must dispose each listed milestone (cancel or
// done) before the epic can reach a terminal status by any path. Runs
// unconditionally, even under --force — matching Cancel's own D-0003
// guard, which has no force-bypass either: force relaxes FSM-
// transition legality, not this structural children precondition.
// Archive's independent subtree-terminality guard
// (internal/verb/archive.go) is the defense-in-depth backstop for the
// state a raw frontmatter hand-edit (bypassing the verb layer
// entirely) can still produce. It implements [entity.Coded], carrying
// CodeEpicPromoteNonTerminalChildren.
type EpicPromoteNonTerminalChildrenError struct {
	// Epic is the id of the epic whose promote was refused.
	Epic string
	// NewStatus is the terminal status the promote attempted to reach.
	NewStatus string
	// Children holds the sorted ids of the offending non-terminal child
	// milestones.
	Children []string
}

// Error implements error. It names the epic, the attempted terminal
// status, the count of offending milestones, the sorted ids, instructs
// the operator to dispose each first, and includes
// CodeEpicPromoteNonTerminalChildren.ID so message-matching consumers
// can recognize the refusal.
func (e *EpicPromoteNonTerminalChildrenError) Error() string {
	return fmt.Sprintf(
		"cannot promote %s to %s: %d non-terminal child milestone(s) [%s] (%s); cancel or done each before promoting the epic to a terminal status",
		e.Epic, e.NewStatus, len(e.Children), strings.Join(e.Children, ", "), CodeEpicPromoteNonTerminalChildren.ID,
	)
}

// Code returns CodeEpicPromoteNonTerminalChildren's ID, satisfying
// [entity.Coded].
func (e *EpicPromoteNonTerminalChildrenError) Code() string {
	return CodeEpicPromoteNonTerminalChildren.ID
}

// CodeMilestonePromoteNonTerminalACs is the typed kernel-code
// descriptor carried by [MilestonePromoteNonTerminalACsError] when
// `aiwf promote` refuses to move a milestone straight to `cancelled`
// while it still carries one or more `open` acceptance criteria
// (G-0335, mirroring G-0393 / G-0394's epic-level fix). Mirrors
// CodeMilestoneCancelNonTerminalACs's D-0004 guard onto Promote's own
// `cancelled` target for KindMilestone, so `aiwf promote <milestone>
// cancelled` can't bypass Cancel's own dedicated guard by going
// through Promote instead. Declares [codes.ClassLegality] (D-0011).
var CodeMilestonePromoteNonTerminalACs = codes.Code{ID: "milestone-promote-non-terminal-acs", Class: codes.ClassLegality}

// MilestonePromoteNonTerminalACsError reports that `aiwf promote`
// refused to move a milestone to `cancelled` (NewStatus) because one or
// more of its acceptance criteria are still `open`
// (refuse-with-listing, no auto-cascade, mirroring D-0004's cancel
// guard). The operator must dispose each listed AC (met, deferred, or
// cancelled) before the milestone can reach `cancelled` by any path.
// The `done` target carries its own, independent precondition — the
// milestone-done-incomplete-acs check-rule that projectionFindings runs
// further down Promote — so it never reaches this error type; NewStatus
// is carried for message symmetry with the sibling
// [EpicPromoteNonTerminalChildrenError], not because this type fires
// for more than one target today. It implements [entity.Coded],
// carrying CodeMilestonePromoteNonTerminalACs.
type MilestonePromoteNonTerminalACsError struct {
	// Milestone is the id of the milestone whose promote was refused.
	Milestone string
	// NewStatus is the terminal status the promote attempted to reach
	// (always "cancelled" today; see the type doc).
	NewStatus string
	// ACs holds the composite ids (`M-NNNN/AC-N`) of the offending open
	// acceptance criteria.
	ACs []string
}

// Error implements error. It names the milestone, the attempted
// target status, the count of offending ACs, the composite ids,
// instructs the operator to dispose each first, and includes
// CodeMilestonePromoteNonTerminalACs.ID so message-matching consumers
// can recognize the refusal.
func (e *MilestonePromoteNonTerminalACsError) Error() string {
	return fmt.Sprintf(
		"cannot promote %s to %s: %d open acceptance criterion(s) [%s] (%s); dispose each (met, deferred, or cancelled) before promoting the milestone to %s",
		e.Milestone, e.NewStatus, len(e.ACs), strings.Join(e.ACs, ", "), CodeMilestonePromoteNonTerminalACs.ID, e.NewStatus,
	)
}

// Code returns CodeMilestonePromoteNonTerminalACs's ID, satisfying
// [entity.Coded].
func (e *MilestonePromoteNonTerminalACsError) Code() string {
	return CodeMilestonePromoteNonTerminalACs.ID
}
