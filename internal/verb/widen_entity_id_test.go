package verb

import "testing"

// TestWidenEntityID pins the bare-id-only contract documented on
// widenEntityID. Composite ids, empty strings, unknown prefixes, and
// non-id shapes all return unchanged so callers can pass any string
// without pre-filtering.
func TestWidenEntityID(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want string
	}{
		// Bare ids: below-grammar narrow widens to canonical.
		{"epic narrow E-1", "E-1", "E-0001"},
		{"epic narrow E-22", "E-22", "E-0022"},
		{"milestone narrow M-77", "M-77", "M-0077"},
		{"gap narrow G-9", "G-9", "G-0009"},
		{"decision narrow D-3", "D-3", "D-0003"},
		{"contract narrow C-5", "C-5", "C-0005"},
		{"forward-compat F-1", "F-1", "F-0001"},
		{"ADR narrow ADR-1", "ADR-1", "ADR-0001"},

		// Bare ids: canonical-or-wider stays unchanged.
		{"canonical M-0001 unchanged", "M-0001", "M-0001"},
		{"wider M-12345 unchanged", "M-12345", "M-12345"},

		// Bare-only contract: composite ids pass through verbatim.
		// Callers split via entity.ParseCompositeID and recompose.
		{"composite M-77/AC-1 unchanged", "M-77/AC-1", "M-77/AC-1"},
		{"composite M-0001/AC-1 unchanged", "M-0001/AC-1", "M-0001/AC-1"},

		// Edge cases: empty, unknown prefix, non-id shape.
		{"empty string", "", ""},
		{"unknown prefix X-1", "X-1", "X-1"},
		{"non-id shape", "milestone", "milestone"},
		{"prefix only (no number)", "M-", "M-"},
		{"prefix + non-digit", "M-foo", "M-foo"},
		{"mixed alphanumeric", "M-1a", "M-1a"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := widenEntityID(tc.in); got != tc.want {
				t.Errorf("widenEntityID(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
