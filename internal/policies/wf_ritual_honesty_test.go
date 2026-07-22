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

// TestWfTddCycle_ForceSovereign pins the RECORD --force note (G-0297): it is
// framed as a human-only sovereign act, and states that --force gives no
// `--force met` shortcut past the acs-tdd-audit — force relaxes only the FSM
// transition check, not the projection audit.
//
// The RED phase-seed half of the original G-0297 assertion (re-running the
// seed is "redundant", not "idempotent") is retired: M-0274/AC-4 makes the
// "" → red promote a live, mandatory step rather than a skippable redundant
// re-run, so TestM0274_TddCycleRedPromoteIsLiveMandatory now pins the RED
// step instead.
func TestWfTddCycle_ForceSovereign(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfTddCycleFixturePath)

	record := sectionUnder(body, "RECORD")
	if record == "" {
		t.Fatal("wf-tdd-cycle has no 'RECORD' section")
	}
	if !strings.Contains(record, "--force") {
		t.Fatal("RECORD section no longer mentions --force; expected the reframed sovereign-act note")
	}
	rl := strings.ToLower(record)
	// The corrected note (B1): --force is human-only / sovereign, AND the
	// acs-tdd-audit refuses `met` regardless of --force — there is no
	// `--force met` bypass (force relaxes only the FSM transition check,
	// not the projection audit). NOTE: do not anchor on "regardless" alone
	// — the RECORD section already carries an unrelated "(regardless of
	// project framework)" bullet, so that token matches vacuously. Anchor
	// on the human-only/sovereign framing plus a phrase that force gives no
	// path to met — both of which appear only in the corrected force note.
	for _, want := range []string{"sovereign", "human"} {
		if !strings.Contains(rl, want) {
			t.Errorf("RECORD --force note omits %q; --force must be framed as a human-only sovereign act (G-0297)", want)
		}
	}
	if !strings.Contains(rl, "shortcut") && !strings.Contains(rl, "does not get") {
		t.Error("RECORD --force note must state --force gives no path to `met` ahead of `done` (the acs-tdd-audit refuses regardless of --force); found neither \"shortcut\" nor \"does not get\" — the false \"--force bypasses the audit\" claim must not return (G-0297)")
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

// TestWfDocLint_SevenHeuristicsPlusStandaloneScan pins the "What it checks"
// section carrying exactly seven numbered heuristics (the original four plus
// link integrity, CLI-invocation resolution, and structural checks), with
// the repo-wide path-leak scan remaining a distinct section outside it; the
// block-on-zero anti-pattern scopes itself to the doc heuristics.
func TestWfDocLint_SevenHeuristicsPlusStandaloneScan(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfDocLintFixturePath)

	checks := sectionUnder(body, "What it checks")
	if checks == "" {
		t.Fatal("wf-doc-lint has no 'What it checks' section")
	}
	if got := countSubHeadings(checks, 3); got != 7 {
		t.Errorf("'What it checks' has %d ### sub-headings; the doc heuristics must be exactly seven (path-leak moves to its own section)", got)
	}
	for _, want := range []string{"Markdown link integrity", "CLI-invocation resolution", "Structural checks"} {
		if headingIndexContaining(checks, want) < 0 {
			t.Errorf("'What it checks' has no %q sub-heading", want)
		}
	}

	// The repo-wide secret/path-leak scan must exist as its own top-level
	// section, distinct from the seven docs-scoped heuristics.
	standalone := headingIndexContaining(body, "repo-wide")
	if standalone < 0 {
		t.Error("wf-doc-lint has no distinct 'repo-wide' secret / path-leak scanning section; the path-leak scan must not be one of the doc heuristics (G-0294)")
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

// TestWfDocLint_ScopeWidenedToRootNarrativeFiles pins G-0390: the docs-root
// default widens to include the repo's hand-authored root narrative files
// (README.md, CONTRIBUTING.md), while generated/gitignored root files and
// the append-only CHANGELOG.md are explicitly carried as exclusions, and the
// orphan-documents check is called out as inapplicable to root files.
func TestWfDocLint_ScopeWidenedToRootNarrativeFiles(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfDocLintFixturePath)

	workflow := sectionUnder(body, "Workflow")
	if workflow == "" {
		t.Fatal("wf-doc-lint has no 'Workflow' section")
	}
	// Scope to the "Default:" line specifically, not the whole Workflow
	// section — README.md is also mentioned pre-widening ("look for the
	// obvious folder"), so a section-wide Contains would pass unchanged.
	defaultLine := lineContaining(workflow, "Default:")
	if defaultLine == "" {
		t.Fatal("Workflow section has no 'Default:' docs-root line")
	}
	for _, want := range []string{"README.md", "CONTRIBUTING.md"} {
		if !strings.Contains(defaultLine, want) {
			t.Errorf("docs-root 'Default:' line does not widen to include %q; line = %q", want, defaultLine)
		}
	}
	for _, excluded := range []string{"ROADMAP.md", "STATUS.md", "WHITEBOARD.md", "TODO.md"} {
		if !strings.Contains(workflow, excluded) {
			t.Errorf("Workflow section does not name %q among the generated/gitignored root files excluded from scope", excluded)
		}
	}
	if !strings.Contains(workflow, "CHANGELOG.md") {
		t.Error("Workflow section does not name CHANGELOG.md as excluded append-only history")
	}
	if !strings.Contains(strings.ToLower(workflow), "orphan documents") {
		t.Error("Workflow section does not call out the orphan-documents check's inapplicability to root narrative files")
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
	// AC-5 documents BOTH current subcommands (history + filesystem), so
	// require both — || fails if either is missing (a && would only fire
	// when both are absent).
	if !strings.Contains(bl, "gitleaks git") || !strings.Contains(bl, "gitleaks dir") {
		t.Error("wf-doc-lint must document both current subcommands `gitleaks git` (history) and `gitleaks dir` (filesystem) (G-0294)")
	}
	// pre-commit must survive only as the framed non-boundary, not as a
	// recommendation — so it appears alongside a latency/boundary framing.
	if strings.Contains(sl, "pre-commit") && !strings.Contains(sl, "latency") && !strings.Contains(sl, "boundary") {
		t.Error("standalone-scan section mentions pre-commit without framing it as a latency-taxing non-boundary; pre-commit must not read as the recommendation (G-0294)")
	}
}
