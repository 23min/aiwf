package config

import "testing"

// TestRefusedCoauthors pins the normalization the refusal depends on. Each
// case is a separate decision the getter makes about a configured entry, and
// each fails silently when it is wrong: a mis-normalized entry matches
// nothing, so the repo reads as protected while refusing nobody.
func TestRefusedCoauthors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cfg  *Config
		want map[string]bool
	}{
		{
			name: "a nil config refuses nothing",
			cfg:  nil,
			want: nil,
		},
		{
			name: "an absent provenance block refuses nothing",
			cfg:  &Config{},
			want: nil,
		},
		{
			name: "a configured address is lowercased for comparison",
			cfg:  &Config{Provenance: Provenance{RefuseCoauthors: []string{"NoReply@Example.Invalid"}}},
			want: map[string]bool{"noreply@example.invalid": true},
		},
		{
			name: "surrounding whitespace is trimmed",
			cfg:  &Config{Provenance: Provenance{RefuseCoauthors: []string{"  a@example.invalid\t"}}},
			want: map[string]bool{"a@example.invalid": true},
		},
		{
			// An empty entry must not enter the set: it would match every
			// trailer whose address parses empty, refusing commits the repo
			// never asked to refuse.
			name: "empty and whitespace-only entries are dropped",
			cfg:  &Config{Provenance: Provenance{RefuseCoauthors: []string{"", "   ", "a@example.invalid"}}},
			want: map[string]bool{"a@example.invalid": true},
		},
		{
			name: "a list of only empty entries refuses nothing",
			cfg:  &Config{Provenance: Provenance{RefuseCoauthors: []string{"", "  "}}},
			want: map[string]bool{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.cfg.RefusedCoauthors()
			if len(got) != len(tc.want) {
				t.Fatalf("RefusedCoauthors() = %v, want %v", got, tc.want)
			}
			for addr := range tc.want {
				if !got[addr] {
					t.Errorf("RefusedCoauthors() missing %q; got %v", addr, got)
				}
			}
			for addr := range got {
				if !tc.want[addr] {
					t.Errorf("RefusedCoauthors() has unexpected %q; got %v", addr, got)
				}
			}
		})
	}
}
