package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// M-0200 (G-0298) — structural tests pinning the five prose/description
// polish fixes. Each test reads the authored skill body from the embedded
// ritual snapshot (the source of truth per ADR-0016) and asserts the
// corrected content, so a future edit that reintroduces the defect reddens.
//
// The two ritual-skill path literals below (plan-epic, wf-codebase-health)
// double as the M-0196 skill-edit→structural-test backstop references for
// those skills — a modified SKILL.md under embedded-rituals/** must be named
// verbatim in some internal/policies/*_test.go. aiwfx-whiteboard and
// aiwfx-wrap-epic are already referenced by their own test files; this file
// reuses their package-level path constants.
const (
	aiwfxPlanEpicFixturePath    = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-plan-epic/SKILL.md"
	wfCodebaseHealthFixturePath = "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-codebase-health/SKILL.md"
	// aiwf-retitle is a verb skill under embedded/ (not embedded-rituals/),
	// so the M-0196 backstop does not require this reference; the AC-promote
	// evidence discipline does.
	aiwfRetitleFixturePath = "internal/skills/embedded/aiwf-retitle/SKILL.md"
)

// loadPolishFixture reads a skill body relative to repo root.
func loadPolishFixture(t *testing.T, path string) string {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatalf("loading %s: %v", path, err)
	}
	return string(data)
}

// firstSentence returns the prefix of s up to (and excluding) the first
// sentence terminator ". " — the description's opening clause. If none is
// found the whole string is the opening.
func firstSentence(s string) string {
	if i := strings.Index(s, ". "); i != -1 {
		return s[:i]
	}
	return s
}

// TestFirstSentence_BranchCoverage exercises both reachable branches of the
// firstSentence helper (terminator present / absent).
func TestFirstSentence_BranchCoverage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, in, want string
	}{
		{"terminator present", "First one. Second one.", "First one"},
		{"no terminator", "Only a fragment", "Only a fragment"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := firstSentence(tc.in); got != tc.want {
				t.Errorf("firstSentence(%q) = %q; want %q", tc.in, got, tc.want)
			}
		})
	}
}

// stopHereRe matches the prohibited "stop … here" completion-boundary
// vocabulary in any of its inflections — "stop here", "stop-here",
// "stopping here", "stops here" — without matching the legitimate
// "stop and report" instruction (which carries no trailing "here").
var stopHereRe = regexp.MustCompile(`(?i)stop\w*[\s-]here`)

// TestProsePolish_AC1_CompletionForksShedPauseVocabulary asserts AC-1:
// aiwfx-plan-epic and aiwfx-wrap-epic keep their completion-boundary forks
// but carry no "pause" / "stop here" vocabulary — reframed as completion.
func TestProsePolish_AC1_CompletionForksShedPauseVocabulary(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path, reframeToken string
	}{
		{"plan-epic", aiwfxPlanEpicFixturePath, "complete for now"},
		{"wrap-epic", aiwfxWrapEpicFixturePath, "whatever's next"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := loadPolishFixture(t, tc.path)
			lower := strings.ToLower(body)
			if strings.Contains(lower, "pause") {
				t.Errorf("AC-1: %s must shed the prohibited completion-boundary word %q", tc.name, "pause")
			}
			if loc := stopHereRe.FindString(body); loc != "" {
				t.Errorf("AC-1: %s must shed the prohibited completion-boundary vocabulary (found %q)", tc.name, loc)
			}
			if !strings.Contains(lower, tc.reframeToken) {
				t.Errorf("AC-1: %s must carry its completion-reframe token %q (fork kept, vocabulary reframed)", tc.name, tc.reframeToken)
			}
		})
	}
}

// TestProsePolish_AC2_WhiteboardDescriptionCacheAndXref asserts AC-2: the
// aiwfx-whiteboard description states it writes a gitignored WHITEBOARD.md
// cache (not "no persisted artefact"), and the body's cache cross-reference
// points "above" (the section precedes the reference).
func TestProsePolish_AC2_WhiteboardDescriptionCacheAndXref(t *testing.T) {
	t.Parallel()
	body := loadAiwfxWhiteboardFixture(t)

	desc := frontmatterField(body, "description")
	if desc == "" {
		t.Fatal("AC-2: whiteboard description is empty")
	}
	if !strings.Contains(desc, "WHITEBOARD.md") {
		t.Error("AC-2: description must state it writes the gitignored `WHITEBOARD.md` cache")
	}
	if strings.Contains(strings.ToLower(desc), "no persisted artefact") {
		t.Error(`AC-2: description must drop the contradictory "no persisted artefact" claim`)
	}

	if !strings.Contains(body, "*Output cache* above") {
		t.Error("AC-2: body cache cross-reference must read `*Output cache* above` (the section is above the reference)")
	}
	if strings.Contains(body, "*Output cache* below") {
		t.Error("AC-2: body must not point `*Output cache* below` — the section precedes the reference")
	}
}

// TestProsePolish_AC3_CodebaseHealthLeadsWithRitualIdentity asserts AC-3:
// wf-codebase-health's description opens with its aiwf-ritual identity — the
// whole-codebase companion to wf-review-code's per-diff gate — rather than the
// generic "field guide of code-health principles" sentence it shared with the
// global code-health skill.
func TestProsePolish_AC3_CodebaseHealthLeadsWithRitualIdentity(t *testing.T) {
	t.Parallel()
	body := loadPolishFixture(t, wfCodebaseHealthFixturePath)
	desc := frontmatterField(body, "description")
	if desc == "" {
		t.Fatal("AC-3: wf-codebase-health description is empty")
	}

	opening := strings.ToLower(firstSentence(desc))
	if !strings.Contains(opening, "wf-review-code") {
		t.Errorf("AC-3: description opening must lead with the wf-review-code differentiator (got opening %q)", firstSentence(desc))
	}
	// Must not begin with the generic sentence shared with the global
	// code-health skill (the coin-flip selection collision the fix removes).
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(desc)), "stack-agnostic field guide of code-health principles") {
		t.Error(`AC-3: description must not open with the generic "Stack-agnostic field guide of code-health principles" sentence that collides with the global code-health skill`)
	}
}

// TestProsePolish_AC4_RetitleDropsRenameCollision asserts AC-4: aiwf-retitle's
// description no longer lists the "rename the title" trigger that collides with
// aiwf-rename's primary "rename" trigger, and retains a change/correct-title
// trigger.
func TestProsePolish_AC4_RetitleDropsRenameCollision(t *testing.T) {
	t.Parallel()
	body := loadPolishFixture(t, aiwfRetitleFixturePath)
	desc := frontmatterField(body, "description")
	if desc == "" {
		t.Fatal("AC-4: aiwf-retitle description is empty")
	}
	lower := strings.ToLower(desc)
	if strings.Contains(lower, "rename the title") {
		t.Error(`AC-4: description must drop the "rename the title" trigger that collides with aiwf-rename`)
	}
	if !strings.Contains(lower, "correct the title") {
		t.Error("AC-4: description must retain a change/correct-title trigger (e.g. `correct the title`)")
	}
}
