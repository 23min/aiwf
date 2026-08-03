package policies

import (
	"strings"
	"testing"
)

// d0054_fixed_and_pinned_disposition_test.go — structural pins for the
// review disposition D-0054 adds: a defect already fixed and pinned by
// the check landing with it needs no further record.
//
// Every other disposition in the review classification yields an
// artifact — a check, a gap, a decision entity, or a line of reviewer
// prose — and the classification closes with "a finding that leaves as
// none of these is the leak", so an unnamed exit reads as no exit. This
// is the one that produces nothing further, and it is the counterweight
// the ritual surface otherwise lacks.
//
// Every surface in the table below states the classification, and a rule
// present in some and absent from others is worse than absent everywhere:
// a reader who meets the enumeration without this exit is told it does
// not exist. The reviewer role card matters most, being what a dispatched
// reviewer actually loads.
//
// Surfaces that mention the dispositions only conditionally — routing the
// unpinnable defect to a record — stay out of the table: they are already
// true of a pinned defect and need no edit.
//
// Each case is section-scoped per CLAUDE.md *Testing* §"Substring
// assertions are not structural assertions": the disposition has to live
// inside the section that carries the verdict handling, not merely appear
// somewhere in the file. Fixture paths and the skill-body reader are the
// package's existing ones, reused rather than redeclared.

// aiwfxReviewerCardPath is the reviewer role card, the fourth surface
// carrying the disposition enumeration. Materialized into consumers'
// `.claude/agents/` by `aiwf init` / `aiwf update`.
const aiwfxReviewerCardPath = "internal/skills/embedded-rituals/plugins/aiwf-extensions/agents/reviewer.md"

// TestD0054_FixedAndPinnedDispositionAcrossSurfaces pins the exit on
// every surface that enumerates the review dispositions.
func TestD0054_FixedAndPinnedDispositionAcrossSurfaces(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		fixture string
		// heading is a substring of the enclosing heading. The "## "
		// prefix, where present, excludes deeper headings carrying the
		// same word: wf-review-code has a checklist step "### 4.
		// Constraints (project-stated invariants)", whose "### 4. " breaks
		// the "## Constraints" substring, leaving only the top-level
		// section to match.
		heading string
		wants   []string
	}{
		{
			name:    "wrap-milestone review step",
			fixture: aiwfxWrapMilestoneFixturePath,
			heading: "Independent two-lens review",
			wants: []string{
				"pinned by the check landing with it takes no gap",
				"records an event rather than an obligation",
			},
		},
		{
			name:    "review-code verdict",
			fixture: wfReviewCodeFixturePath,
			heading: "Verdict",
			wants: []string{
				"already fixed and pinned needs no further record",
				"records an event rather than an obligation",
			},
		},
		{
			name:    "review-code constraints restatement",
			fixture: wfReviewCodeFixturePath,
			heading: "## Constraints",
			wants:   []string{"needs no further record, since the check is the record"},
		},
		{
			name:    "reviewer role card constraints",
			fixture: aiwfxReviewerCardPath,
			heading: "## Constraints",
			wants:   []string{"needs no further record, since the check is the record"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			section := sectionUnder(readVerbSkill(t, tc.fixture), tc.heading)
			if section == "" {
				t.Fatalf("%s: no section under a heading containing %q — the heading it is scoped to has moved", tc.fixture, tc.heading)
			}
			for _, want := range tc.wants {
				if !strings.Contains(section, want) {
					t.Errorf("%s: section %q does not carry the fixed-and-pinned disposition: missing %q", tc.fixture, tc.heading, want)
				}
			}
		})
	}
}
