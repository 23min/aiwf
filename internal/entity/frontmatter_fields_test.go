package entity

import "testing"

// SameFrontmatterFields answers "did a field change", which is not the
// same question as "are these bytes equal". Callers refuse on the
// answer, so each case below is a decision someone's work depends on.
func TestSameFrontmatterFields(t *testing.T) {
	t.Parallel()

	const canonical = "---\nid: E-0001\ntitle: Foundations\nstatus: proposed\n---\n## Goal\n\nProse.\n"

	cases := []struct {
		name string
		a, b string
		want bool
	}{
		{
			// The case that makes byte-comparison wrong: Serialize emits a
			// canonical field order, so a file committed with hand-ordered
			// frontmatter differs byte-for-byte from its own re-serialization
			// while declaring the same state.
			name: "same fields in a different order are the same",
			a:    canonical,
			b:    "---\nstatus: proposed\ntitle: Foundations\nid: E-0001\n---\n## Goal\n\nProse.\n",
			want: true,
		},
		{
			name: "a differing body does not make the frontmatter differ",
			a:    canonical,
			b:    "---\nid: E-0001\ntitle: Foundations\nstatus: proposed\n---\n## Goal\n\nEntirely rewritten.\n",
			want: true,
		},
		{
			name: "a changed value differs",
			a:    canonical,
			b:    "---\nid: E-0001\ntitle: Foundations\nstatus: active\n---\n## Goal\n\nProse.\n",
			want: false,
		},
		{
			// An added key must register even though no Entity field claims
			// it — the reason the decode is into a generic mapping.
			name: "an added key differs",
			a:    canonical,
			b:    "---\nid: E-0001\ntitle: Foundations\nstatus: proposed\npriority: high\n---\n## Goal\n\nProse.\n",
			want: false,
		},
		{
			name: "a missing delimiter on the first side is not the same",
			a:    "no frontmatter here\n",
			b:    canonical,
			want: false,
		},
		{
			name: "a missing delimiter on the second side is not the same",
			a:    canonical,
			b:    "no frontmatter here\n",
			want: false,
		},
		{
			name: "unparseable YAML on the first side is not the same",
			a:    "---\nid: [unclosed\n---\nbody\n",
			b:    canonical,
			want: false,
		},
		{
			name: "unparseable YAML on the second side is not the same",
			a:    canonical,
			b:    "---\nid: [unclosed\n---\nbody\n",
			want: false,
		},
		{
			// Two unreadable sides are not "equally unreadable, therefore
			// equal": the callers refuse on false, and waving both through
			// would grant an exemption on a comparison that never happened.
			name: "two unparseable sides are still not the same",
			a:    "---\nid: [unclosed\n---\nbody\n",
			b:    "---\nid: [unclosed\n---\nbody\n",
			want: false,
		},
		{
			// A duplicate key has no single value to compare, so it must
			// not read as equal to a side declaring the key once.
			name: "a duplicate key is not the same as a single one",
			a:    "---\nid: E-0001\nid: E-0002\ntitle: Foundations\n---\nbody\n",
			b:    canonical,
			want: false,
		},
		{
			// Frontmatter that is not a mapping at all decodes without
			// error into a nil map, which would otherwise compare equal to
			// any other non-mapping side.
			name: "a top-level sequence is not the same as a mapping",
			a:    "---\n- id: E-0001\n- title: Foundations\n---\nbody\n",
			b:    canonical,
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := SameFrontmatterFields([]byte(tc.a), []byte(tc.b)); got != tc.want {
				t.Errorf("SameFrontmatterFields = %v, want %v", got, tc.want)
			}
		})
	}
}
