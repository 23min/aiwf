package main

import (
	"testing"
)

// TestPluralToEntityKind_UnknownReturnsFalse — branch-coverage for
// pluralToEntityKind's negative path. Closed-set switch with a
// default arm; the positive cases are exercised by the per-kind
// integration tests above. Without this test the default arm goes
// uncovered (see CLAUDE.md "Test untested code paths before
// declaring code paths 'done'").
func TestPluralToEntityKind_UnknownReturnsFalse(t *testing.T) {
	t.Parallel()
	cases := []string{
		"", "gap", "milestones", "tomatoes",
	}
	for _, plural := range cases {
		t.Run(plural, func(t *testing.T) {
			_, ok := pluralToEntityKind(plural)
			if ok {
				t.Errorf("pluralToEntityKind(%q) = (_, true), want (_, false) for unknown plural", plural)
			}
		})
	}
}

// TestParseSweepPending_MalformedMessageReturnsNil — branch-coverage
// for parseSweepPending's defensive guard. The rule's upstream
// contract guarantees the message starts with a digit-count, but
// the parser still defends against drift by returning nil on a
// message that doesn't match — the renderer's nil-check skips the
// section cleanly without panicking.
//
// Inputs cover: empty string, prose-without-leading-digit, leading
// zero (the rule itself returns nil at zero so this won't fire in
// practice but the parser still guards against it).
func TestParseSweepPending_MalformedMessageReturnsNil(t *testing.T) {
	t.Parallel()
	cases := []string{
		"",
		"no digit here",
		"0 terminal entities awaiting sweep",
	}
	for _, msg := range cases {
		t.Run(msg, func(t *testing.T) {
			got := parseSweepPending(msg)
			if got != nil {
				t.Errorf("parseSweepPending(%q) = %+v, want nil", msg, got)
			}
		})
	}
}

// TestRunResolverKindIndexData_UnknownKindReturnsNil — branch-
// coverage for the renderResolver.KindIndexData nil-return path.
// The renderer skips file emission when this returns nil, so the
// guard is the chokepoint that prevents a bad plural from writing
// a malformed page. Pinned as a unit test so coverage hits the
// branch even when the integration set doesn't reach unknown
// kinds.
func TestRunResolverKindIndexData_UnknownKindReturnsNil(t *testing.T) {
	t.Parallel()
	r := &renderResolver{}
	data, err := r.KindIndexData("widgets", false)
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	if data != nil {
		t.Errorf("data = %+v, want nil for unknown plural", data)
	}
}

// TestTitleForKindIndex_CapitalizesActive — branch-coverage for
// titleForKindIndex's title-case helper. Active page is "Gaps"
// (capitalized); all-set page is "All gaps" (lowercase, prefixed).
// Empty plural and pre-capitalized plural take the no-op branches.
func TestTitleForKindIndex_CapitalizesActive(t *testing.T) {
	t.Parallel()
	cases := []struct {
		plural          string
		includeArchived bool
		want            string
	}{
		{"gaps", false, "Gaps"},
		{"gaps", true, "All gaps"},
		{"decisions", false, "Decisions"},
		{"", false, ""},
		{"ADRs", false, "ADRs"}, // already capitalized — no-op branch
	}
	for _, c := range cases {
		t.Run(c.plural+"/archived="+itoaBool(c.includeArchived), func(t *testing.T) {
			got := titleForKindIndex(c.plural, c.includeArchived)
			if got != c.want {
				t.Errorf("titleForKindIndex(%q, %v) = %q, want %q", c.plural, c.includeArchived, got, c.want)
			}
		})
	}
}

func itoaBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
