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

// TestAiwfxWrapEpic_AC2_TrailerSequenceInMergeStep asserts M-0090
// AC-2 / spec's AC-2: the merge-step section names the trailered-
// commit sequence (no-commit merge followed by `git commit
// --trailer ...` with the three kernel-required trailer keys).
//
// This test together with AC-6 below pins the structural claim —
// AC-2 asserts the *content* (the right shell strings, in the right
// merge step); AC-6 (TestAiwfxWrapEpic_AC6_StructuralMergeStepDriftCheck)
// asserts the same content lives inside the named section, not
// merely "somewhere in the file." Both are required because a flat
// substring match passes vacuously if the merge instructions are
// reshaped into an unrelated section.
func TestAiwfxWrapEpic_AC2_TrailerSequenceInMergeStep(t *testing.T) {
	t.Parallel()
	body := loadAiwfxWrapEpicFixture(t)

	// AC-6's structural locator finds the merge-step section by
	// walking the heading hierarchy. AC-2 reuses that to ensure
	// the content claims are scoped to the same place. If AC-6's
	// section-extraction fails, AC-2 cannot meaningfully run.
	section := findMergeStepSection(body)
	if section == "" {
		t.Fatal("AC-2: merge-step section not found (see AC-6 for the structural drift-check on heading shape)")
	}

	// The two-step trailered merge: stage with --no-commit, then
	// commit with explicit trailers. Both halves must be named
	// in the merge-step section.
	wantStage := "git merge --no-ff --no-commit"
	if !strings.Contains(section, wantStage) {
		t.Errorf("AC-2: merge-step section must name the staged-merge step (substring %q) so the trailered commit follows", wantStage)
	}

	// Conventional Commits-shaped subject for the wrap commit.
	// The skill instruction's subject template (chore(epic): wrap
	// E-NNNN — <title>) is what CLAUDE.md §"Commit conventions"
	// requires; assert by the leading shape.
	if !regexp.MustCompile(`chore\(epic\):\s+wrap\s+E-NNNN`).MatchString(section) {
		t.Error("AC-2: merge-step section must use a Conventional Commits subject template `chore(epic): wrap E-NNNN — <title>`")
	}
}

// TestAiwfxWrapEpic_AC6_StructuralMergeStepDriftCheck asserts
// M-0090 AC-6 / spec's AC-3: the drift-check is structural — the
// trailered-merge instructions live inside the `## Workflow` →
// `### <merge-step>` section, not floating elsewhere. Per CLAUDE.md
// *Substring assertions are not structural assertions*, this test
// walks the heading hierarchy and scopes the content claim to the
// named section.
//
// Concretely: locate the merge-step subsection inside `## Workflow`
// (heading text contains "Merge epic branch"), then assert each
// required trailer flag appears *inside that subsection*. A
// reshuffle of the SKILL.md that moved the trailered commit into
// (say) the "Branch cleanup" section would fail this test even if
// a flat grep over the file still passed.
func TestAiwfxWrapEpic_AC6_StructuralMergeStepDriftCheck(t *testing.T) {
	t.Parallel()
	body := loadAiwfxWrapEpicFixture(t)

	// Step 1: the parent section exists.
	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		t.Fatal("AC-6: SKILL.md must have a `## Workflow` section that contains the merge step as a subsection")
	}

	// Step 2: the merge-step subsection is reachable via a `### `
	// heading whose text references the merge action. Match the
	// heading prefix (the heading carries a trailing rationale
	// after `—`), so a heading like
	//   `### 5. 🛑 Merge gate — merge epic branch into integration target`
	// matches the prefix locator.
	merge := findMergeStepSection(body)
	if merge == "" {
		t.Fatal("AC-6: `## Workflow` must contain a `### …merge…` subsection that documents the integration-branch merge")
	}

	// Step 3: each required trailer flag is named *inside* the
	// merge-step subsection. The keys are quoted from CLAUDE.md
	// §"Commit conventions" verbatim — variant casings (e.g.
	// `Aiwf-Verb`) would fail the kernel's trailer-keys policy and
	// must fail this test too.
	requiredTrailerFlags := []string{
		`--trailer "aiwf-verb: wrap-epic"`,
		`--trailer "aiwf-entity: E-NNNN"`,
		`--trailer "aiwf-actor: human/`,
	}
	for _, flag := range requiredTrailerFlags {
		if !strings.Contains(merge, flag) {
			t.Errorf("AC-6: merge-step subsection must name the trailer flag %q (in the right section, not just somewhere in the file)", flag)
		}
	}

	// Step 4: the trailered commit *follows* the staged merge.
	// Ordering matters — a fixture that documented the trailer
	// flags first and then a plain `git merge --no-ff` would
	// produce an untrailered commit at run time. Assert by index.
	stageIdx := strings.Index(merge, "git merge --no-ff --no-commit")
	commitIdx := strings.Index(merge, `--trailer "aiwf-verb: wrap-epic"`)
	if stageIdx < 0 || commitIdx < 0 {
		t.Fatal("AC-6: merge-step subsection must contain both the staged-merge command and the trailer-emitting commit")
	}
	if stageIdx > commitIdx {
		t.Error("AC-6: the staged-merge (`--no-commit`) must appear *before* the trailered `git commit` so the commit-emitting step is the one carrying trailers")
	}
}

