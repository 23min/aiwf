package render

import (
	"os"

	"github.com/23min/aiwf/internal/entity"
)

// ColorEnabled returns true when f is a terminal and the operator has
// not opted out of color via the NO_COLOR environment variable. Per
// https://no-color.org, NO_COLOR is honored when set to any non-empty
// value regardless of content; an empty NO_COLOR is treated as unset.
//
// Callers gate ANSI escape sequences on this predicate. Plain glyphs
// (✓ → ○ ✗) are content, not color, and stay enabled everywhere — they
// render in any UTF-8-capable consumer including CI logs and pipes.
//
// The TTY check rides on TerminalWidth's non-zero return, so the same
// "no styling under `go test`, pipes, and redirected output" guarantee
// applies — golden tests stay byte-identical to the un-styled rendering
// without any test-time override.
func ColorEnabled(f *os.File) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return TerminalWidth(f) > 0
}

// ansiBoldOn / ansiDimOn / ansi*Color* / ansiReset are the SGR
// sequences for the styling helpers below. Reset-all (parameter 0) is
// used to close every wrapper because some terminals interpret the
// targeted off codes (22 for bold/dim, 39 for default-fg) without
// clearing other attributes — reset-all is the safe path.
const (
	ansiBoldOn  = "\x1b[1m"
	ansiBoldOff = "\x1b[0m"

	ansiDimOn = "\x1b[2m"

	ansiFgGreen  = "\x1b[32m"
	ansiFgYellow = "\x1b[33m"
	ansiFgCyan   = "\x1b[36m"
	ansiFgRed    = "\x1b[31m"
	ansiResetAll = "\x1b[0m"
)

// Bold wraps s in the ANSI bold-on / reset-all escape sequence when
// enabled is true; otherwise returns s unchanged. Callers resolve
// `enabled` via ColorEnabled at the call site — Bold itself is a pure
// string operation with no IO side effects.
//
// An empty string returns "" unchanged: a bold-on/reset-all wrapper
// around no content is wasted bytes that some pagers render as an
// empty 1-char attribute region.
func Bold(s string, enabled bool) string {
	if !enabled || s == "" {
		return s
	}
	return ansiBoldOn + s + ansiBoldOff
}

// Dim wraps s in the ANSI dim-on / reset-all escape sequence when
// enabled is true; otherwise returns s unchanged. Used for secondary
// context lines (branch, age, etc.) so the eye lands on primary content
// first. G-0122.
func Dim(s string, enabled bool) string {
	if !enabled || s == "" {
		return s
	}
	return ansiDimOn + s + ansiResetAll
}

// StatusColor wraps s in the ANSI color appropriate for the entity /
// AC status when enabled is true; otherwise returns s unchanged. The
// color mapping mirrors StatusGlyph's grouping:
//
//   - green:  terminal positive (done, met, addressed, accepted, active)
//   - yellow: in flight (in_progress)
//   - cyan:   pending (open, draft, proposed)
//   - red:    terminal negative (cancelled, wontfix, rejected,
//     deprecated, retired, superseded)
//   - uncolored: status not in the closed-set vocabulary
//
// G-0122 color hierarchy.
func StatusColor(s, status string, enabled bool) string {
	if !enabled || s == "" {
		return s
	}
	var code string
	switch status {
	case entity.StatusDone, entity.StatusMet, entity.StatusAddressed,
		entity.StatusAccepted, entity.StatusActive:
		code = ansiFgGreen
	case entity.StatusInProgress:
		code = ansiFgYellow
	case entity.StatusOpen, entity.StatusDraft, entity.StatusProposed:
		code = ansiFgCyan
	case entity.StatusCancelled, entity.StatusWontfix, entity.StatusRejected,
		entity.StatusDeprecated, entity.StatusRetired, entity.StatusSuperseded:
		code = ansiFgRed
	default:
		return s
	}
	return code + s + ansiResetAll
}
