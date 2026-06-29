package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// m0209GuidanceFixturePath is the canonical authoring location for the
// per-turn LLM guidance fragment (ADR-0018). `.claude/aiwf-guidance.md` in a
// consumer repo is materialized from these embedded bytes by `aiwf init` /
// `aiwf update`, so AC-1's content claims are asserted against the source,
// never the gitignored rendered artifact.
const m0209GuidanceFixturePath = "internal/skills/embedded-guidance/aiwf-guidance.md"

// gateDisciplineBullet returns the "Gate discipline survives compaction"
// bullet block from CLAUDE.md's `## Working with the user` section — from the
// bolded lead-in up to the next top-level `- **` bullet (or the section end).
//
// Scoping to the bullet (rather than grepping the whole file) is required by
// CLAUDE.md *Testing* §"Substring assertions are not structural assertions":
// the generalized-gate language must live in the gate-discipline rule itself,
// not float anywhere in a 600-line file.
func gateDisciplineBullet(t *testing.T, claudeMd string) string {
	t.Helper()
	section := extractMarkdownSection(claudeMd, 2, "Working with the user")
	if section == "" {
		t.Fatal("CLAUDE.md must have a `## Working with the user` section carrying the gate-discipline rule")
	}
	const lead = "**Gate discipline survives compaction.**"
	start := strings.Index(section, lead)
	if start < 0 {
		t.Fatalf("`## Working with the user` must contain the %q bullet", lead)
	}
	rest := section[start+len(lead):]
	// The bullet ends at the next top-level list item.
	if end := strings.Index(rest, "\n- **"); end >= 0 {
		return lead + rest[:end]
	}
	return lead + rest
}

// TestM0209_AC1_GeneralizedGateInClaudeMd asserts M-0209/AC-1 for CLAUDE.md:
// the declared-sequence gate is documented as a *general* capability for any
// local, reversible mutation sequence (with the bright line that excludes
// outward/irreversible and timing-bearing actions), and the false "wf-patch
// only; milestone and epic wraps keep per-action gates" scoping is gone.
func TestM0209_AC1_GeneralizedGateInClaudeMd(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "CLAUDE.md"))
	if err != nil {
		t.Fatalf("reading CLAUDE.md: %v", err)
	}
	bullet := gateDisciplineBullet(t, string(data))
	lower := strings.ToLower(bullet)

	// The generalized capability and its bright line must be present.
	wantPresent := []string{
		"declared-sequence gate",
		"local, reversible",
		"outward",
		"timing-bearing",
	}
	for _, w := range wantPresent {
		if !strings.Contains(lower, strings.ToLower(w)) {
			t.Errorf("AC-1: gate-discipline bullet must document the generalized gate — missing %q", w)
		}
	}

	// The false restrictive scoping must be gone. G-0295: CLAUDE.md asserted
	// the wraps "keep per-action gates," which was untrue.
	wantAbsent := []string{
		"Scope is wf-patch only",
		"milestone and epic wraps keep per-action gates",
	}
	for _, w := range wantAbsent {
		if strings.Contains(bullet, w) {
			t.Errorf("AC-1: the false %q scoping must be rewritten (G-0295)", w)
		}
	}
}

// TestM0209_AC1_GeneralizedGateInGuidance asserts M-0209/AC-1 for the embedded
// guidance source: the mutating-action gate rule names the declared-sequence
// gate as the bounded exception to "don't bundle," so a consumer's
// materialized guidance carries the same generalized rule CLAUDE.md does.
func TestM0209_AC1_GeneralizedGateInGuidance(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), m0209GuidanceFixturePath))
	if err != nil {
		t.Fatalf("reading %s: %v", m0209GuidanceFixturePath, err)
	}
	// Scope to the "Each mutating action is its own approval gate" bullet — the
	// rule the declared-sequence gate is an exception to — rather than grepping
	// the whole file (CLAUDE.md *Substring assertions are not structural
	// assertions*).
	const lead = "**Each mutating action is its own approval gate.**"
	body := string(data)
	start := strings.Index(body, lead)
	if start < 0 {
		t.Fatalf("AC-1: guidance must contain the %q bullet", lead)
	}
	bullet := body[start:]
	if end := strings.Index(bullet[len(lead):], "\n- **"); end >= 0 {
		bullet = bullet[:len(lead)+end]
	}
	lower := strings.ToLower(bullet)
	for _, w := range []string{"declared-sequence gate", "local, reversible"} {
		if !strings.Contains(lower, strings.ToLower(w)) {
			t.Errorf("AC-1: the guidance gate bullet must document the generalized declared-sequence gate — missing %q", w)
		}
	}
}

// m0209ReleaseFixturePath is the canonical authoring location for the
// `aiwfx-release` ritual body — the embedded snapshot the binary ships.
const m0209ReleaseFixturePath = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-release/SKILL.md"

