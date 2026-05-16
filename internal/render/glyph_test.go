package render

import "testing"

func TestStatusGlyph(t *testing.T) {
	t.Parallel()
	tests := []struct {
		status string
		want   string
	}{
		// ✓ — finished
		{"done", "✓"},
		{"met", "✓"},
		{"addressed", "✓"},
		{"accepted", "✓"},
		// → — moving
		{"in_progress", "→"},
		{"active", "→"},
		// ○ — not started
		{"open", "○"},
		{"draft", "○"},
		{"proposed", "○"},
		// ✗ — closed off
		{"cancelled", "✗"},
		{"wontfix", "✗"},
		{"rejected", "✗"},
		{"retired", "✗"},
		{"superseded", "✗"},
		// unknown returns ""
		{"", ""},
		{"deferred", ""}, // AC status, not entity status — no glyph
		{"bogus", ""},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			t.Parallel()
			if got := StatusGlyph(tt.status); got != tt.want {
				t.Errorf("StatusGlyph(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}
