package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// deployerCardFixturePath is the canonical authoring location for the
// deployer role-agent card (ADR-0014). `.claude/agents/deployer.md` in a
// consumer repo is materialized from these embedded bytes by `aiwf init` /
// `aiwf update`, so the wording claims are asserted against the source,
// never the gitignored render.
const deployerCardFixturePath = "internal/skills/embedded-rituals/plugins/aiwf-extensions/agents/deployer.md"

// TestDeployerCard_FrontmatterDescriptionNamesReleaseTriggers pins the
// independent fix G-0362 flagged alongside ADR-0028: the deployer card's own
// frontmatter `description:` previously named no release-trigger phrasing at
// all, unlike aiwfx-release's description (G-0361). An assistant deciding
// which subagent to dispatch reads agent-card descriptions directly (the
// "Available agent types" listing), not just skill descriptions — a
// description with no matching phrase never surfaces deployer as a
// candidate before any skill body is ever read.
func TestDeployerCard_FrontmatterDescriptionNamesReleaseTriggers(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), deployerCardFixturePath))
	if err != nil {
		t.Fatalf("reading %s: %v", deployerCardFixturePath, err)
	}
	body := string(data)

	desc := frontmatterField(body, "description")
	if desc == "" {
		t.Fatal("deployer card frontmatter `description:` must be non-empty")
	}

	lower := strings.ToLower(desc)
	for _, p := range []string{"cut a release", "let's ship", "let's release", "make a new version", "tag a release", "publish"} {
		if !strings.Contains(lower, p) {
			t.Errorf("frontmatter `description:` must name the trigger phrase %q", p)
		}
	}
}

// TestDeployerCard_G0384_DoesNotRunPushItself pins G-0384's fix: the
// deployer card's Constraints section instructs handing each approved
// push command back to the orchestrating session to execute, rather
// than running `git push` from within the deployer subagent's own
// sandboxed tool context — which stalled on the network-write phase
// twice during a real release run.
func TestDeployerCard_G0384_DoesNotRunPushItself(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), deployerCardFixturePath))
	if err != nil {
		t.Fatalf("reading %s: %v", deployerCardFixturePath, err)
	}
	body := string(data)

	constraints := extractMarkdownSection(body, 2, "Constraints")
	if constraints == "" {
		t.Fatal("deployer card must have a `## Constraints` section")
	}
	if !strings.Contains(constraints, "Hand the exact approved command back to the orchestrating session") {
		t.Error("G-0384: `## Constraints` must instruct handing each approved push command back to the orchestrating session, not running git push itself")
	}
}
