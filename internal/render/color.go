package render

import "os"

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

// ansiBoldOn and ansiBoldOff are the SGR sequences for bold attribute
// on (parameter 1) and reset all (parameter 0). Reset-all is used
// instead of "bold off" (22) because some terminals interpret 22 as
// "normal intensity" without clearing other attributes — reset-all is
// the safe path for code that only touches bold.
const (
	ansiBoldOn  = "\x1b[1m"
	ansiBoldOff = "\x1b[0m"
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
