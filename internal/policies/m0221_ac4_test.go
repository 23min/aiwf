package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestM0221_AC4_ValidationRecordsRenderMeasurement asserts M-0221/AC-4:
// the milestone's `## Validation` section records a before/after render
// wall-time measurement taken via performance.md's recipe, naming BOTH
// mechanisms measured (the old per-entity fan-out and the new single
// HEAD pass) and carrying actual wall-time values. The absolute numbers
// are environment-specific and not gated; this pins that the record
// exists and is populated, not a threshold.
//
// Section-scoped per CLAUDE.md's "substring assertions are not structural
// assertions" rule: the entity is resolved through the loader (never a
// hardcoded work/ path), and every assertion is bounded to the
// `## Validation` section via extractMarkdownSection.
func TestM0221_AC4_ValidationRecordsRenderMeasurement(t *testing.T) {
	t.Parallel()
	root, tr := sharedRepoTree(t)
	e := tr.ByID("M-0221")
	if e == nil {
		t.Fatal("AC-4: M-0221 not found in tree")
	}
	data, err := os.ReadFile(filepath.Join(root, e.Path))
	if err != nil {
		t.Fatalf("reading M-0221 at %s: %v", e.Path, err)
	}
	validation := extractMarkdownSection(string(data), 2, "Validation")
	if validation == "" {
		t.Fatal("AC-4: M-0221 must carry a `## Validation` section recording the before/after render measurement")
	}
	// The record must be populated, not a draft — no leftover placeholder
	// tokens standing in for a measured value.
	if strings.Contains(validation, "PLACEHOLDER") {
		t.Error("AC-4: `## Validation` still contains an unfilled PLACEHOLDER token — the measurement was not recorded")
	}
	lower := strings.ToLower(validation)

	// A before/after axis is named.
	for _, axis := range []string{"before", "after"} {
		if !strings.Contains(lower, axis) {
			t.Errorf("AC-4: `## Validation` must name a %q wall-time", axis)
		}
	}

	// Both mechanisms are named: the old per-entity fan-out and the new
	// single shared HEAD pass (the M-0219 trap — name which mechanism you
	// measured).
	if !strings.Contains(lower, "per-entity") {
		t.Errorf("AC-4: `## Validation` must name the before mechanism (per-entity fan-out)")
	}
	if !strings.Contains(lower, "single") && !strings.Contains(validation, "WalkHeadCommits") {
		t.Errorf("AC-4: `## Validation` must name the after mechanism (single HEAD pass / WalkHeadCommits)")
	}

	// Actual wall-time values are recorded — at least three unit-bearing
	// tokens (before + after, with the after best-of runs). A populated
	// record, not a placeholder.
	times := regexp.MustCompile(`\d[\d.]*\s*(?:ms|min|s)\b`).FindAllString(validation, -1)
	if len(times) < 3 {
		t.Errorf("AC-4: `## Validation` must record wall-time values for before/after; found %d unit-bearing token(s) %v, want >=3", len(times), times)
	}
}
