package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// verb-skill factual-correction tests (M-0198 / G-0301). Each pins one
// corrected fact in an aiwf-* verb skill; the AC-1 and AC-3 assertions
// are source-derived (they read the kernel set they document), so the
// skill cannot silently drift from the kernel again.

const (
	aiwfContractSkillPath  = "internal/skills/embedded/aiwf-contract/SKILL.md"
	aiwfAuthorizeSkillPath = "internal/skills/embedded/aiwf-authorize/SKILL.md"
	aiwfAddSkillPath       = "internal/skills/embedded/aiwf-add/SKILL.md"
)

// readVerbSkill reads a verb skill body relative to the repo root.
func readVerbSkill(t *testing.T, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

// lineContaining returns the first line of body that contains sub, or ""
// — used to scope an assertion to the table row that carries a fact
// rather than grepping the whole body (structural, not substring).
func lineContaining(body, sub string) string {
	for _, ln := range strings.Split(body, "\n") {
		if strings.Contains(ln, sub) {
			return ln
		}
	}
	return ""
}

// headingLevel returns the number of leading '#' on a markdown heading
// line, or 0 if the line is not a heading.
func headingLevel(ln string) int {
	if !strings.HasPrefix(ln, "#") {
		return 0
	}
	return len(ln) - len(strings.TrimLeft(ln, "#"))
}

// sectionUnder returns the body text from the first heading containing
// headingSub up to (not including) the next heading of the same-or-
// shallower level. Scopes an assertion to one section so a fact in an
// unrelated section (e.g. an FSM diagram elsewhere) doesn't satisfy it.
//
// Caveat: it is NOT markdown-code-fence-aware — a `#`-prefixed comment
// line inside a ```bash block reads as a heading and truncates the
// returned section early. When a section-scoped assertion must reach
// content that sits after a fenced block containing `#` comments, assert
// that (uniquely-named) content at body scope instead (see
// TestWfDocLint_SecretScanPrePushCIAndCurrentGitleaks).
func sectionUnder(body, headingSub string) string {
	lines := strings.Split(body, "\n")
	start, level := -1, 0
	for i, ln := range lines {
		if headingLevel(ln) > 0 && strings.Contains(ln, headingSub) {
			start, level = i, headingLevel(ln)
			break
		}
	}
	if start < 0 {
		return ""
	}
	var b strings.Builder
	for i := start + 1; i < len(lines); i++ {
		if l := headingLevel(lines[i]); l > 0 && l <= level {
			break
		}
		b.WriteString(lines[i])
		b.WriteString("\n")
	}
	return b.String()
}

// TestAiwfCheckSkill_ACStatusSetMatchesKernel pins AC-1: the aiwf-check
// skill's acs-shape/status row names the full kernel AC status set.
func TestAiwfCheckSkill_ACStatusSetMatchesKernel(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, aiwfCheckSkillPath)
	row := lineContaining(body, "acs-shape/status")
	if row == "" {
		t.Fatal("aiwf-check skill has no acs-shape/status row")
	}
	for _, s := range entity.AllowedACStatuses() {
		if !strings.Contains(row, s) {
			t.Errorf("acs-shape/status row omits kernel AC status %q; row = %q", s, row)
		}
	}
	if strings.Contains(row, "three") {
		t.Errorf("acs-shape/status row still says \"three\" statuses (kernel has %d); row = %q",
			len(entity.AllowedACStatuses()), row)
	}
}

// TestAiwfArchiveSkill_NoFindingsAsKind pins AC-2: the aiwf-archive skill
// does not present "findings" as an archivable kind.
func TestAiwfArchiveSkill_NoFindingsAsKind(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, aiwfArchiveSkillPath)
	// The specific error was the prose "typically gaps or findings" —
	// findings is not an entity kind. "findings" legitimately appears
	// elsewhere (the skill discusses check findings), so guard the exact
	// kind-offender phrasing rather than the bare word.
	if strings.Contains(body, "or findings") {
		t.Errorf("aiwf-archive skill still lists \"findings\" as a kind (\"or findings\"); " +
			"the archivable kinds are epic, contract, gap, decision, adr")
	}
	// Source-derived, stronger: every `--kind <x>` the skill demonstrates
	// must name a real entity kind (entity.AllKinds()) — catches a
	// fabricated flag like `--kind findings` that the prose guard misses.
	valid := map[string]bool{}
	for _, k := range entity.AllKinds() {
		valid[string(k)] = true
	}
	for _, m := range regexp.MustCompile(`--kind\s+([a-z]+)`).FindAllStringSubmatch(body, -1) {
		if !valid[m[1]] {
			t.Errorf("aiwf-archive shows `--kind %s`, which is not a real entity kind (kinds: %v)", m[1], entity.AllKinds())
		}
	}
}

