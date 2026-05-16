package main

import "testing"

func TestStripTrailers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "trailers only",
			in:   "aiwf-verb: cancel\naiwf-entity: M-002\naiwf-actor: human/peter",
			want: "",
		},
		{
			name: "body then trailers",
			in:   "scope folded into M-001\n\naiwf-verb: cancel\naiwf-entity: M-002\naiwf-actor: human/peter",
			want: "scope folded into M-001",
		},
		{
			name: "multi-paragraph body",
			in:   "first paragraph\n\nsecond paragraph\n\naiwf-verb: cancel\naiwf-entity: M-002",
			want: "first paragraph\n\nsecond paragraph",
		},
		{
			name: "no trailers, body only",
			in:   "just a body, nothing else",
			want: "just a body, nothing else",
		},
		{
			name: "empty input",
			in:   "",
			want: "",
		},
		{
			name: "Co-Authored-By trailer too",
			in:   "scope folded into M-001\n\naiwf-verb: cancel\naiwf-entity: M-002\nCo-Authored-By: Claude <noreply@anthropic.com>",
			want: "scope folded into M-001",
		},
		{
			name: "body with colon-containing prose, then trailers",
			// The body line "decided: ..." looks shaped like a trailer
			// (alphanumeric key + colon + value), but it sits in the
			// middle of the body, not at the end. The blank-line guard
			// preserves it as prose: only a contiguous trailer block at
			// the end, preceded by a blank line, gets stripped.
			in:   "decided: the legal review window is 30 days\n\naiwf-verb: cancel",
			want: "decided: the legal review window is 30 days",
		},
		{
			name: "trailer-shaped line in body without trailing trailer block",
			// No trailer block at all → return the body untouched.
			in:   "decided: 30 days",
			want: "decided: 30 days",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := stripTrailers(tt.in)
			if got != tt.want {
				t.Errorf("stripTrailers(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
