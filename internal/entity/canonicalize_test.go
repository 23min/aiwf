package entity

import (
	"regexp"
	"testing"
)

// regexpCompile is a thin alias used inside tests so the regex-pattern
// assertions read clearly. Also pins go vet's regex-compile check on
// the test inputs.
func regexpCompile(pat string) (*regexp.Regexp, error) {
	return regexp.Compile(pat)
}

// TestCanonicalize covers AC-2's parser-tolerance contract: every
// recognizable id is left-padded to CanonicalPad on the lookup side,
// while ids already at or above canonical width pass through
// unchanged. Composite ids recurse on the parent; unrecognized
// inputs pass through verbatim.
//
// Inputs are sourced from internal/entity/entity.go::idPatterns and
// compositeIDPattern (the closed grammar for aiwf ids). The narrow
// inputs (`E-22`, `M-007`, etc.) are intentional: AC-2 is the
// parser-tolerance test, so narrow inputs by design.
func TestCanonicalize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		// Bare ids — narrow → canonical. The grammar's per-kind floor
		// (E-\d{2,}, M-\d{3,}, etc., per internal/entity/entity.go::idPatterns)
		// is the input space; ids below the floor pass through verbatim
		// — Canonicalize tolerates legacy widths but does not invent
		// well-formed ids from non-conforming input.
		{"epic-narrow", "E-22", "E-0022"},
		{"epic-below-floor-passthrough", "E-1", "E-1"},
		{"milestone-narrow", "M-007", "M-0007"},
		{"milestone-below-floor-passthrough", "M-22", "M-22"},
		{"adr-already-canonical", "ADR-0001", "ADR-0001"},
		{"gap-narrow", "G-093", "G-0093"},
		{"decision-narrow", "D-005", "D-0005"},
		{"contract-narrow", "C-009", "C-0009"},
		// Already-canonical or wider — pass through unchanged.
		{"epic-canonical", "E-0023", "E-0023"},
		{"milestone-canonical", "M-0007", "M-0007"},
		{"epic-wider-than-canonical", "E-12345", "E-12345"},
		{"milestone-wider-than-canonical", "M-99999", "M-99999"},
		// Composite ids — recurse on parent, leave AC-N alone. Parent
		// must satisfy the milestone-id floor (\d{3,}); below-floor
		// inputs pass through verbatim.
		{"composite-narrow-parent", "M-007/AC-1", "M-0007/AC-1"},
		{"composite-narrow-parent-wide-ac", "M-007/AC-12", "M-0007/AC-12"},
		{"composite-canonical-parent", "M-0007/AC-1", "M-0007/AC-1"},
		{"composite-below-floor-passthrough", "M-22/AC-1", "M-22/AC-1"},
		// Unrecognized / pass-through cases.
		{"empty", "", ""},
		{"non-id", "hello", "hello"},
		{"prefix-but-no-digits", "E-", "E-"},
		{"prefix-but-non-numeric", "E-abc", "E-abc"},
		{"prefix-with-trailing-junk", "E-22-foo", "E-22-foo"},
		{"composite-malformed-no-ac", "M-22/", "M-22/"},
		{"composite-malformed-bare-ac", "AC-1", "AC-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Canonicalize(tt.in)
			if got != tt.want {
				t.Errorf("Canonicalize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestIDGrepAlternation_MatchesBothWidths exercises the
// regex-alternation helper used by `git log --grep` callers
// (admin_cmd.go's history reader, scopes.go's authorize-commit
// reader). The pattern must compile cleanly under POSIX-extended
// regex semantics and match both narrow and canonical-width
// renderings of the input id.
func TestIDGrepAlternation_MatchesBothWidths(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		matches    []string // strings the pattern must match
		nonMatches []string // strings the pattern must not match
	}{
		{
			"epic-22",
			"E-22",
			[]string{"E-22", "E-0022", "E-022"},
			[]string{"E-220", "E-2", "E-23", "E-0023"},
		},
		{
			"epic-canonical",
			"E-0022",
			[]string{"E-22", "E-0022"},
			[]string{"E-220", "E-23"},
		},
		{
			"milestone-narrow",
			"M-007",
			[]string{"M-007", "M-0007", "M-7"}, // M-7 below grammar floor but matches numerically
			[]string{"M-070", "M-008"},
		},
		{
			"composite-narrow",
			"M-007/AC-1",
			[]string{"M-007/AC-1", "M-0007/AC-1"},
			[]string{"M-007/AC-10", "M-070/AC-1"},
		},
		{
			"adr-canonical",
			"ADR-0001",
			[]string{"ADR-0001", "ADR-1"},
			[]string{"ADR-0010", "ADR-0002"},
		},
		{
			// All-zeros input: trimmed numeric becomes empty, so the
			// helper falls back to "0" so the pattern is still valid.
			"epic-all-zeros",
			"E-0000",
			[]string{"E-0000", "E-00"},
			[]string{"E-0001"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pat := IDGrepAlternation(tt.id)
			// Anchor end-to-end so the test mirrors how the grep is used:
			// `^aiwf-entity: <pat>$` against trailer lines.
			re, err := regexpCompile("^" + pat + "$")
			if err != nil {
				t.Fatalf("compile %q: %v", pat, err)
			}
			for _, m := range tt.matches {
				if !re.MatchString(m) {
					t.Errorf("pattern %q does not match %q (want match)", pat, m)
				}
			}
			for _, n := range tt.nonMatches {
				if re.MatchString(n) {
					t.Errorf("pattern %q matches %q (want no match)", pat, n)
				}
			}
		})
	}
}

// TestIDGrepAlternation_EdgeCases covers the helper's defensive paths
// (empty input, prefix-only input, non-grammar input). These are
// branches the matchers above don't traverse; the seam-test rule
// from CLAUDE.md asks for explicit coverage, not just integration.
func TestIDGrepAlternation_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		// Prefix-only string passes through unchanged (regex-quoted).
		{"prefix-only", "E-", `E-`},
		// Below-grammar-floor inputs (E-1) pass through (regex-quoted).
		{"below-floor", "E-1", `E-1`},
		// Unrecognized input passes through verbatim (regex-quoted).
		{"non-id", "hello", "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IDGrepAlternation(tt.in); got != tt.want {
				t.Errorf("IDGrepAlternation(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestIsCompositeID_TolerantOfBothWidths confirms that the composite-id
// grammar accepts both narrow and canonical parent widths. Inputs come
// from internal/entity/entity.go::compositeIDPattern, which uses
// `\d{3,}` so ≥3 digits is the floor.
func TestIsCompositeID_TolerantOfBothWidths(t *testing.T) {
	tests := []string{"M-22/AC-1", "M-007/AC-1", "M-0007/AC-1", "M-12345/AC-3"}
	for _, in := range tests {
		t.Run(in, func(t *testing.T) {
			// compositeIDPattern requires `M-\d{3,}` so M-22 is still
			// rejected by the grammar — Canonicalize is the right
			// chokepoint for that case, not IsCompositeID.
			if in == "M-22/AC-1" {
				if IsCompositeID(in) {
					t.Errorf("IsCompositeID(%q) = true, want false (grammar floor is 3 digits)", in)
				}
				return
			}
			if !IsCompositeID(in) {
				t.Errorf("IsCompositeID(%q) = false, want true", in)
			}
			parent, sub, ok := ParseCompositeID(in)
			if !ok {
				t.Errorf("ParseCompositeID(%q) ok=false", in)
			}
			if parent == "" || sub == "" {
				t.Errorf("ParseCompositeID(%q) = (%q, %q), want non-empty pair", in, parent, sub)
			}
		})
	}
}

// TestAllocateID_CanonicalFourDigitForEveryKind asserts the
// AC-1 contract from M-081: AllocateID emits a 4-digit zero-padded
// number for every entity kind, regardless of the previous high-water
// mark width on disk. Per ADR-0008, the canonical pad is uniform 4
// across all kinds; this is the load-bearing assertion that the
// allocator never re-emits narrow widths once M-A ships.
func TestAllocateID_CanonicalFourDigitForEveryKind(t *testing.T) {
	tests := []struct {
		kind Kind
		want string
	}{
		{KindEpic, "E-0001"},
		{KindMilestone, "M-0001"},
		{KindADR, "ADR-0001"},
		{KindGap, "G-0001"},
		{KindDecision, "D-0001"},
		{KindContract, "C-0001"},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			got := AllocateID(tt.kind, nil, nil)
			if got != tt.want {
				t.Errorf("AllocateID(%s, empty, empty) = %q, want %q", tt.kind, got, tt.want)
			}
		})
	}
}

// TestAllocateID_CanonicalAfterNarrowHighWater asserts that even
// when the working tree's existing entities use narrow-width ids
// (the typical post-migration legacy state), the allocator's next
// allocation lands at the canonical 4-digit width.
func TestAllocateID_CanonicalAfterNarrowHighWater(t *testing.T) {
	tests := []struct {
		kind  Kind
		prior string
		want  string
	}{
		{KindEpic, "E-22", "E-0023"},
		{KindMilestone, "M-007", "M-0008"},
		{KindADR, "ADR-0001", "ADR-0002"},
		{KindGap, "G-093", "G-0094"},
		{KindDecision, "D-005", "D-0006"},
		{KindContract, "C-009", "C-0010"},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			entities := []*Entity{{ID: tt.prior, Kind: tt.kind}}
			got := AllocateID(tt.kind, entities, nil)
			if got != tt.want {
				t.Errorf("AllocateID(%s, prior=%s) = %q, want %q", tt.kind, tt.prior, got, tt.want)
			}
		})
	}
}
