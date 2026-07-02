package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// builderCardFixturePath is the canonical authoring location for the builder
// role-agent card (ADR-0014). `.claude/agents/builder.md` in a consumer repo is
// materialized from these embedded bytes by `aiwf init` / `aiwf update`, so the
// wording claims are asserted against the source, never the gitignored render.
const builderCardFixturePath = "internal/skills/embedded-rituals/plugins/aiwf-extensions/agents/builder.md"

// builderCardDescription returns the value of the frontmatter `description:`
// field (the `key: value` between the leading `---` fences). The builder card
// uses only single-line frontmatter values, which is all this needs to handle.
func builderCardDescription(t *testing.T, body string) string {
	t.Helper()
	if !strings.HasPrefix(body, "---\n") {
		t.Fatal("builder card must open with a `---` frontmatter fence")
	}
	rest := body[len("---\n"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		t.Fatal("builder card frontmatter must close with a `---` fence")
	}
	for _, line := range strings.Split(rest[:end], "\n") {
		if strings.HasPrefix(line, "description:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
	}
	t.Fatal("builder card frontmatter must have a `description` field")
	return ""
}

// builderCardIntro returns the prose between the `# Builder` H1 and the first
// `## ` section — the card's opening statement of what the builder is.
func builderCardIntro(t *testing.T, body string) string {
	t.Helper()
	const h1 = "\n# Builder\n"
	start := strings.Index(body, h1)
	if start < 0 {
		t.Fatal("builder card must have a `# Builder` heading")
	}
	rest := body[start+len(h1):]
	if end := strings.Index(rest, "\n## "); end >= 0 {
		return rest[:end]
	}
	return rest
}

// TestG0342_BuilderConditionsTestFirstOnTDDFlag pins the reconciliation the
// G-0342 patch made: the builder card must present test-first *ordering* as
// opt-in per the milestone's `tdd:` flag, while the coverage obligation stays
// unconditional. Before the patch the card asserted test-first as identity
// ("You follow TDD", "via TDD", "Write tests first") regardless of the flag,
// contradicting the kernel's opt-in-per-milestone model (CLAUDE.md commit #8;
// the `acs-tdd-audit` rule fires only when `tdd: required`).
//
// Structural per CLAUDE.md *Substring assertions are not structural
// assertions*: each claim is scoped to the section that must carry it — the
// frontmatter description, the `# Builder` intro, `## Responsibilities`, and
// `## Constraints` — not grepped file-wide.
func TestG0342_BuilderConditionsTestFirstOnTDDFlag(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), builderCardFixturePath))
	if err != nil {
		t.Fatalf("reading %s: %v", builderCardFixturePath, err)
	}
	body := string(data)

	// --- Frontmatter description: no TDD-as-identity framing. ---
	desc := builderCardDescription(t, body)
	if strings.Contains(desc, "via TDD") {
		t.Error("description must not frame the builder as working `via TDD` — test-first ordering is opt-in per milestone, not the agent's identity")
	}
	if !strings.Contains(desc, "tdd:") {
		t.Error("description must reference the `tdd:` flag so the card signals ordering is conditional")
	}

	// --- `# Builder` intro: ordering is opt-in; no blanket "You follow TDD". ---
	intro := builderCardIntro(t, body)
	if strings.Contains(intro, "You follow TDD") {
		t.Error("the intro must not state the unconditional `You follow TDD.` — ordering follows the milestone's `tdd:` flag")
	}
	if !strings.Contains(strings.ToLower(intro), "opt-in") || !strings.Contains(intro, "tdd:") {
		t.Error("the intro must state test-first ordering is opt-in per milestone (the `tdd:` flag)")
	}

	// --- Responsibilities: ordering conditioned on the flag; coverage unconditional. ---
	resp := extractMarkdownSection(body, 2, "Responsibilities")
	if resp == "" {
		t.Fatal("builder card must have a `## Responsibilities` section")
	}
	respLower := strings.ToLower(resp)
	for _, w := range []string{"tdd: required", "red → green → refactor", "advisory", "unconditional"} {
		if !strings.Contains(respLower, strings.ToLower(w)) {
			t.Errorf("Responsibilities must condition ordering on the flag and keep coverage unconditional — missing %q", w)
		}
	}

	// --- Constraints: the branch-coverage hard rule stays unconditional. ---
	// Guards against over-correcting the fix into gating coverage on the flag.
	cons := extractMarkdownSection(body, 2, "Constraints")
	if cons == "" {
		t.Fatal("builder card must have a `## Constraints` section")
	}
	if !strings.Contains(cons, "Branch-coverage hard rule") {
		t.Error("Constraints must retain the branch-coverage hard rule")
	}
	if !strings.Contains(cons, "Every reachable conditional branch") {
		t.Error("the branch-coverage hard rule must stay unconditional (every reachable conditional branch), not gated on the `tdd:` flag")
	}
	if coverIdx := strings.Index(cons, "Branch-coverage hard rule"); coverIdx >= 0 {
		ruleTail := cons[coverIdx:]
		if nextBullet := strings.Index(ruleTail, "\n- "); nextBullet >= 0 {
			ruleTail = ruleTail[:nextBullet]
		}
		if strings.Contains(ruleTail, "tdd:") {
			t.Error("the branch-coverage hard rule must not reference the `tdd:` flag — coverage is unconditional")
		}
	}
}