// TestM0209_AC2_ReleaseSplitsPushGates asserts M-0209/AC-2: the `aiwfx-release`
// ritual no longer bundles the two origin pushes under one approval. Push is an
// outward/irreversible action, so per the declared-sequence bright line each
// push stands as its own gate.
//
// Structural, not a flat grep (CLAUDE.md *Substring assertions are not
// structural assertions*): the two push commands must live under *different*
// `### ` Workflow steps, and the old bundled prompt must be gone.
func TestM0209_AC2_ReleaseSplitsPushGates(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), m0209ReleaseFixturePath))
	if err != nil {
		t.Fatalf("reading %s: %v", m0209ReleaseFixturePath, err)
	}
	body := string(data)

	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		t.Fatal("AC-2: aiwfx-release must have a `## Workflow` section")
	}

	// Both pushes must still be documented.
	mainIdx := strings.Index(workflow, "git push origin main")
	tagIdx := strings.Index(workflow, "git push origin vX.Y.Z")
	if mainIdx < 0 || tagIdx < 0 {
		t.Fatalf("AC-2: Workflow must document both pushes (commit push found=%t, tag push found=%t)", mainIdx >= 0, tagIdx >= 0)
	}

	// The two pushes must be separated by a `### ` step heading — i.e. they
	// live in distinct gated steps, not one shared code block.
	lo, hi := mainIdx, tagIdx
	if lo > hi {
		lo, hi = hi, lo
	}
	if !strings.Contains(workflow[lo:hi], "\n### ") {
		t.Error("AC-2: the commit push and the tag push must be in separate `### ` gated steps (no `### ` heading separates them — they are still bundled in one step)")
	}

	// At least two push-gate step headings must exist.
	pushGateHeadings := 0
	for _, line := range strings.Split(workflow, "\n") {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		if strings.Contains(strings.ToLower(line), "push gate") {
			pushGateHeadings++
		}
	}
	if pushGateHeadings < 2 {
		t.Errorf("AC-2: Workflow must have two separate push-gate steps (found %d `### …Push gate…` headings)", pushGateHeadings)
	}

	// The old bundled prompt must be gone.
	if strings.Contains(body, "Push the commit and the tag to origin?") {
		t.Error("AC-2: the bundled `Push the commit and the tag to origin?` prompt must be split into two separate per-push confirmations")
	}
}

// m0209WrapMilestoneFixturePath is the canonical authoring location for the
// `aiwfx-wrap-milestone` ritual body.
const m0209WrapMilestoneFixturePath = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md"

// findWorkflowSubsection returns the `### ` subsection of `## Workflow` whose
// heading text (lowercased) contains needle, or "" if none matches.
func findWorkflowSubsection(body, needle string) string {
	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		return ""
	}
	for _, line := range strings.Split(workflow, "\n") {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		text := strings.TrimPrefix(line, "### ")
		if strings.Contains(strings.ToLower(text), needle) {
			return extractMarkdownSection(body, 3, text)
		}
	}
	return ""
}

// TestM0209_AC3_WrapMilestoneDeclaredSequenceGate asserts M-0209/AC-3: the
// `aiwfx-wrap-milestone` ritual governs its terminal local sequence
// (promote-done + local merge + local cleanup) under one declared-sequence
// gate, with push excluded as its own outward gate.
//
// Structural per CLAUDE.md *Substring assertions are not structural
// assertions*: the claims are scoped to the gate subsection, not grepped over
// the whole file.
func TestM0209_AC3_WrapMilestoneDeclaredSequenceGate(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), m0209WrapMilestoneFixturePath))
	if err != nil {
		t.Fatalf("reading %s: %v", m0209WrapMilestoneFixturePath, err)
	}
	body := string(data)

	gate := findWorkflowSubsection(body, "declared-sequence")
	if gate == "" {
		t.Fatal("AC-3: `## Workflow` must contain a `### …declared-sequence…` gate governing the terminal local sequence")
	}

	// The gate's enumerated local sequence: promote-done, the local merge,
	// and local cleanup.
	if !strings.Contains(gate, "aiwf promote M-NNNN done") {
		t.Error("AC-3: the declared-sequence gate must enumerate the promote-done step (`aiwf promote M-NNNN done`) — promote is no longer a separate ungated step")
	}
	if !strings.Contains(gate, "git merge --no-ff") {
		t.Error("AC-3: the declared-sequence gate must include the local merge into the epic branch")
	}
	if !strings.Contains(gate, "git branch -d") {
		t.Error("AC-3: the declared-sequence gate must include local cleanup (delete the local milestone branch)")
	}

	// The local merge must precede promote-done in the gate: promote lands last
	// so a delegated milestone's authorize scope stays live for the merge commit
	// (the G-0119 invariant, applied at milestone scope).
	if mi, pi := strings.Index(gate, "git merge --no-ff"), strings.Index(gate, "aiwf promote M-NNNN done"); mi >= 0 && pi >= 0 && mi > pi {
		t.Error("AC-3: the local merge must appear before `aiwf promote M-NNNN done` in the gate (promote lands last, per G-0119)")
	}

	// Push is excluded from the gate (outward action, its own gate).
	lower := strings.ToLower(gate)
	if !strings.Contains(lower, "push") || !strings.Contains(lower, "exclud") {
		t.Error("AC-3: the declared-sequence gate must state that push is excluded (outward actions stand as their own gate)")
	}
	if strings.Contains(gate, "git push") {
		t.Error("AC-3: the declared-sequence gate must not contain a `git push` command — push is a separate outward gate")
	}

	// A separate push-gate step still exists.
	if findWorkflowSubsection(body, "push gate") == "" {
		t.Error("AC-3: a separate `### …Push gate…` step must remain (push is its own outward gate)")
	}

	// The old ungated standalone promote step must be gone.
	if findWorkflowSubsection(body, "promote the milestone status") != "" {
		t.Error("AC-3: the ungated `### Promote the milestone status` step must be folded into the declared-sequence gate")
	}
}

