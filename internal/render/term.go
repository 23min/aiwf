package render

import (
	"os"
	"unicode/utf8"

	"golang.org/x/term"
)

// TerminalWidth returns the column width of f when f is a terminal, or
// 0 when it is not (piped, redirected to a file, run under `go test`,
// etc.). Callers gate truncation on the non-zero result so the same
// rendering code stays byte-identical in non-TTY contexts — golden
// tests, CI logs, and pipelines all see untruncated output.
//
// The "not a terminal" path is the silent default: any error from the
// underlying syscall (closed fd, unsupported platform, etc.) collapses
// to the same zero return. A separate IsTerminal predicate would let
// callers distinguish "no TTY" from "TTY but width-detection failed",
// but no current call site cares.
func TerminalWidth(f *os.File) int {
	if f == nil {
		return 0
	}
	// File descriptors fit in int on every platform x/term supports;
	// gosec's uintptr→int conversion warning is a false-positive in
	// this context (the term package's own API takes int by design).
	fd := int(f.Fd()) //nolint:gosec // see comment above
	if !term.IsTerminal(fd) {
		return 0
	}
	w, _, err := term.GetSize(fd)
	if err != nil || w <= 0 {
		return 0
	}
	return w
}

// Truncate returns s capped to maxRunes runes, replacing the tail with
// the single-rune ellipsis "…" when truncation occurred. When maxRunes
// is non-positive or s already fits, returns s unchanged. When maxRunes
// is 1, returns "…" (the ellipsis itself is the only output that fits).
//
// Operates on runes, not bytes, so multibyte characters in titles are
// handled correctly. Display-cell width is not modeled — every rune
// counts as one cell. The aiwf glyph palette (`✓ → ○ ✗`) is all
// 1-cell BMP, so this matches reality for current output; emoji or
// CJK would need go-runewidth (deferred per G-0080's out-of-scope).
func Truncate(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return s
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	if maxRunes == 1 {
		return "…"
	}
	// Keep maxRunes-1 runes, then append the ellipsis rune.
	runes := []rune(s)
	return string(runes[:maxRunes-1]) + "…"
}
