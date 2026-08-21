package policies

import (
	"strings"
	"testing"
)

// wf-codebase-health "H. Economy" section tests (G-0451). These structural
// tests pin the new section's prescriptions. The rubric path is the shared
// wfCodebaseHealthFixturePath const declared by a sibling policy test in
// this package, and the guidance path is g0343GuidanceFixturePath — both
// reused here rather than redeclared (H1 in practice: one source per path).
// Each assertion is scoped to a named markdown section — heading count or
// section-local content — never a flat body grep, per CLAUDE.md
// §"Substring assertions are not structural assertions".

// TestWfCodebaseHealth_EconomySectionHasThreeForces pins the section's shape:
// a `## H. Economy` carrying exactly three `###` principles — reuse over
// duplication, no dead weight, and additions carry. Dropping any of them fails
// here, as does an id cited by the shipped guidance digest that resolves to no
// heading in this rubric.
func TestWfCodebaseHealth_EconomySectionHasThreeForces(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfCodebaseHealthFixturePath)

	econ := extractMarkdownSection(body, 2, "H. Economy")
	if econ == "" {
		t.Fatal("wf-codebase-health must have a `## H. Economy` section")
	}
	if got := countSubHeadings(econ, 3); got != 3 {
		t.Errorf("`## H. Economy` has %d `###` principles; want exactly 3 (H1 reuse, H2 dead weight, H3 additions carry)", got)
	}
	for _, want := range []string{"Reuse over duplication", "No dead weight", "Additions carry"} {
		if !strings.Contains(econ, want) {
			t.Errorf("section H is missing a principle named %q", want)
		}
	}
}

// TestWfCodebaseHealth_ReuseForceNamesSearchBeforeWrite pins the headline
// habit of H1 — search for an existing unit before authoring, and treat the
// second copy as the extraction trigger. Without these the force degrades to
// a vague "don't duplicate" with no actionable write-time move.
func TestWfCodebaseHealth_ReuseForceNamesSearchBeforeWrite(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfCodebaseHealthFixturePath)

	h1 := extractMarkdownSection(body, 3, "H1.")
	if h1 == "" {
		t.Fatal("wf-codebase-health must have a `### H1. …` reuse principle")
	}
	low := strings.ToLower(h1)
	if !strings.Contains(low, "search") {
		t.Error("H1 must instruct searching for an existing unit before writing")
	}
	if !strings.Contains(low, "second copy") {
		t.Error("H1 must name the second copy as the extraction trigger")
	}
	if !strings.Contains(low, "route through") {
		t.Error("H1 must instruct routing through an existing helper rather than re-inlining")
	}
}

// TestWfCodebaseHealth_DeadWeightForceNamesGuardAndRemovalTrigger pins the
// two load-bearing moves of H2: the reference-only guard that silences the
// unused check is called out, and retained dead code requires a named
// removal trigger rather than silence.
func TestWfCodebaseHealth_DeadWeightForceNamesGuardAndRemovalTrigger(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfCodebaseHealthFixturePath)

	h2 := extractMarkdownSection(body, 3, "H2.")
	if h2 == "" {
		t.Fatal("wf-codebase-health must have a `### H2. …` dead-weight principle")
	}
	low := strings.ToLower(h2)
	if !strings.Contains(low, "unused") {
		t.Error("H2 must call out the reference-only guard that silences the unused-symbol check")
	}
	if !strings.Contains(low, "removal trigger") {
		t.Error("H2 must require a named removal trigger for dead code that must stay")
	}
}

// TestEmbeddedGuidance_PrimingCarriesReuseForce pins that the always-on
// code-health priming subset gained the reuse force (H1) — the one force
// worth priming every turn — so the highest-frequency habit is primed, not
// only reachable via the full rubric.
func TestEmbeddedGuidance_PrimingCarriesReuseForce(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, g0343GuidanceFixturePath)

	priming := extractMarkdownSection(body, 2, "Code-health priming")
	if priming == "" {
		t.Fatal("aiwf-guidance.md must have a `## Code-health priming` section")
	}
	low := strings.ToLower(priming)
	if !strings.Contains(low, "reuse over duplication") {
		t.Error("the always-on priming subset must carry the reuse-over-duplication force")
	}
}
