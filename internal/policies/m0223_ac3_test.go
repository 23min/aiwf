package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestM0223_AC3_ValidationRecordsReadVerbMeasurement asserts M-0223/AC-3:
// the milestone's `## Validation` section records a before/after wall-time
// measurement for BOTH read verbs (`aiwf history` and `aiwf show`), taken via
// performance.md's "How to measure" recipe. The absolute numbers are
// environment-specific and not gated; this pins that the record exists and
// names both verbs with actual before/after numbers — not an empty or
// half-filled section.
//
// Section-scoped per CLAUDE.md's "substring assertions are not structural
// assertions" rule: the entity is resolved through the loader (never a
// hardcoded work/ path), and every assertion is bounded to the `## Validation`
// section via extractMarkdownSection, not flat-grepped over the whole spec.
func TestM0223_AC3_ValidationRecordsReadVerbMeasurement(t *testing.T) {
	t.Parallel()
	root, tr := sharedRepoTree(t)
	e := tr.ByID("M-0223")
	if e == nil {
		t.Fatal("AC-3: M-0223 not found in tree")
	}
	data, err := os.ReadFile(filepath.Join(root, e.Path))
	if err != nil {
		t.Fatalf("reading M-0223 at %s: %v", e.Path, err)
	}
	validation := extractMarkdownSection(string(data), 2, "Validation")
	if validation == "" {
		t.Fatal("AC-3: M-0223 must carry a `## Validation` section recording the before/after read-verb measurement")
	}

	// Both read verbs measured.
	for _, verb := range []string{"aiwf history", "aiwf show"} {
		if !strings.Contains(validation, verb) {
			t.Errorf("AC-3: `## Validation` must record a measurement for `%s`", verb)
		}
	}

	// A before/after axis is named.
	lower := strings.ToLower(validation)
	for _, axis := range []string{"before", "after"} {
		if !strings.Contains(lower, axis) {
			t.Errorf("AC-3: `## Validation` must name a %q wall-time", axis)
		}
	}

	// Actual wall-time numbers are recorded — at least four (before + after
	// for each of the two verbs). A populated table, not a placeholder.
	times := regexp.MustCompile(`\d+\.\d+s`).FindAllString(validation, -1)
	if len(times) < 4 {
		t.Errorf("AC-3: `## Validation` must record before/after wall-times for both verbs; found %d time value(s) %v, want ≥4", len(times), times)
	}
}
