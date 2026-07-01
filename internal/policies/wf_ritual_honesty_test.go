package policies

import (
	"strings"
	"testing"
)

// wf-ritual honesty / reframe tests (M-0199 / G-0309, G-0297, G-0294). Each
// pins one corrected fact in a wf-* engineering ritual. The edited skills
// live under internal/skills/embedded-rituals/**, so referencing their
// paths here also discharges the skill-edit-structural-test-backstop
// (G-0220): every edited ritual SKILL.md is referenced by a test below.
//
// These are doc-shaped assertions — there is no kernel set to source-derive
// — so each is scoped to the named section (heading order or section-local
// content), never a flat body grep, per CLAUDE.md §"Substring assertions are
// not structural assertions".

const (
	wfTddCycleFixturePath   = "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-tdd-cycle/SKILL.md"
	wfReviewCodeFixturePath = "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-review-code/SKILL.md"
	wfDocLintFixturePath    = "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-doc-lint/SKILL.md"
)

// headingIndexContaining returns the line index of the first markdown
// heading line whose text contains sub, or -1. Restricting the match to
// heading lines (not any line) is what makes a positional order assertion
// structural rather than a substring coincidence.
func headingIndexContaining(body, sub string) int {
	for i, ln := range strings.Split(body, "\n") {
		if headingLevel(ln) > 0 && strings.Contains(ln, sub) {
			return i
		}
	}
	return -1
}

// countSubHeadings returns how many lines in section are markdown headings
// at exactly the given level.
func countSubHeadings(section string, level int) int {
	n := 0
	for _, ln := range strings.Split(section, "\n") {
		if headingLevel(ln) == level {
			n++
		}
	}
	return n
}

// TestWfTddCycle_RecordFollowsEvidence pins AC-1 (G-0309): the RECORD step
// (which promotes the AC to `met`) is narrated after the branch-coverage
// audit and the vacuity check — the "done" judgment sits after the evidence.
func TestWfTddCycle_RecordFollowsEvidence(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfTddCycleFixturePath)

	audit := headingIndexContaining(body, "Branch-coverage audit")
	vacuity := headingIndexContaining(body, "Vacuity check")
	record := headingIndexContaining(body, "RECORD")

	if audit < 0 {
		t.Fatal("wf-tdd-cycle has no 'Branch-coverage audit' heading")
	}
	if vacuity < 0 {
		t.Fatal("wf-tdd-cycle has no 'Vacuity check' heading")
	}
	if record < 0 {
		t.Fatal("wf-tdd-cycle has no 'RECORD' heading")
	}
	if record < audit {
		t.Errorf("RECORD heading (line %d) precedes the branch-coverage audit (line %d); the done-judgment must follow the evidence (G-0309)", record, audit)
	}
	if record < vacuity {
		t.Errorf("RECORD heading (line %d) precedes the vacuity check (line %d); the done-judgment must follow the evidence (G-0309)", record, vacuity)
	}
}

// TestWfTddCycle_RedRedundantAndForceSovereign pins AC-2 (G-0297): the RED
// phase-seed names re-running "redundant" not "idempotent", and the RECORD
// --force note is framed as a human-only sovereign act that bypasses the
// TDD audit.
func TestWfTddCycle_RedRedundantAndForceSovereign(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfTddCycleFixturePath)

	red := sectionUnder(body, "RED — Write")
	if red == "" {
		t.Fatal("wf-tdd-cycle has no 'RED — Write ...' section")
	}
	if strings.Contains(red, "idempotent") {
		t.Error("RED section still calls the phase-seed re-run \"idempotent\"; a step the FSM refuses errors on re-run — it is redundant, not idempotent (G-0297)")
	}
	if !strings.Contains(red, "redundant") {
		t.Error("RED section should describe the redundant phase-seed re-run (the FSM refuses red -> red, so skip it) (G-0297)")
	}

	record := sectionUnder(body, "RECORD")
	if record == "" {
		t.Fatal("wf-tdd-cycle has no 'RECORD' section")
	}
	if !strings.Contains(record, "--force") {
		t.Fatal("RECORD section no longer mentions --force; expected the reframed sovereign-act note")
	}
	rl := strings.ToLower(record)
	for _, want := range []string{"sovereign", "human", "bypass"} {
		if !strings.Contains(rl, want) {
			t.Errorf("RECORD --force note omits %q; it must frame --force as a human-only sovereign act that bypasses the TDD audit (G-0297)", want)
		}
	}
}