// findMergeStepSection returns the body of the merge-step
// subsection inside `## Workflow`. The subsection's `###` heading
// is identified by a case-insensitive substring match on "merge"
// AND "epic branch" (the canonical phrasing in M-0090's design
// notes). Returns "" if no matching subsection is found.
//
// The two-substring match makes the locator resilient to small
// heading-text rewordings ("Merge gate — merge epic branch …")
// without matching unrelated `###` headings that happen to contain
// the word "merge" in a different sense (e.g. "Merge milestone
// specs into the wrap artefact").
func findMergeStepSection(body string) string {
	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		return ""
	}
	// Walk the workflow body line-by-line, find each `### ` heading,
	// and pick the first whose text contains both "merge" and "epic
	// branch" (case-insensitive). Then extract that subsection from
	// the full body via the matched heading-prefix.
	lines := strings.Split(workflow, "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		text := strings.TrimPrefix(line, "### ")
		lower := strings.ToLower(text)
		if strings.Contains(lower, "merge") && strings.Contains(lower, "epic branch") {
			// Use the literal heading text as the prefix locator
			// against the full body. extractMarkdownSection's
			// prefix-match keys off the start of the heading text.
			return extractMarkdownSection(body, 3, text)
		}
	}
	return ""
}

// TestAiwfxWrapEpic_G0119_PromoteIsLastCommitInBundle asserts the
// closing condition for G-0119: the `aiwf promote E-NN done` step
// must appear *after* every other commit-emitting step in the
// `## Workflow` section. Specifically: after the merge gate, after
// the wrap-artefact commit, and before the push gate.
//
// Rationale: `aiwf promote E-NN done` ends the authorize scope
// opened by `aiwfx-start-epic`. Any commit produced after the
// promote carries `aiwf-authorized-by:` referencing an ended scope
// and triggers the kernel's `provenance-authorization-ended`
// finding on push, blocking the wrap with no clean remediation
// short of `--no-verify`. The skill must order the bundle so the
// promote is the last commit before push.
//
// This is a structural test per CLAUDE.md *Substring assertions
// are not structural assertions*: it walks the `## Workflow`
// heading hierarchy, locates each step's `### ` heading, and
// asserts ordering by line index — so a reshuffle that moved the
// promote step back to its pre-fix position would fail this test
// even if the literal phrases still appeared somewhere in the
// fixture.
func TestAiwfxWrapEpic_G0119_PromoteIsLastCommitInBundle(t *testing.T) {
	t.Parallel()
	body := loadAiwfxWrapEpicFixture(t)

	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		t.Fatal("G-0119: SKILL.md must have a `## Workflow` section")
	}

	// Walk Workflow's `### ` headings, capturing each step's body-
	// relative line index. We need promote, wrap-artefact commit,
	// merge gate, and push gate.
	mergeIdx, wrapArtefactIdx, promoteIdx, pushIdx := -1, -1, -1, -1
	for i, line := range strings.Split(workflow, "\n") {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		lower := strings.ToLower(strings.TrimPrefix(line, "### "))
		switch {
		case strings.Contains(lower, "merge") && strings.Contains(lower, "epic branch"):
			mergeIdx = i
		case strings.Contains(lower, "wrap-artefact commit"):
			// The wrap-artefact commit step's heading is
			// "Wrap-artefact commit — CHANGELOG + wrap.md" — the step
			// that emits the CHANGELOG + wrap.md commit. (Pre-M-0209
			// this was a separate "After commit approval" step with its
			// own commit gate; M-0209 folded that gate into the
			// step-4 declared-sequence gate, but the commit step — and
			// its position in the promote-last ordering — remains.)
			wrapArtefactIdx = i
		case strings.Contains(lower, "promote the epic to `done`"):
			promoteIdx = i
		case strings.Contains(lower, "push gate"):
			pushIdx = i
		}
	}

	if mergeIdx < 0 {
		t.Error("G-0119: `## Workflow` must contain a `### …merge…epic branch…` step")
	}
	if wrapArtefactIdx < 0 {
		t.Error("G-0119: `## Workflow` must contain a `### …Wrap-artefact commit…` step (the CHANGELOG + wrap.md commit)")
	}
	if promoteIdx < 0 {
		t.Error("G-0119: `## Workflow` must contain a `### …Promote the epic to `done`…` step")
	}
	if pushIdx < 0 {
		t.Error("G-0119: `## Workflow` must contain a `### …Push gate` step")
	}
	if mergeIdx < 0 || wrapArtefactIdx < 0 || promoteIdx < 0 || pushIdx < 0 {
		return
	}

	// The promote step is last among commit-emitting steps and
	// strictly before the push gate.
	if mergeIdx >= wrapArtefactIdx || wrapArtefactIdx >= promoteIdx || promoteIdx >= pushIdx {
		t.Errorf("G-0119: wrap-bundle ordering must be merge → wrap-artefact commit → promote → push (got line indices: merge=%d, wrap-artefact=%d, promote=%d, push=%d). The promote must be the last commit before push so the authorize scope is still live for every other wrap commit.", mergeIdx, wrapArtefactIdx, promoteIdx, pushIdx)
	}
}

