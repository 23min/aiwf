package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// aiwfxWrapEpicFixturePath is the canonical authoring location for
// the `aiwfx-wrap-epic` skill body — the embedded ritual snapshot
// the aiwf binary ships. Per G-0182, AC content assertions read the
// embedded bytes directly rather than a duplicated fixture under
// internal/policies/testdata/. ADR-0014 retired the marketplace
// channel; the pending ADR-0016 follow-up retires the upstream
// authoring channel — in both states, the embedded snapshot is the
// source of truth.
const aiwfxWrapEpicFixturePath = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-epic/SKILL.md"

// loadAiwfxWrapEpicFixture reads the fixture relative to repo root.
// The tests under this file are seam-tests against the authored
// skill body — they assert the doctrinal content M-0090's ACs
// require, scoped to the relevant markdown section per CLAUDE.md
// *Testing* §"Substring assertions are not structural assertions".
func loadAiwfxWrapEpicFixture(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, aiwfxWrapEpicFixturePath))
	if err != nil {
		t.Fatalf("loading %s: %v", aiwfxWrapEpicFixturePath, err)
	}
	return string(data)
}

// TestAiwfxWrapEpic_AC1_FixtureExists asserts M-0090/AC-1 in the
// spec's intended landing zone (here: AC-1): the fixture SKILL.md
// is present at the canonical authoring location with frontmatter
// declaring `name: aiwfx-wrap-epic` and a non-empty `description:`.
func TestAiwfxWrapEpic_AC1_FixtureExists(t *testing.T) {
	t.Parallel()
	body := loadAiwfxWrapEpicFixture(t)

	name := frontmatterField(body, "name")
	if name != "aiwfx-wrap-epic" {
		t.Errorf("AC-1: frontmatter `name:` must be `aiwfx-wrap-epic` (got %q)", name)
	}

	desc := frontmatterField(body, "description")
	if desc == "" {
		t.Error("AC-1: frontmatter `description:` must be non-empty")
	}
}

// TestAiwfxWrapEpic_AC5_KernelRuleUnchanged was M-0090's
// implementation-window self-discipline: during M-0090's
// implementation, no commit may touch trailer_keys.go or
// principal_write_sites.go. M-0090 is `status: done` and archived
// under `work/epics/archive/E-0027-.../M-0090-...md` (milestones
// reach `done`; ACs reach `met`). AC-5 itself is `status: met,
// tdd_phase: done`. The implementation-window scope the AC defended
// lapsed by design when the milestone closed.
//
// Retired here so that future scope-bounded discipline tests don't
// silently outlive their milestone's window. Later wrap-epic
// milestones assert their own scope discipline if needed; an
// unbounded "kernel rule files never change" invariant would be
// stronger than any AC actually claimed.
//
// Kept as a comment so future readers (and `git log -S`-style
// searches for the test name) find the retirement rationale rather
// than a silent deletion.
//
// Original mechanism (for reference): shell out to
// `git diff --name-only <base>...HEAD` and assert no kernel rule
// files appear in the changed set.

// TestAiwfxWrapEpic_AC4_RitualsRepoSHARecordedAtWrap asserts M-0090
// AC-4 / spec AC-5: at wrap, the rituals-repo commit SHA that
// carries the fixture-copy is recorded in the milestone spec's
// *Validation* section. During implementation the section carries
// a `(pending: <sha-will-be-recorded-at-wrap>)` placeholder; at
// wrap, the placeholder is replaced with the real SHA.
//
// This test runs in two modes:
//   - Pre-wrap: the milestone spec's *Validation* section contains
//     the placeholder phrase. The test passes (placeholder is the
//     correct state during implementation).
//   - Post-wrap: the placeholder is gone and a 7-or-more-hex-char
//     SHA appears in the *Validation* section, marked as the
//     rituals-repo commit. The test passes.
//
// The test FAILS only when both conditions are absent — i.e. the
// milestone reached wrap without the SHA being recorded. That's
// the AC-5/AC-4 failure mode the spec calls out.
func TestAiwfxWrapEpic_AC4_RitualsRepoSHARecordedAtWrap(t *testing.T) {
	t.Parallel()
	// Resolve M-0090's spec via sharedRepoTree (per ADR-0004 +
	// M-0091/AC-4). A hardcoded path under work/epics/E-0027-.../
	// would break the moment `aiwf archive --apply` moves the
	// milestone into the per-kind archive/ subdir — the bug
	// enforced by PolicyNoHardcodedEntityPaths.
	root, tr := sharedRepoTree(t)
	e := tr.ByID("M-0090")
	if e == nil {
		t.Fatal("AC-4: milestone M-0090 not found in tree (active or archive)")
	}
	specPath := filepath.Join(root, e.Path)
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("AC-4: reading milestone spec %q: %v", specPath, err)
	}
	body := string(data)

	validation := extractMarkdownSection(body, 2, "Validation")
	if validation == "" {
		t.Fatal("AC-4: milestone spec must have a `## Validation` section to record the rituals-repo SHA at wrap")
	}

	// Acceptable placeholder phrasings during implementation.
	// Either of these is fine — they signal the SHA-record step
	// hasn't happened yet but is acknowledged.
	placeholders := []string{
		"pending",
		"recorded at wrap",
		"will be recorded",
	}
	hasPlaceholder := false
	lowerValidation := strings.ToLower(validation)
	for _, p := range placeholders {
		if strings.Contains(lowerValidation, p) {
			hasPlaceholder = true
			break
		}
	}

	// Post-wrap shape: a SHA (≥7 hex chars) accompanied by a
	// rituals-repo reference. The combination disambiguates from
	// any incidental hex string elsewhere in the section.
	shaPattern := regexp.MustCompile(`(?i)\b[0-9a-f]{7,40}\b`)
	hasSHA := shaPattern.MatchString(validation) &&
		(strings.Contains(lowerValidation, "rituals") ||
			strings.Contains(lowerValidation, "ai-workflow-rituals"))

	if !hasPlaceholder && !hasSHA {
		t.Errorf("AC-4: milestone spec's *Validation* section must either carry a `(pending)` placeholder (during implementation) or a rituals-repo SHA (post-wrap). Current section reads:\n%s", validation)
	}
}
