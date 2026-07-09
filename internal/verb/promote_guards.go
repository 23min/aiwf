package verb

import (
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/codes"
)

// CodeEpicPromoteNonTerminalChildren is the typed kernel-code
// descriptor carried by [EpicPromoteNonTerminalChildrenError] when
// `aiwf promote` refuses to move an epic straight to a terminal status
// while it still owns one or more non-terminal child milestones
// (G-0393). It mirrors CodeEpicCancelNonTerminalChildren's own
// refuse-with-listing shape (D-0003) so the two entry points to the
// same invalid state — `aiwf cancel <epic>` and `aiwf promote <epic>
// done`/`cancelled` — are guarded symmetrically, rather than one
// producing a clean refusal and the other silently producing a
// non-terminal milestone under an archived, terminal epic that only
// `aiwf check`'s archived-entity-not-terminal rule catches after the
// fact. It declares [codes.ClassLegality] (D-0011).
var CodeEpicPromoteNonTerminalChildren = codes.Code{ID: "epic-promote-non-terminal-children", Class: codes.ClassLegality}

// EpicPromoteNonTerminalChildrenError reports that `aiwf promote`
// refused to move an epic to a terminal status (NewStatus) because one
// or more of its child milestones are still non-terminal. The operator
// must dispose each listed milestone (cancel or done) before the epic
// can reach a terminal status by any path. It implements
// [entity.Coded], carrying CodeEpicPromoteNonTerminalChildren.
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
