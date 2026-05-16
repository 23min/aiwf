package render

import (
	"os"
	"testing"
)

func TestTruncate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"empty no-op", "", 10, ""},
		{"shorter than max", "hello", 10, "hello"},
		{"equal to max", "hello", 5, "hello"},
		{"one over max", "hello!", 5, "hell…"},
		{"max zero", "hello", 0, "hello"},
		{"max negative", "hello", -3, "hello"},
		{"max one", "hello", 1, "…"},
		{"max two", "hello", 2, "h…"},
		{"multibyte rune kept whole", "café", 4, "café"},
		{"multibyte truncated by rune count", "caféééé", 4, "caf…"},
		{"glyph plus title", "→ Roll out TestMain", 10, "→ Roll ou…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Truncate(tt.in, tt.max); got != tt.want {
				t.Fatalf("Truncate(%q, %d) = %q, want %q", tt.in, tt.max, got, tt.want)
			}
		})
	}
}

// TestTerminalWidth_NonTTYReturnsZero pins the non-TTY contract: when
// stdin/stdout is piped (which is always the case under `go test`),
// width must be 0. Callers depend on this so truncation stays off in
// test, CI, and pipeline contexts — the bedrock that keeps golden
// tests stable.
func TestTerminalWidth_NonTTYReturnsZero(t *testing.T) {
	t.Parallel()
	// os.Stdout under `go test` is not a terminal.
	if got := TerminalWidth(os.Stdout); got != 0 {
		t.Fatalf("TerminalWidth(non-TTY) = %d, want 0", got)
	}
}

// TestTerminalWidth_NilReturnsZero pins the nil-file safety: callers
// pass os.Stdout or a *os.File they've opened; a nil here should not
// panic, just collapse to "no TTY".
func TestTerminalWidth_NilReturnsZero(t *testing.T) {
	t.Parallel()
	if got := TerminalWidth(nil); got != 0 {
		t.Fatalf("TerminalWidth(nil) = %d, want 0", got)
	}
}