// TestAiwfxWrapEpic_RoadmapRegenWriteOnlyAfterPromote pins G-0350's fix:
// the roadmap-regen step lands between the promote-done and push-gate
// steps (so it captures the epic's actual final `done` status rather
// than a stale pre-promote snapshot), documents that `--write` no
// longer commits, and hand-composes its own commit (never routing
// through a kernel verb) — the one deliberate exception to "promote is
// last among entity-mutating commits" the "Why promote is last" section
// carves out. Mirrors TestAiwfxWrapEpic_G0119_PromoteIsLastCommitInBundle's
// heading-walk shape.
func TestAiwfxWrapEpic_RoadmapRegenWriteOnlyAfterPromote(t *testing.T) {
	t.Parallel()
	body := loadAiwfxWrapEpicFixture(t)

	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		t.Fatal("SKILL.md must have a `## Workflow` section")
	}

	promoteIdx, regenIdx, pushIdx := -1, -1, -1
	for i, line := range strings.Split(workflow, "\n") {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		lower := strings.ToLower(strings.TrimPrefix(line, "### "))
		switch {
		case strings.Contains(lower, "promote the epic to `done`"):
			promoteIdx = i
		case strings.Contains(lower, "regenerate the roadmap"):
			regenIdx = i
		case strings.Contains(lower, "push gate"):
			pushIdx = i
		}
	}
	if promoteIdx < 0 {
		t.Fatal("`## Workflow` must contain a `### …Promote the epic to `done`…` step")
	}
	if regenIdx < 0 {
		t.Fatal("`## Workflow` must contain a `### …Regenerate the roadmap` step")
	}
	if pushIdx < 0 {
		t.Fatal("`## Workflow` must contain a `### …Push gate` step")
	}
	if promoteIdx >= regenIdx || regenIdx >= pushIdx {
		t.Errorf("roadmap-regen step must land after promote-done and before the push gate (got line indices: promote=%d, regen=%d, push=%d)", promoteIdx, regenIdx, pushIdx)
	}

	regen := extractMarkdownSection(body, 3, "9. Regenerate the roadmap")
	if regen == "" {
		t.Fatal("could not extract the `### 9. Regenerate the roadmap` section")
	}
	if !strings.Contains(regen, "aiwf render roadmap --write") {
		t.Error("roadmap-regen step must run `aiwf render roadmap --write`")
	}
	if !strings.Contains(regen, "never commits") {
		t.Error("roadmap-regen step must document that --write never commits (G-0350)")
	}
	requiredTrailerFlags := []string{
		`--trailer "aiwf-verb: wrap-epic"`,
		`--trailer "aiwf-entity: E-NNNN"`,
		`--trailer "aiwf-actor: human/`,
	}
	for _, flag := range requiredTrailerFlags {
		if !strings.Contains(regen, flag) {
			t.Errorf("roadmap-regen step must hand-compose the commit with trailer flag %q", flag)
		}
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

// TestAiwfxWrapEpic_ReconcileMainlineBeforeMerge pins the
// reconcile-first practice: before the epic-to-mainline merge, if
// mainline has advanced past the epic branch's fork point, mainline
// must be integrated into the epic branch and the full local gate
// re-run there — never resolved on mainline itself, mid-merge.
//
// Structural per CLAUDE.md *Substring assertions are not structural
// assertions*: the reconcile step must exist as its own `### `
// subsection inside `## Workflow`, positioned BEFORE the merge
// subsection (not folded into the merge step's own prose, and not
// appearing after the merge already ran). The precondition must also
// name the "after integrating current mainline" qualifier — "gate
// green on the epic branch" alone is the exact gap this pins closed:
// a gate run that predates mainline's latest commits is green on a
// tree that omits them.
func TestAiwfxWrapEpic_ReconcileMainlineBeforeMerge(t *testing.T) {
	t.Parallel()
	body := loadAiwfxWrapEpicFixture(t)

	precondition := extractMarkdownSection(body, 2, "Precondition")
	if precondition == "" {
		t.Fatal("SKILL.md must have a `## Precondition` section")
	}
	if !strings.Contains(strings.ToLower(precondition), "after integrating current mainline") {
		t.Error("Precondition section must require the full local CI gate green on the epic branch AFTER integrating current mainline, not merely green on the epic branch in isolation")
	}

	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		t.Fatal("SKILL.md must have a `## Workflow` section")
	}
	reconcileIdx, mergeIdx := -1, -1
	for i, line := range strings.Split(workflow, "\n") {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		text := strings.ToLower(strings.TrimPrefix(line, "### "))
		switch {
		case reconcileIdx < 0 && strings.Contains(text, "reconcile") && strings.Contains(text, "mainline"):
			reconcileIdx = i
		case mergeIdx < 0 && strings.Contains(text, "merge epic branch"):
			mergeIdx = i
		}
	}
	if reconcileIdx < 0 {
		t.Fatal("`## Workflow` must contain a `### …Reconcile…mainline…` step")
	}
	if mergeIdx < 0 {
		t.Fatal("`## Workflow` must contain a `### …Merge epic branch…` step")
	}
	if reconcileIdx >= mergeIdx {
		t.Errorf("the reconcile step must appear BEFORE the merge step in `## Workflow` (reconcile at line %d, merge at line %d), so the merge that follows is already-validated", reconcileIdx, mergeIdx)
	}

	reconcile := extractMarkdownSection(body, 3, "5. Reconcile")
	if reconcile == "" {
		t.Fatal("could not extract the `### 5. Reconcile the epic branch with mainline` section")
	}

	// The ancestor guard compares against *local* mainline, not the
	// remote-tracking ref: local `main` advancing under a concurrent
	// session is a divergence `origin/main` would not reflect. The
	// remote-tracking ref appears only in the fetch/fast-forward
	// preamble that folds in the origin axis before the check.
	wantGuard := "git merge-base --is-ancestor main epic/E-NN-<slug>"
	if !strings.Contains(reconcile, wantGuard) {
		t.Errorf("reconcile step must name the ancestor guard %q (local mainline, not origin/main)", wantGuard)
	}

	// The fetch/fast-forward preamble folds in commits another clone
	// pushed before the ancestor guard runs, and must precede it.
	fetchIdx := strings.Index(reconcile, "git fetch")
	ffIdx := strings.Index(reconcile, "--ff-only origin/main")
	guardIdx := strings.Index(reconcile, wantGuard)
	if fetchIdx < 0 || ffIdx < 0 {
		t.Fatal("reconcile step must document `git fetch` and fast-forwarding local main to origin/main")
	}
	if fetchIdx >= guardIdx || ffIdx >= guardIdx {
		t.Errorf("reconcile step must run the fetch/fast-forward preamble BEFORE the ancestor guard (fetch=%d, ff=%d, guard=%d)", fetchIdx, ffIdx, guardIdx)
	}

	// Ordering: integrate mainline into the epic branch, then re-run
	// the gate, then (and only then) proceed to the merge step.
	integrateIdx := strings.Index(reconcile, "integrate mainline into the epic branch")
	gateIdx := strings.Index(reconcile, "re-run the project's full local CI gate")
	proceedIdx := strings.Index(reconcile, "Only once the check passes does the merge")
	if integrateIdx < 0 || gateIdx < 0 || proceedIdx < 0 {
		t.Fatal("reconcile step must document integrate-mainline, re-run-the-gate, and only-then-proceed-to-merge")
	}
	if integrateIdx >= gateIdx || gateIdx >= proceedIdx {
		t.Errorf("reconcile step must order integrate mainline -> re-run gate -> only then proceed to merge (got indices %d, %d, %d)", integrateIdx, gateIdx, proceedIdx)
	}
}
