package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// aiwfArchiveSkillPath is the on-disk canonical location of the
// per-verb skill ADR-0004 calls for (per ADR-0006 "per-verb skill
// default"). Kernel-embedded skills under `internal/skills/embedded/`
// ship via `//go:embed embedded` in `internal/skills/skills.go`; the
// aiwfx-* rituals plugin skills ship via the sibling go:embed of
// `embedded-rituals/` (G-0182 consolidated their authoring onto the
// same embedded snapshot, eliminating the prior duplicate fixture
// surface under `internal/policies/testdata/`).
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	if rationale, ok := skillCoverageAllowlist["archive"]; ok {
		t.Errorf("AC-4: skillCoverageAllowlist still carries an entry for `archive` (rationale: %q); remove it now that internal/skills/embedded/aiwf-archive/SKILL.md ships",
			rationale)
	}
}
