package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// aiwfxRecordDecisionFixturePath is the canonical authoring location for
// the `aiwfx-record-decision` skill body — the embedded ritual snapshot
// the aiwf binary ships (the source of truth per ADR-0016). AC content
// assertions read the embedded bytes directly per G-0182.
//
// Naming this path here is also what clears the M-0196 skill-edit →
// structural-test backstop for `aiwfx-record-decision` (G-0331): before
// M-0201 no test under internal/policies/ referenced this skill, so any
// edit to it (M-0201 routes its body fill through `aiwf edit-body` and
// adds the ADR-authoring note) would be flagged by
// PolicySkillEditStructuralTestBackstop. This file supplies the missing
// reference.
const aiwfxRecordDecisionFixturePath = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-record-decision/SKILL.md"

// loadAiwfxRecordDecisionFixture reads the skill body relative to repo root.
func loadAiwfxRecordDecisionFixture(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, aiwfxRecordDecisionFixturePath))
	if err != nil {
		t.Fatalf("loading %s: %v", aiwfxRecordDecisionFixturePath, err)
	}
	return string(data)
}

// TestAiwfxRecordDecision_AC1_BodyFillRoutesThroughEditBody pins AC-1 for
// the record-decision skill: the body-fill/commit step routes through
// `aiwf edit-body` (the trailered route) instead of a plain
// `git commit -m "docs(adr): ..."` that trips the kernel's
// provenance-untrailered-entity-commit finding on every recorded decision.
func TestAiwfxRecordDecision_AC1_BodyFillRoutesThroughEditBody(t *testing.T) {
	t.Parallel()
	body := loadAiwfxRecordDecisionFixture(t)

	// The commit step (### 7. …) must name the trailered verb.
	step7 := extractMarkdownSection(body, 3, "7.")
	if step7 == "" {
		t.Fatal("AC-1: `## Workflow` must retain a `### 7.` commit step for the body fill")
	}
	if !strings.Contains(step7, "aiwf edit-body") {
		t.Error("AC-1: record-decision's commit step must land the body fill via `aiwf edit-body` (the trailered route)")
	}

	// The untrailered plain-commit route must be gone entirely.
	if strings.Contains(body, `git commit -m "docs(adr):`) {
		t.Error("AC-1: record-decision must drop the untrailered `git commit -m \"docs(adr): …\"` body-fill route — use `aiwf edit-body` instead")
	}
}

// TestAiwfxRecordDecision_AC2_ADRAuthoringDiscipline pins AC-2: the
// record-decision skill carries the CLAUDE.md §"Authoring an ADR"
// discipline — it references the section by name and warns against writing
// gate/schedule language into an ADR body ("decision is decision").
func TestAiwfxRecordDecision_AC2_ADRAuthoringDiscipline(t *testing.T) {
	t.Parallel()
	body := loadAiwfxRecordDecisionFixture(t)

	if !strings.Contains(body, "Authoring an ADR") {
		t.Error(`AC-2: record-decision must point at CLAUDE.md §"Authoring an ADR" so the author inherits its discipline`)
	}
	// The distinctive doctrine phrase — "decision is decision" — anchors the
	// no-gate-language rule (no "ratify after X", no "status stays proposed
	// through Y" in an ADR body).
	if !strings.Contains(strings.ToLower(body), "decision is decision") {
		t.Error(`AC-2: record-decision must carry the "decision is decision" no-gate-language discipline (no gate/schedule language in ADR bodies)`)
	}
}

// TestAiwfxRecordDecision_RichTemplateSelfLocating pins G-0345: step 3 points
// the author at the *materialized* template path (`.claude/templates/…`, what
// an AI in a consumer repo can actually find) rather than an authoring-relative
// "this plugin's" reference, names the `aiwf update` self-heal when it's
// absent, and warns against reconstructing the format by copying an existing
// entity — the failure mode that produced an ADR missing its H1 and Date header.
func TestAiwfxRecordDecision_RichTemplateSelfLocating(t *testing.T) {
	t.Parallel()
	body := loadAiwfxRecordDecisionFixture(t)

	step3 := extractMarkdownSection(body, 3, "3.")
	if step3 == "" {
		t.Fatal("G-0345: record-decision must retain a `### 3.` rich-template step")
	}
	for _, want := range []string{
		".claude/templates/adr.md",      // materialized, locatable path for ADR
		".claude/templates/decision.md", // and for D-NNNN
		"aiwf update",                   // self-heal when the template isn't materialized
	} {
		if !strings.Contains(step3, want) {
			t.Errorf("G-0345: record-decision step 3 must name %q so the author locates the rich template instead of copying an existing entity", want)
		}
	}
	if !strings.Contains(strings.ToLower(step3), "copying an existing") {
		t.Error("G-0345: record-decision step 3 must warn against reconstructing the body by copying an existing entity (it drifts from the template and drops the H1/header)")
	}
	// The obsolete authoring-relative reference must be gone from step 3.
	if strings.Contains(step3, "this plugin's") {
		t.Error("G-0345: record-decision step 3 must drop the authoring-relative `this plugin's templates/…` reference (no live plugin exists)")
	}
}

// TestAiwfxRecordDecision_M0229_AC1_ReferencingDecisionSection pins M-0229/AC-1:
// the record-decision skill — the ritual that authors decisions — carries a
// `## Referencing a decision` section stating the self-contained reference
// rule. A behavioral skill states its fact directly and self-contained; it does
// not embed a repo-path link to a decision record or design doc under `docs/`;
// a decision's rationale lives in its own entry, not in a link from a
// behavioral skill.
//
// Section-scoped per CLAUDE.md §"Substring assertions are not structural
// assertions": each marker must appear inside the named section, not merely
// somewhere in the file (`docs/` already appears in the ADR-vs-D table up top).
// Every marker is absent before the section lands, so the assertion is
// non-vacuous — it goes red if the section is dropped or narrowed.
func TestAiwfxRecordDecision_M0229_AC1_ReferencingDecisionSection(t *testing.T) {
	t.Parallel()
	body := loadAiwfxRecordDecisionFixture(t)

	section := extractMarkdownSection(body, 2, "Referencing a decision")
	if section == "" {
		t.Fatal("M-0229/AC-1: record-decision must carry a `## Referencing a decision` section stating the self-contained reference rule")
	}
	lower := strings.ToLower(section)
	for _, m := range []struct{ name, needle string }{
		{"the self-contained framing", "self-contained"},
		{"the no-repo-link rule", "does not embed"},
		{"the docs/ non-shipping path", "docs/"},
		{"the rationale-in-its-own-entry rule", "rationale"},
		{"the own-entry rule", "own entry"},
	} {
		if !strings.Contains(lower, m.needle) {
			t.Errorf("M-0229/AC-1: `## Referencing a decision` section must state %s (substring %q)", m.name, m.needle)
		}
	}
}
