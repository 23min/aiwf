package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// g0343GuidanceFixturePath is the canonical authoring location for the shipped
// per-turn guidance fragment (ADR-0018). `.claude/aiwf-guidance.md` in a
// consumer repo is materialized from these embedded bytes by `aiwf init` /
// `aiwf update`, so the one-decision rule's content claims are asserted against
// the source, never the gitignored render.
const g0343GuidanceFixturePath = "internal/skills/embedded-guidance/aiwf-guidance.md"

// oneDecisionBullet returns the "Decide one thing at a time" bullet from the
// shipped guidance — from its bolded lead-in up to the next top-level `- **`
// bullet (or the section end). Scoping to the bullet (rather than grepping the
// whole file) is required by CLAUDE.md *Substring assertions are not structural
// assertions*: the enriched content requirement must live in the one-decision
// rule itself, not float anywhere in the fragment.
func oneDecisionBullet(t *testing.T, body string) string {
	t.Helper()
	const lead = "**Decide one thing at a time.**"
	start := strings.Index(body, lead)
	if start < 0 {
		t.Fatalf("guidance must contain the %q bullet", lead)
	}
	rest := body[start+len(lead):]
	if end := strings.Index(rest, "\n- **"); end >= 0 {
		return lead + rest[:end]
	}
	// The bullet may be the last one before a `## ` section heading.
	if end := strings.Index(rest, "\n## "); end >= 0 {
		return lead + rest[:end]
	}
	return lead + rest
}

// TestG0343_OneDecisionGuidanceCarriesFullContent pins the enrichment the
// G-0343 patch made to the shipped one-decision-at-a-time rule. Before the
// patch the shipped bullet was thinner than the kernel's actual rule — "one at
// a time with context and a recommendation" — dropping the pros/cons, the
// risks, and the argued lean a human needs in order to decide. The patch
// restores the full content requirement and adds a content-over-container
// clause so a terse `AskUserQuestion` card cannot be rationalized as compliant.
//
// Structural per CLAUDE.md *Substring assertions are not structural
// assertions*: every claim is scoped to the one-decision bullet, not grepped
// over the whole fragment.
func TestG0343_OneDecisionGuidanceCarriesFullContent(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), g0343GuidanceFixturePath))
	if err != nil {
		t.Fatalf("reading %s: %v", g0343GuidanceFixturePath, err)
	}
	bullet := oneDecisionBullet(t, string(data))
	lower := strings.ToLower(bullet)

	// The M0211 `one-decision-at-a-time` anchor fragment must survive the edit.
	if !strings.Contains(lower, "one thing at a time") {
		t.Error("the M0211 `one thing at a time` anchor fragment must stay in the bullet")
	}

	// Full content requirement: context, pros/cons, risks, an argued lean, a
	// numbered pick-list — the reasoning the thin shipped bullet had dropped.
	for _, w := range []string{"context", "pros/cons", "risk", "plain lean", "argument", "numbered"} {
		if !strings.Contains(lower, w) {
			t.Errorf("the one-decision bullet must carry the full content requirement — missing %q", w)
		}
	}

	// Content-over-container clause: the reasoning is the deliverable; a card's
	// terseness must not strip it, and decisions are never batched into one card.
	if !strings.Contains(lower, "container") {
		t.Error("the bullet must state the reasoning is the deliverable and the container serves it")
	}
	if !strings.Contains(bullet, "AskUserQuestion") {
		t.Error("the content-over-container clause must name the `AskUserQuestion` card as a container that must still carry the full content")
	}
	if !strings.Contains(lower, "never batch") {
		t.Error("the clause must forbid batching several decisions into one card")
	}
}

// TestOneDecisionGuidanceRequiresPlainLanguage pins the register requirement
// on the shipped one-decision rule. The rule specified the decision's
// *structure* — context, pros/cons, risks, an argued lean, a pick-list — and
// said nothing about the language it is written in, so options could arrive
// carrying the vocabulary of whatever analysis produced them. A decision the
// human has to translate before answering is not one they can make.
//
// Scoped to the one-decision bullet per CLAUDE.md *Substring assertions are
// not structural assertions*, and matched on distinctive terms rather than
// whole clauses so a reword that preserves the rule does not turn this red.
func TestOneDecisionGuidanceRequiresPlainLanguage(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), g0343GuidanceFixturePath))
	if err != nil {
		t.Fatalf("reading %s: %v", g0343GuidanceFixturePath, err)
	}
	bullet := oneDecisionBullet(t, string(data))
	lower := strings.ToLower(bullet)

	if !strings.Contains(lower, "plainest language") {
		t.Error("the one-decision bullet must require the plainest language that still carries the trade")
	}
	// The carve-out is what keeps the rule usable: a specialist term is
	// allowed exactly where the decision is about that term.
	if !strings.Contains(lower, "subject of the decision") {
		t.Error("the register clause must permit a specialist term where it is the subject of the decision")
	}
	// The container half: a card's question is the field that renders as an
	// unreadable block when it carries the whole decision.
	if !strings.Contains(lower, "one short line") {
		t.Error("the register clause must hold a card's question to one short line, with detail in its options")
	}
}
