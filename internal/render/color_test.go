package render

import (
	"os"
	"testing"

	"github.com/23min/aiwf/internal/entity"
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

// TestStatusColor pins the G-0122 status-to-color mapping: each
// bucket in the switch takes its documented ANSI code, an unrecognized
// status passes the label through uncolored, and disabled/empty short-
// circuit before the switch runs at all. status is a plain string here
// (the boundary StatusColor's callers cross when a typed entity.Status
// is rendered) — StatusColor casts it back to entity.Status internally
// to compare against the typed constants.
func TestStatusColor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		label   string
		status  string
		enabled bool
		want    string
	}{
		{"disabled passes label through regardless of status", "dirty", string(entity.StatusInProgress), false, "dirty"},
		{"empty label short-circuits before the switch", "", string(entity.StatusDone), true, ""},
		{"done bucket is green", "[done]", string(entity.StatusDone), true, "\x1b[32m[done]\x1b[0m"},
		{"met bucket is green", "[met]", string(entity.StatusMet), true, "\x1b[32m[met]\x1b[0m"},
		{"in_progress bucket is yellow", "dirty", string(entity.StatusInProgress), true, "\x1b[33mdirty\x1b[0m"},
		{"open bucket is cyan", "[open]", string(entity.StatusOpen), true, "\x1b[36m[open]\x1b[0m"},
		{"cancelled bucket is red", "[cancelled]", string(entity.StatusCancelled), true, "\x1b[31m[cancelled]\x1b[0m"},
		{"unrecognized status is uncolored", "[weird]", "not-a-real-status", true, "[weird]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := StatusColor(tt.label, tt.status, tt.enabled); got != tt.want {
				t.Errorf("StatusColor(%q, %q, %v) = %q, want %q", tt.label, tt.status, tt.enabled, got, tt.want)
			}
		})
	}
}