// m0209WrapEpicFixturePath is the canonical authoring location for the
// `aiwfx-wrap-epic` ritual body.
const m0209WrapEpicFixturePath = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-epic/SKILL.md"

// TestM0209_AC4_WrapEpicDeclaredSequenceGate asserts M-0209/AC-4: `aiwfx-wrap-epic`
// runs its terminal local sequence (merge → wrap-artefact commit → promote-done)
// under one declared-sequence gate — replacing the separate merge-gate +
// commit-gate + ungated-promote — with push excluded, and the origin-branch
// deletes split into per-action outward gates rather than batch-approved.
//
// Structural per CLAUDE.md *Substring assertions are not structural assertions*:
// claims are scoped to the gate / cleanup subsections, not grepped file-wide.
func TestM0209_AC4_WrapEpicDeclaredSequenceGate(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), m0209WrapEpicFixturePath))
	if err != nil {
		t.Fatalf("reading %s: %v", m0209WrapEpicFixturePath, err)
	}
	body := string(data)

	gate := findWorkflowSubsection(body, "declared-sequence")
	if gate == "" {
		t.Fatal("AC-4: wrap-epic `## Workflow` must contain a `### …declared-sequence…` gate over the terminal local sequence")
	}
	lower := strings.ToLower(gate)

	// The gate enumerates the three local actions: merge, the wrap-artefact /
	// CHANGELOG commit, and promote-done.
	for _, w := range []string{"merge", "promote"} {
		if !strings.Contains(lower, w) {
			t.Errorf("AC-4: the declared-sequence gate must enumerate %q", w)
		}
	}
	if !strings.Contains(lower, "wrap") && !strings.Contains(lower, "changelog") {
		t.Error("AC-4: the declared-sequence gate must enumerate the wrap-artefact / CHANGELOG commit")
	}

	// Push is excluded (outward, its own gate); no push command inside the gate.
	if !strings.Contains(lower, "push") || !strings.Contains(lower, "exclud") {
		t.Error("AC-4: the declared-sequence gate must state that push is excluded (outward, its own gate)")
	}
	if strings.Contains(gate, "git push") {
		t.Error("AC-4: the declared-sequence gate must not contain a `git push` command")
	}

	// A separate push gate remains.
	if findWorkflowSubsection(body, "push gate") == "" {
		t.Error("AC-4: a separate `### …Push gate…` step must remain")
	}

	// The old separate merge-gate and commit-gate steps are folded into the one
	// declared-sequence gate.
	if findWorkflowSubsection(body, "merge gate") != "" {
		t.Error("AC-4: the separate `🛑 Merge gate` step must be folded into the declared-sequence gate")
	}
	if findWorkflowSubsection(body, "commit gate") != "" {
		t.Error("AC-4: the separate `🛑 Commit gate` step must be folded into the declared-sequence gate")
	}

	// Origin-branch deletes are split into per-action outward gates, not batched.
	cleanup := findWorkflowSubsection(body, "cleanup")
	if cleanup == "" {
		t.Fatal("AC-4: wrap-epic must retain a branch-cleanup step")
	}
	if strings.Contains(cleanup, "batch approval for the full list") {
		t.Error("AC-4: origin-branch deletes must be split into per-action gates, not batch-approved")
	}
	if !strings.Contains(strings.ToLower(cleanup), "own gate") {
		t.Error("AC-4: each origin-branch delete must be its own outward gate")
	}
}