// TestAiwfContractSkill_RecipePathAndCancelFSM pins AC-3: the aiwf-contract
// skill cites the real recipe path and the complete contract cancel FSM.
func TestAiwfContractSkill_RecipePathAndCancelFSM(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, aiwfContractSkillPath)

	const realPath = "internal/recipe/embedded"
	if !strings.Contains(body, realPath) {
		t.Errorf("aiwf-contract skill does not reference the real recipe path %q", realPath)
	}
	if strings.Contains(body, "tools/"+realPath) {
		t.Errorf("aiwf-contract skill still references the nonexistent %q path", "tools/"+realPath)
	}
	if _, err := os.Stat(filepath.Join(repoRoot(t), realPath)); err != nil {
		t.Errorf("recipe path %q referenced by the skill does not exist on disk: %v", realPath, err)
	}

	// Source-derived cancel FSM, scoped to the cancel-description section
	// (so an incidental mention in the FSM diagram elsewhere does not
	// satisfy it): every non-empty CancelTarget for a contract status
	// must be documented there. The omitted case was deprecated → retired.
	cancelSection := sectionUnder(body, "Cancel a contract entirely")
	if cancelSection == "" {
		t.Fatal(`aiwf-contract skill has no "Cancel a contract entirely" section`)
	}
	contractStatuses := []string{
		entity.StatusProposed, entity.StatusAccepted,
		entity.StatusDeprecated, entity.StatusRetired, entity.StatusRejected,
	}
	for _, from := range contractStatuses {
		to := entity.CancelTarget(entity.KindContract, from)
		if to == "" {
			continue
		}
		if !strings.Contains(cancelSection, to) {
			t.Errorf("cancel section omits cancel target %q (from %q)", to, from)
		}
	}
	// The specific omission this AC fixes: the deprecated-source cancel case.
	if !strings.Contains(cancelSection, "deprecated") {
		t.Error("cancel section omits the deprecated → retired cancel case")
	}
}

// TestAiwfAuthorizeSkill_ProvenanceDocLinkResolves pins AC-4: the
// aiwf-authorize provenance-model doc-link resolves to a real file.
func TestAiwfAuthorizeSkill_ProvenanceDocLinkResolves(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, aiwfAuthorizeSkillPath)
	re := regexp.MustCompile(`\]\((\.\.[^)]*provenance-model\.md)\)`)
	m := re.FindStringSubmatch(body)
	if m == nil {
		t.Fatal("aiwf-authorize skill has no relative provenance-model doc-link")
	}
	skillDir := filepath.Dir(filepath.Join(repoRoot(t), aiwfAuthorizeSkillPath))
	target := filepath.Clean(filepath.Join(skillDir, m[1]))
	if _, err := os.Stat(target); err != nil {
		t.Errorf("provenance-model doc-link %q resolves to %q, which does not exist: %v", m[1], target, err)
	}
}

// TestAiwfAddSkill_ExampleSelfConsistentAndSectionCites pins AC-5: the
// aiwf-add typo example uses two distinct ids and cites doc sections, not
// pinned line numbers.
func TestAiwfAddSkill_ExampleSelfConsistentAndSectionCites(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, aiwfAddSkillPath)

	typo := regexp.MustCompile("A typo \\(`(M-\\d+)` for `(M-\\d+)`\\)")
	if m := typo.FindStringSubmatch(body); m == nil {
		t.Error("aiwf-add skill has no recognizable typo example of the form \"A typo (`M-NNN` for `M-NNN`)\"")
	} else if m[1] == m[2] {
		t.Errorf("aiwf-add typo example uses the same id twice (%q for %q); it must be self-contradictory to make sense", m[1], m[2])
	}

	pinned := regexp.MustCompile(`docs/[^\s` + "`" + `)]+\.md:\d+`)
	if loc := pinned.FindString(body); loc != "" {
		t.Errorf("aiwf-add skill cites a doc by fragile pinned line number (%q); cite the section name instead", loc)
	}
}
