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
