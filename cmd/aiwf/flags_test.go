package main

import (
	"reflect"
	"testing"
)

func TestReorderFlagsFirst(t *testing.T) {
	known := []string{"actor", "root", "reason"}
	bools := []string{"force"}
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "flag at end with separate value",
			in:   []string{"M-001", "--reason", "scope folded"},
			want: []string{"--reason", "scope folded", "M-001"},
		},
		{
			name: "flag at end with =value",
			in:   []string{"M-001", "--reason=scope folded"},
			want: []string{"--reason=scope folded", "M-001"},
		},
		{
			name: "two flags after positional, mixed forms",
			in:   []string{"M-001", "--actor", "human/peter", "--reason=note"},
			want: []string{"--actor", "human/peter", "--reason=note", "M-001"},
		},
		{
			name: "flags already first",
			in:   []string{"--reason", "note", "M-001"},
			want: []string{"--reason", "note", "M-001"},
		},
		{
			name: "two positionals plus flag",
			in:   []string{"E-01", "active", "--reason", "ready"},
			want: []string{"--reason", "ready", "E-01", "active"},
		},
		{
			name: "unknown flag falls through to positional position",
			in:   []string{"M-001", "--unknown", "foo"},
			want: []string{"M-001", "--unknown", "foo"},
		},
		{
			name: "no flags",
			in:   []string{"M-001", "active"},
			want: []string{"M-001", "active"},
		},
		{
			name: "empty",
			in:   nil,
			want: nil,
		},
		// Bool flags do not consume a following token. The cases
		// below are the load-bearing fix that allowed --force to be
		// safely interleaved with --reason on transition verbs.
		{
			name: "bool flag alone after positional",
			in:   []string{"E-01", "done", "--force"},
			want: []string{"--force", "E-01", "done"},
		},
		{
			name: "bool flag preceding value flag — must not eat --reason",
			in:   []string{"E-01", "done", "--force", "--reason", "got urgent"},
			want: []string{"--force", "--reason", "got urgent", "E-01", "done"},
		},
		{
			name: "value flag preceding bool flag",
			in:   []string{"M-001", "--reason", "now", "--force"},
			want: []string{"--reason", "now", "--force", "M-001"},
		},
		{
			name: "bool flag with =true form",
			in:   []string{"M-001", "--force=true"},
			want: []string{"--force=true", "M-001"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reorderFlagsFirst(tt.in, known, bools)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
