package entity

import (
	"errors"
	"fmt"
	"testing"
)

// stubCoded is a minimal [Coded] implementor used to exercise [Code]'s
// errors.As extraction without depending on AC-2's FSMTransitionError.
type stubCoded struct{ code string }

func (s stubCoded) Error() string { return "stub error: " + s.code }
func (s stubCoded) Code() string  { return s.code }

func TestCodedError_ErrorsAs(t *testing.T) {
	t.Parallel()

	const want = "fsm-transition-illegal"

	tests := []struct {
		name     string
		err      error
		wantCode string
		wantOK   bool
	}{
		{
			name:     "direct coded error",
			err:      stubCoded{code: want},
			wantCode: want,
			wantOK:   true,
		},
		{
			name:     "wrapped one level via %w",
			err:      fmt.Errorf("context: %w", stubCoded{code: want}),
			wantCode: want,
			wantOK:   true,
		},
		{
			name: "wrapped three levels deep",
			err: fmt.Errorf("outer: %w",
				fmt.Errorf("middle: %w",
					fmt.Errorf("inner: %w", stubCoded{code: want}))),
			wantCode: want,
			wantOK:   true,
		},
		{
			name:     "plain error is not coded",
			err:      errors.New("just a message"),
			wantCode: "",
			wantOK:   false,
		},
		{
			// Anti-cheat: a plain error whose *text* contains a code-like
			// string must NOT resolve. Code extracts structurally via
			// errors.As; it does not scan the message.
			name:     "plain error mentioning a code in its text is not coded",
			err:      errors.New("failed: fsm-transition-illegal happened"),
			wantCode: "",
			wantOK:   false,
		},
		{
			name:     "nil error",
			err:      nil,
			wantCode: "",
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotCode, gotOK := Code(tt.err)
			if gotOK != tt.wantOK {
				t.Errorf("Code() ok = %v, want %v", gotOK, tt.wantOK)
			}
			if gotCode != tt.wantCode {
				t.Errorf("Code() code = %q, want %q", gotCode, tt.wantCode)
			}
		})
	}
}

// TestCode_EmptyCodeStillFound pins the bool's purpose: a Coded error
// whose Code() returns "" is still "found" (ok=true). This is the
// branch a naive `code != ""` collapse would get wrong, and it's why
// Code returns (string, bool) rather than just a string.
func TestCode_EmptyCodeStillFound(t *testing.T) {
	t.Parallel()
	code, ok := Code(stubCoded{code: ""})
	if !ok {
		t.Error("Code(stubCoded{\"\"}) ok = false, want true (an empty code is still a code)")
	}
	if code != "" {
		t.Errorf("Code(stubCoded{\"\"}) code = %q, want empty", code)
	}
}
