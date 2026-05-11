package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// aiwfArchiveSkillPath is the on-disk canonical location of the
// per-verb skill ADR-0004 calls for (per ADR-0006 "per-verb skill
// default"). Unlike the aiwfx-* rituals plugin skills that ship from
// an upstream repo and live as fixtures under
// `internal/policies/testdata/` until copied at wrap, the kernel's
// own embedded skills ship from this path directly via
// `//go:embed embedded` in `internal/skills/skills.go`.
const aiwfArchiveSkillPath = "internal/skills/embedded/aiwf-archive/SKILL.md"

// loadAiwfArchiveSkill reads the on-disk SKILL.md relative to the
// repo root.
func loadAiwfArchiveSkill(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, aiwfArchiveSkillPath))
	if err != nil {
		t.Fatalf("loading %s: %v", aiwfArchiveSkillPath, err)
	}
	return string(data)
}

// TestAiwfArchive_AC3_Scaffolded asserts AC-3: the skill exists with
// frontmatter declaring `name: aiwf-archive` (matching the directory)
// and a non-empty `description:`. The skill_coverage policy
// (`PolicySkillCoverageMatchesVerbs`) also pins this invariant, but
// AC-3 wants a dedicated assertion so the AC's mechanical evidence
// is visible at the AC granularity.
func TestAiwfArchive_AC3_Scaffolded(t *testing.T) {
	body := loadAiwfArchiveSkill(t)

	name := frontmatterField(body, "name")
	if name != "aiwf-archive" {
		t.Errorf("AC-3: frontmatter `name:` must be `aiwf-archive` (got %q)", name)
	}

	desc := frontmatterField(body, "description")
	if desc == "" {
		t.Error("AC-3: frontmatter `description:` must be non-empty (host's match-scoring depends on it)")
	}
}

// TestAiwfArchive_AC3_DryRunVsApplySection pins the §"What to run"
// guidance ADR-0004 requires: when to use the dry-run default vs
// `--apply`, and the `--kind` example. The structural assertion is
// scoped to the named §; substring-greppimg across the whole file
// would pass even if the prose lived in the wrong section.
func TestAiwfArchive_AC3_DryRunVsApplySection(t *testing.T) {
	body := loadAiwfArchiveSkill(t)
	section := extractMarkdownSection(body, 2, "What to run")
	if section == "" {
		t.Fatal("AC-3: SKILL.md must have a `## What to run` section")
	}
	required := []string{
		"--apply",
		"--dry-run",
		"--kind",
	}
	for _, want := range required {
		if !strings.Contains(section, want) {
			t.Errorf("AC-3: §What to run must reference %q", want)
		}
	}
}

// TestAiwfArchive_AC3_NoReverseSection pins the no-reverse-sweep
// rule per ADR-0004 §"Reversal — what verb undoes archive?". The
// canonical pattern ("file a new entity that references the
// archived one") must appear in the dedicated reversal section so
// readers learn the rule from the skill, not the ADR alone.
func TestAiwfArchive_AC3_NoReverseSection(t *testing.T) {
	body := loadAiwfArchiveSkill(t)
	section := extractMarkdownSection(body, 2, "Reversal")
	if section == "" {
		t.Fatal("AC-3: SKILL.md must have a `## Reversal` section (no-reverse-sweep rule)")
	}
	lower := strings.ToLower(section)
	// The section must state plainly that there is no reverse verb
	// — the reader should not have to infer it from absence.
	if !strings.Contains(lower, "no") {
		t.Error("AC-3: §Reversal must state the no-reverse rule plainly")
	}
	// The canonical pattern: file a new entity referencing the
	// archived one. Use "new entity" as the structural anchor.
	if !strings.Contains(lower, "new entity") {
		t.Error("AC-3: §Reversal must name the canonical pattern (file a new entity that references the archived one)")
	}
}

