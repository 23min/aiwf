package stresstest

import (
	"fmt"
	"strings"
	"testing"
)

// concurrent_milestone_race_patch_test.go — the anchor-patching helper
// the regression probe in concurrent_milestone_race_regression_test.go
// builds its patched source copy with, and its own test. Both live
// here, untagged, because the helper is a pure string function and its
// test decides an outcome from fabricated input: no subprocess, no
// clock, no goroutine. An untagged file compiles into the `stress`
// build too, so the tagged probe still reaches the helper.

// patchExactlyOnce replaces old with newText in content, refusing
// unless old appears exactly once. The regression probe uses it to
// build a deliberately broken copy of this module's source, so an
// anchor that has drifted must fail loudly rather than silently
// patching the wrong spot — or nothing at all, which would leave the
// probe testing an unpatched binary and reporting success.
func patchExactlyOnce(content, old, newText string) (string, error) {
	switch n := strings.Count(content, old); n {
	case 0:
		return "", fmt.Errorf("patch anchor not found (want exactly 1 occurrence, got 0): %q", old)
	case 1:
		return strings.Replace(content, old, newText, 1), nil
	default:
		return "", fmt.Errorf("patch anchor is ambiguous (want exactly 1 occurrence, got %d): %q", n, old)
	}
}

// TestPatchExactlyOnce pins patchExactlyOnce's three outcomes — the
// anchor missing entirely, matching exactly once, and matching more
// than once — proving the sanity check fails loudly rather than
// silently patching the wrong spot (or nothing at all).
func TestPatchExactlyOnce(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		old     string
		newText string
		want    string
		wantErr bool
	}{
		{
			name:    "anchor missing entirely errors",
			content: "alpha\nbeta\ngamma\n",
			old:     "delta",
			newText: "epsilon",
			wantErr: true,
		},
		{
			name:    "anchor matching exactly once replaces cleanly",
			content: "alpha\nbeta\ngamma\n",
			old:     "beta",
			newText: "BETA",
			want:    "alpha\nBETA\ngamma\n",
			wantErr: false,
		},
		{
			name:    "anchor matching more than once errors, leaving content untouched",
			content: "alpha\nbeta\nalpha\n",
			old:     "alpha",
			newText: "ALPHA",
			wantErr: true,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := patchExactlyOnce(tc.content, tc.old, tc.newText)
			if (err != nil) != tc.wantErr {
				t.Fatalf("patchExactlyOnce error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("patchExactlyOnce = %q, want %q", got, tc.want)
			}
		})
	}
}
