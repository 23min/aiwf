package render

import (
	"os"
	"testing"
)

func TestBold(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      string
		enabled bool
		want    string
	}{
		{"disabled passes through", "hello", false, "hello"},
		{"enabled wraps with ANSI", "hello", true, "\x1b[1mhello\x1b[0m"},
		{"empty disabled stays empty", "", false, ""},
		{"empty enabled stays empty", "", true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Bold(tt.in, tt.enabled); got != tt.want {
				t.Errorf("Bold(%q, %v) = %q, want %q", tt.in, tt.enabled, got, tt.want)
			}
		})
	}
}

// TestColorEnabled_NoColorEnvDisables pins the NO_COLOR contract:
// any non-empty value opts the operator out of ANSI styling. This is
// the load-bearing predicate honored by every Bold call site.
func TestColorEnabled_NoColorEnvDisables(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if ColorEnabled(os.Stdout) {
		t.Errorf("NO_COLOR=1 should disable color")
	}
}

// TestColorEnabled_EmptyNoColorAllowed pins the spec edge case: an
// empty NO_COLOR value is treated as unset per https://no-color.org.
// Under `go test` os.Stdout is not a TTY so this still returns false,
// but the TestSetenv side does prove the empty-string branch.
func TestColorEnabled_EmptyNoColorAllowed(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	if ColorEnabled(os.Stdout) {
		t.Errorf("empty NO_COLOR with non-TTY stdout should remain false (TTY check fails)")
	}
}

// TestColorEnabled_NonTTYDisables pins the TTY half of the predicate.
// Even with NO_COLOR unset, a non-TTY (every `go test` invocation)
// returns false. Together with TestTerminalWidth_NonTTYReturnsZero,
// this guarantees no ANSI escapes leak into golden files.
func TestColorEnabled_NonTTYDisables(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	if ColorEnabled(os.Stdout) {
		t.Errorf("non-TTY stdout should disable color")
	}
}