// TestAiwfArchive_AC3_ThresholdKnobSection pins the M-0088 knob
// itself: the skill must explain `archive.sweep_threshold` and how
// to set it in `aiwf.yaml`. Per ADR-0004 §"Drift control" layer (2).
func TestAiwfArchive_AC3_ThresholdKnobSection(t *testing.T) {
	body := loadAiwfArchiveSkill(t)
	// Accept either "Drift control" or "Threshold" headings — the
	// concept is one; the heading text is the author's call. Try
	// the ADR's terminology first, fall back to the alternative.
	section := extractMarkdownSection(body, 2, "Drift control")
	if section == "" {
		section = extractMarkdownSection(body, 2, "Threshold")
	}
	if section == "" {
		t.Fatal("AC-3: SKILL.md must have a `## Drift control` (or `## Threshold`) section explaining the knob")
	}
	required := []string{
		"archive.sweep_threshold",
		"aiwf.yaml",
	}
	for _, want := range required {
		if !strings.Contains(section, want) {
			t.Errorf("AC-3: §Drift control / §Threshold must reference %q", want)
		}
	}
}

// TestAiwfArchive_AC3_MergeEdgeSection pins the merge-edge-case
// guidance per ADR-0004 §Consequences §Negative ("rename+modify
// conflict"). The skill body covers it so AI assistants reach for
// the right resolution without re-reading the ADR.
func TestAiwfArchive_AC3_MergeEdgeSection(t *testing.T) {
	body := loadAiwfArchiveSkill(t)
	section := extractMarkdownSection(body, 2, "Merge")
	if section == "" {
		t.Fatal("AC-3: SKILL.md must have a `## Merge ...` section covering the rename+modify case")
	}
	lower := strings.ToLower(section)
	// The conflict shape: a branch archives an entity while another
	// edits it in place. The section must name the shape and the
	// resolution.
	for _, kw := range []string{"rename", "modify", "branch"} {
		if !strings.Contains(lower, kw) {
			t.Errorf("AC-3: §Merge must reference %q", kw)
		}
	}
}

// TestAiwfArchive_AC3_PerKindStorageSection pins the per-kind
// storage layout per ADR-0004 §"Storage — per-kind layout". The
// SKILL.md replicates the table — every kind named in the ADR must
// appear so the AI reader sees the active→archive path mapping.
func TestAiwfArchive_AC3_PerKindStorageSection(t *testing.T) {
	body := loadAiwfArchiveSkill(t)
	section := extractMarkdownSection(body, 2, "Per-kind storage")
	if section == "" {
		section = extractMarkdownSection(body, 2, "Storage")
	}
	if section == "" {
		t.Fatal("AC-3: SKILL.md must have a `## Per-kind storage` (or `## Storage`) section")
	}
	// Every kind from the ADR's table (epic, milestone, contract,
	// gap, decision, adr). The kinds appear in the active→archive
	// path strings, which is the structurally relevant location.
	lower := strings.ToLower(section)
	for _, kind := range []string{"epic", "milestone", "contract", "gap", "decision", "adr"} {
		if !strings.Contains(lower, kind) {
			t.Errorf("AC-3: §Per-kind storage / §Storage must name kind %q", kind)
		}
	}
}

// TestAiwfArchive_AC3_CitesADR0004 pins the citation invariant:
// every embedded skill that flows from an ADR cites the ADR by id
// (per CLAUDE.md §"Kernel functionality must be AI-discoverable" —
// the reader who lands on the skill must be able to follow the
// thread back to the ratified decision).
func TestAiwfArchive_AC3_CitesADR0004(t *testing.T) {
	body := loadAiwfArchiveSkill(t)
	if !strings.Contains(body, "ADR-0004") {
		t.Error("AC-3: SKILL.md must cite ADR-0004 by id")
	}
}

