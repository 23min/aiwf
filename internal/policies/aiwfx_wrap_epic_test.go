package policies

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/tree"
)

// aiwfxWrapEpicFixturePath is the canonical authoring location for
// the `aiwfx-wrap-epic` skill body during M-0090, per CLAUDE.md
// §"Cross-repo plugin testing". At wrap, the fixture content is
// copied to the rituals plugin repo (`plugins/aiwf-extensions/
// skills/aiwfx-wrap-epic/SKILL.md` there); the cache-comparison
// drift-check below guards the long-term coupling.
const aiwfxWrapEpicFixturePath = "internal/policies/testdata/aiwfx-wrap-epic/SKILL.md"

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
		case strings.Contains(lower, "after commit approval"):
			// The wrap-artefact commit step's heading is "After
			// commit approval" — that's the step that emits the
			// CHANGELOG + wrap.md commit.
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
		t.Error("G-0119: `## Workflow` must contain a `### …After commit approval` step (the wrap-artefact commit)")
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

// TestAiwfxWrapEpic_AC3_CacheComparison asserts M-0090 AC-3 / spec
// AC-4: the fixture content matches the currently-active plugin
// install per `installed_plugins.json`. Skip semantics follow
// M-0079's precedent — if the cache is absent (CI without a plugin
// install) the test skips cleanly; if the plugin is installed but
// the skill is missing, the test fails ("not materialised"); if
// the cached content differs from the fixture, the test fails
// ("drift").
//
// AC-3 is the long-term drift-check chokepoint. On the M-0090
// commit itself it may legitimately stay red — the fixture is
// authored *here first*, then copied to the rituals repo as
// part of the wrap (per CLAUDE.md *Cross-repo plugin testing*).
// Once the wrap-time copy lands and `/reload-plugins` runs, the
// cache catches up and the test turns green. The acceptance
// criterion is "the test exists and asserts the right thing,"
// which holds independent of the cache's current state.
func TestAiwfxWrapEpic_AC3_CacheComparison(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	manifestPath := filepath.Join(home, ".claude", "plugins", "installed_plugins.json")
	manifest, err := os.ReadFile(manifestPath)
	if os.IsNotExist(err) {
		t.Skipf("AC-3 skip: %q not present; run after plugin install to verify drift-check", manifestPath)
	}
	if err != nil {
		t.Fatalf("AC-3: reading %q: %v", manifestPath, err)
	}

	// Resolve the *active* install path from installed_plugins.json,
	// not whichever sha-prefix directory `os.ReadDir` happens to
	// enumerate first (the cache typically holds several historical
	// versions).
	var parsed struct {
		Plugins map[string][]struct {
			InstallPath string `json:"installPath"`
		} `json:"plugins"`
	}
	if jsonErr := json.Unmarshal(manifest, &parsed); jsonErr != nil {
		t.Fatalf("AC-3: parsing %q: %v", manifestPath, jsonErr)
	}
	installs, ok := parsed.Plugins["aiwf-extensions@ai-workflow-rituals"]
	if !ok || len(installs) == 0 {
		t.Skipf("AC-3 skip: aiwf-extensions@ai-workflow-rituals not installed (no entry in %q)", manifestPath)
	}
	skillPath := filepath.Join(installs[0].InstallPath, "skills", "aiwfx-wrap-epic", "SKILL.md")
	if _, statErr := os.Stat(skillPath); os.IsNotExist(statErr) {
		t.Errorf("AC-3: aiwfx-wrap-epic not materialised in active install (expected at %q)", skillPath)
		return
	} else if statErr != nil {
		t.Fatalf("AC-3: stat %q: %v", skillPath, statErr)
	}

	cached, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("AC-3: reading cached skill at %q: %v", skillPath, err)
	}

	fixture := loadAiwfxWrapEpicFixture(t)
	if string(cached) != fixture {
		t.Errorf("AC-3: drift between fixture and cached skill at %q — re-deploy fixture to rituals repo and reload plugins, or update the fixture if the rituals-side is canonical", skillPath)
	}
}

// TestAiwfxWrapEpic_AC5_KernelRuleUnchanged asserts M-0090 AC-5 /
// spec AC-6: this milestone does not modify the kernel's untrailered
// -entity audit or its supporting rule files. The principle is that
// the chokepoint stays strict — the ritual aligns with the rule,
// the rule does not relax for the ritual. If a future PR
// accidentally bundles a rule-loosening edit, this test fires.
//
// Mechanism: shell out to `git diff` against the milestone's base
// commit (the merge-base with `main`) and assert no lines under the
// kernel rule files are touched. Skips cleanly when the env doesn't
// expose a base ref (e.g. a worktree without `main` reachable).
func TestAiwfxWrapEpic_AC5_KernelRuleUnchanged(t *testing.T) {
	root := repoRoot(t)
	// Files this milestone must not touch. The kernel's
	// `provenance-untrailered-entity-commit` finding is rendered
	// through the trailer-keys policy and the principal-write-sites
	// policy; either file growing a special-case here would imply
	// loosening the chokepoint.
	guarded := []string{
		"internal/policies/trailer_keys.go",
		"internal/policies/principal_write_sites.go",
	}

	// Resolve the merge base with origin/main if available, else
	// main. If neither exists in this worktree, skip — we can't
	// derive an authoritative base ref and a false-positive diff
	// (e.g., from a recently rebased main) would defeat the test.
	baseRef := ""
	for _, candidate := range []string{"origin/main", "main"} {
		cmd := exec.Command("git", "-C", root, "rev-parse", "--verify", candidate)
		if err := cmd.Run(); err == nil {
			baseRef = candidate
			break
		}
	}
	if baseRef == "" {
		t.Skip("AC-5 skip: no `main` ref reachable from this worktree; cannot derive milestone base for diff")
	}

	cmd := exec.Command("git", "-C", root, "diff", "--name-only", baseRef+"...HEAD")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("AC-5: `git diff --name-only %s...HEAD` failed: %v", baseRef, err)
	}
	changed := strings.Split(strings.TrimSpace(string(out)), "\n")
	changedSet := map[string]bool{}
	for _, f := range changed {
		if f != "" {
			changedSet[f] = true
		}
	}
	for _, f := range guarded {
		if changedSet[f] {
			t.Errorf("AC-5: kernel rule file %q must not be touched by this milestone — the chokepoint stays strict; align the ritual, do not relax the rule", f)
		}
	}
}

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
	root := repoRoot(t)
	// Resolve M-0090's spec via tree.Load so the lookup survives
	// archive sweeps (per ADR-0004). A hardcoded path under
	// work/epics/E-0027-.../ would break the moment `aiwf archive
	// --apply` moves the milestone into the per-kind archive/
	// subdir — the bug enforced by PolicyNoHardcodedEntityPaths.
	tr, _, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("AC-4: tree.Load: %v", err)
	}
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