// TestBranchCoverageAudit_FramedAgentPerformed pins AC-3 (G-0297): both
// wf-tdd-cycle and wf-review-code state the branch-coverage audit is an
// agent-performed manual walk, and that a project's mechanical coverage
// gate is typically statement-level (so the manual walk supplies the
// branch-level assurance).
func TestBranchCoverageAudit_FramedAgentPerformed(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		path    string
		heading string
	}{
		{"wf-tdd-cycle", wfTddCycleFixturePath, "Branch-coverage audit"},
		{"wf-review-code", wfReviewCodeFixturePath, "Tests"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := readVerbSkill(t, tc.path)
			section := strings.ToLower(sectionUnder(body, tc.heading))
			if section == "" {
				t.Fatalf("%s has no %q section", tc.name, tc.heading)
			}
			if !strings.Contains(section, "agent-performed") {
				t.Errorf("%s %q section does not frame the branch-coverage audit as agent-performed (G-0297)", tc.name, tc.heading)
			}
			if !strings.Contains(section, "statement") {
				t.Errorf("%s %q section does not distinguish the statement-level mechanical gate from the manual branch-walk (G-0297)", tc.name, tc.heading)
			}
		})
	}
}

// TestWfDocLint_FourHeuristicsPlusStandaloneScan pins AC-4 (G-0294): the
// "What it checks" section carries exactly four numbered heuristics, and the
// repo-wide path-leak scan is a distinct section outside it; the
// block-on-zero anti-pattern scopes itself to the doc heuristics.
func TestWfDocLint_FourHeuristicsPlusStandaloneScan(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfDocLintFixturePath)

	checks := sectionUnder(body, "What it checks")
	if checks == "" {
		t.Fatal("wf-doc-lint has no 'What it checks' section")
	}
	if got := countSubHeadings(checks, 3); got != 4 {
		t.Errorf("'What it checks' has %d ### sub-headings; the doc heuristics must be exactly four (path-leak moves to its own section) (G-0294)", got)
	}

	// The repo-wide secret/path-leak scan must exist as its own top-level
	// section, distinct from the four docs-scoped heuristics.
	standalone := headingIndexContaining(body, "repo-wide")
	if standalone < 0 {
		t.Error("wf-doc-lint has no distinct 'repo-wide' secret / path-leak scanning section; the path-leak scan must not be one of the four doc heuristics (G-0294)")
	}

	// The block-on-zero anti-pattern must scope to the doc heuristics rather
	// than contradict the standalone deterministic gate.
	bz := lineContaining(strings.ToLower(body), "block-on-zero")
	if bz == "" {
		t.Fatal("wf-doc-lint no longer mentions the block-on-zero anti-pattern")
	}
	if !strings.Contains(bz, "heuristic") {
		t.Errorf("block-on-zero anti-pattern does not scope itself to the doc heuristics; as written it contradicts the standalone tool's legitimate gate (G-0294); line = %q", bz)
	}
}

// TestWfDocLint_SecretScanPrePushCIAndCurrentGitleaks pins AC-5 (G-0294):
// the reframed standalone-scan section recommends a pre-push hook + CI job
// (not pre-commit) using the current gitleaks git / gitleaks dir
// subcommands, and the deprecated `gitleaks detect` is gone.
func TestWfDocLint_SecretScanPrePushCIAndCurrentGitleaks(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfDocLintFixturePath)

	if strings.Contains(body, "gitleaks detect") {
		t.Error("wf-doc-lint still shows the deprecated `gitleaks detect` subcommand; use `gitleaks git` (history) / `gitleaks dir` (filesystem) (G-0294)")
	}

	section := sectionUnder(body, "repo-wide")
	if section == "" {
		t.Fatal("wf-doc-lint has no 'repo-wide' standalone-scan section")
	}
	sl := strings.ToLower(section)
	if !strings.Contains(sl, "pre-push") {
		t.Error("standalone-scan section does not recommend a pre-push hook (the push is the trust boundary) (G-0294)")
	}
	if !strings.Contains(section, "CI") {
		t.Error("standalone-scan section does not recommend a CI job (operator-independent chokepoint) (G-0294)")
	}
	// The gitleaks command lines live in a ```bash block whose `#` comments
	// truncate sectionUnder (it is not fence-aware), so assert the current
	// subcommands at body scope — they are unique tokens introduced only by
	// this section, and `gitleaks detect` is asserted absent above.
	bl := strings.ToLower(body)
	if !strings.Contains(bl, "gitleaks git") && !strings.Contains(bl, "gitleaks dir") {
		t.Error("wf-doc-lint does not use the current `gitleaks git` / `gitleaks dir` subcommands (G-0294)")
	}
	// pre-commit must survive only as the framed non-boundary, not as a
	// recommendation — so it appears alongside a latency/boundary framing.
	if strings.Contains(sl, "pre-commit") && !strings.Contains(sl, "latency") && !strings.Contains(sl, "boundary") {
		t.Error("standalone-scan section mentions pre-commit without framing it as a latency-taxing non-boundary; pre-commit must not read as the recommendation (G-0294)")
	}
}