// TestAiwfArchive_AC6_ClaudeMdNamesArchiveConvention pins M-0088/AC-6:
// CLAUDE.md's "What aiwf commits to" section gains a numbered item
// naming the archive convention and pointing at ADR-0004. The
// assertion is structural per CLAUDE.md §"Substring assertions are
// not structural assertions" — the test extracts the named section
// and looks inside it, not flat over the whole file. A literal
// "ADR-0004" elsewhere in CLAUDE.md (e.g. a future reference link)
// must not silently satisfy the AC.
//
// "What aiwf commits to" is a numbered list. Pre-M-0088 it carries
// 9 items (the kernel's load-bearing properties); post-M-0088 the
// 10th names the archive convention. The test counts the items as
// the structural anchor: the convention prose must live inside the
// section, and the section must grow to ≥10 items.
func TestAiwfArchive_AC6_ClaudeMdNamesArchiveConvention(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("loading CLAUDE.md: %v", err)
	}
	body := string(data)
	section := extractMarkdownSection(body, 2, "What aiwf commits to")
	if section == "" {
		t.Fatal("AC-6: CLAUDE.md must have a `## What aiwf commits to` section")
	}

	// Count numbered items. The section is a numbered list of the
	// form "1. **...** ...". Match start-of-line "N. " patterns.
	items := numberedListItems(section)
	if len(items) < 10 {
		t.Fatalf("AC-6: §What aiwf commits to must carry ≥10 numbered items (the M-0088 amendment adds item 10); got %d", len(items))
	}

	// The archive-convention item names the verb and cites ADR-0004
	// by id. Locate it by archive-convention keywords; the new item
	// is the only one that should contain "archive" with "ADR-0004".
	var found bool
	for _, item := range items {
		lower := strings.ToLower(item)
		if !strings.Contains(lower, "archive") {
			continue
		}
		if !strings.Contains(item, "ADR-0004") {
			continue
		}
		if !strings.Contains(lower, "aiwf archive") {
			continue
		}
		found = true
		break
	}
	if !found {
		t.Errorf("AC-6: §What aiwf commits to must contain a numbered item that names `aiwf archive`, references ADR-0004 by id, and uses the word `archive`; the section's numbered items were:\n%s",
			strings.Join(items, "\n---\n"))
	}
}

// numberedListItems splits a markdown section's prose into its
// numbered-list items. Each item starts at a "N. " line at column 0
// and runs until the next "N. " line or end-of-section. Blank lines
// between items belong to the preceding item.
func numberedListItems(section string) []string {
	var items []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			items = append(items, strings.TrimRight(cur.String(), "\n"))
			cur.Reset()
		}
	}
	for _, line := range strings.Split(section, "\n") {
		if isNumberedListStart(line) {
			flush()
		}
		cur.WriteString(line)
		cur.WriteByte('\n')
	}
	flush()
	// Drop any "items" that don't actually start with "N. " (e.g.,
	// the leading paragraph before the list).
	out := items[:0]
	for _, item := range items {
		if isNumberedListStart(item) {
			out = append(out, item)
		}
	}
	return out
}

// isNumberedListStart reports whether a line begins with a markdown
// numbered-list marker like "1. " (digits + dot + space, at column 0).
func isNumberedListStart(line string) bool {
	i := 0
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	if i == 0 || i >= len(line) {
		return false
	}
	if line[i] != '.' {
		return false
	}
	if i+1 >= len(line) || line[i+1] != ' ' {
		return false
	}
	return true
}

// TestNumberedListItems_BranchCoverage exercises every reachable
// branch of numberedListItems and isNumberedListStart against
// synthetic inputs. The helper is only ever called from
// TestAiwfArchive_AC6_ClaudeMdNamesArchiveConvention today; the
// branch-coverage hard rule applies even to test-package helpers
// (precedent: TestFrontmatterField_BranchCoverage in
// aiwfx_whiteboard_test.go).
func TestNumberedListItems_BranchCoverage(t *testing.T) {
	cases := []struct {
		name      string
		section   string
		wantCount int
	}{
		{"empty section", "", 0},
		{"single item", "1. **One** ...\n", 1},
		{"two items", "1. **One**.\n2. **Two**.\n", 2},
		{"leading paragraph then list", "Some paragraph.\n\n1. One.\n2. Two.\n", 2},
		{"multi-line item", "1. **One** continues\n   on this line\n2. **Two**.\n", 2},
		{"no list at all", "Just prose.\nMore prose.\n", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := numberedListItems(tc.section)
			if len(got) != tc.wantCount {
				t.Errorf("got %d items, want %d; items: %q", len(got), tc.wantCount, got)
			}
		})
	}

	startCases := []struct {
		name string
		line string
		want bool
	}{
		{"empty line", "", false},
		{"plain prose", "hello world", false},
		{"no digit prefix", ". no digits", false},
		{"digits but no dot", "12 nope", false},
		{"digits + dot + no space", "12.text", false},
		{"digits + dot at end of string", "12.", false},
		{"valid 1. start", "1. item", true},
		{"valid 10. start", "10. item", true},
		{"indented number", "   1. nope", false},
	}
	for _, tc := range startCases {
		t.Run("isStart/"+tc.name, func(t *testing.T) {
			if got := isNumberedListStart(tc.line); got != tc.want {
				t.Errorf("isNumberedListStart(%q) = %v, want %v", tc.line, got, tc.want)
			}
		})
	}
}

// TestAiwfArchive_AC5_SkillCoveragePolicyClean is the AC-5 seam pin
// in PolicySkillCoverageMatchesVerbs: with AC-3 (skill exists) and
// AC-4 (allowlist entry removed) in place, the policy must be silent
// for the `aiwf-archive` skill specifically — its frontmatter
// validates, every backticked `aiwf <verb>` mention resolves, and
// `archive` does not show up as an uncovered verb. The whole-policy
// runPolicy(t, PolicySkillCoverageMatchesVerbs) elsewhere in this
// package already covers the global invariant; this test scopes the
// assertion to AC-5's named surface so a future drift specific to
// the archive skill surfaces at this AC.
func TestAiwfArchive_AC5_SkillCoveragePolicyClean(t *testing.T) {
	root := repoRoot(t)
	violations, err := PolicySkillCoverageMatchesVerbs(root)
	if err != nil {
		t.Fatalf("PolicySkillCoverageMatchesVerbs error: %v", err)
	}
	for _, v := range violations {
		// Any violation naming the archive skill or the verb is the
		// AC-5 failure mode.
		if strings.Contains(v.File, "aiwf-archive") ||
			strings.Contains(v.Detail, "archive") {
			t.Errorf("AC-5: skill-coverage policy fires on archive surface: %s — %s", v.File, v.Detail)
		}
	}
}

// TestAiwfArchive_AC4_AllowlistEntryRemoved pins that the placeholder
// allowlist entry M-0085 added (`"archive": "embedded skill lands in
// M-0088 ..."`) is removed now that the per-verb skill exists. The
// allowlist's purpose is making each *intentional* absence visible —
// a stale entry pointing at a skill that does ship would mislead
// every reviewer who reads the allowlist trying to understand why
// a verb skips skill coverage.
//
// Per M-074 / ADR-0006 the per-verb skill is the default for
// mutating verbs that carry decision logic, and `aiwf archive` is
// exactly that shape. The PolicySkillCoverageMatchesVerbs check
// (run elsewhere in this package) already enforces "every verb is
// either covered or allowlisted"; this test is the dedicated AC-4
// drift-check so a future regression that re-adds the entry is
// caught explicitly, not only via the policy's coverage walk.
func TestAiwfArchive_AC4_AllowlistEntryRemoved(t *testing.T) {
	if rationale, ok := skillCoverageAllowlist["archive"]; ok {
		t.Errorf("AC-4: skillCoverageAllowlist still carries an entry for `archive` (rationale: %q); remove it now that internal/skills/embedded/aiwf-archive/SKILL.md ships",
			rationale)
	}
}
