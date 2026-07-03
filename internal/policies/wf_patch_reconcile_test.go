package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// wf_patch_reconcile_test.go pins the reconcile-first practice added
// to wf-patch's step 9 (the wrap gate's "Merge to mainline" bullet):
// before merging a patch branch to mainline, if mainline has
// advanced past the branch's fork point, mainline must be integrated
// into the patch branch first and the full local CI gate re-run
// there — never resolved on mainline itself, mid-merge, with the
// "gate green before merge" precondition passing vacuously against
// a tree that omits mainline's newer commits.

// wfPatchFixturePath is the canonical authoring location for the
// `wf-patch` skill body — the embedded ritual snapshot the aiwf
// binary ships. Per G-0182, AC content assertions read the embedded
// bytes directly rather than a duplicated fixture under
// internal/policies/testdata/.
const wfPatchFixturePath = "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-patch/SKILL.md"

// loadWfPatchFixture reads the fixture relative to repo root.
func loadWfPatchFixture(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, wfPatchFixturePath))
	if err != nil {
		t.Fatalf("loading %s: %v", wfPatchFixturePath, err)
	}
	return string(data)
}

// findWfPatchWrapGateStep returns step 9's numbered-list body inside
// `## Workflow` — the wrap-gate step that enumerates the merge to
// mainline, tracker closure, and cleanup. Unlike the wrap-epic and
// wrap-milestone skills (whose per-step bodies are `### ` headings),
// wf-patch's `## Workflow` section is a flat numbered list with no
// per-step subheadings, so the locator scans for the "9. " list-item
// prefix and stops at the next top-level "10. " item.
func findWfPatchWrapGateStep(body string) string {
	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		return ""
	}
	lines := strings.Split(workflow, "\n")
	start, end := -1, len(lines)
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case start == -1 && strings.HasPrefix(trimmed, "9. "):
			start = i
		case start != -1 && strings.HasPrefix(trimmed, "10. "):
			end = i
		}
		if start != -1 && end != len(lines) {
			break
		}
	}
	if start == -1 {
		return ""
	}
	return strings.Join(lines[start:end], "\n")
}

// TestWfPatch_ReconcileMainlineBeforeMerge asserts the reconcile-
// first content lives inside step 9's "Merge to mainline" bullet —
// not floating elsewhere in the skill — and that it orders the
// existing mechanism-neutral lead, the ancestor guard, and the
// integrate-then-re-gate instruction correctly. Per CLAUDE.md
// *Substring assertions are not structural assertions*, the section
// locator scopes every assertion to step 9 specifically, not the
// file as a whole.
func TestWfPatch_ReconcileMainlineBeforeMerge(t *testing.T) {
	t.Parallel()
	body := loadWfPatchFixture(t)

	step := findWfPatchWrapGateStep(body)
	if step == "" {
		t.Fatal("could not locate wf-patch's step 9 (wrap gate) inside `## Workflow`")
	}

	// The ancestor guard compares against *local* mainline, not the
	// remote-tracking ref: the G-0346 divergence was local `main`
	// advancing under a concurrent session, which `origin/main` would
	// not reflect. The remote-tracking ref appears only in the
	// fetch/fast-forward preamble that folds in the origin axis.
	wantGuard := "git merge-base --is-ancestor main <branch>"
	if !strings.Contains(step, wantGuard) {
		t.Errorf("step 9 must name the ancestor guard %q (local mainline, not origin/main)", wantGuard)
	}

	// The fetch/fast-forward preamble must precede the ancestor guard so
	// the local target reflects both divergence axes (local concurrent
	// commits, already present; and commits another clone pushed, folded
	// in via the fetch) before the check runs.
	fetchIdx := strings.Index(step, "git fetch")
	ffIdx := strings.Index(step, "--ff-only origin/main")
	guardIdx := strings.Index(step, wantGuard)
	if fetchIdx < 0 || ffIdx < 0 {
		t.Fatal("step 9 must document `git fetch` and fast-forwarding local main to origin/main before the ancestor guard")
	}
	if fetchIdx >= guardIdx || ffIdx >= guardIdx {
		t.Errorf("step 9 must run the fetch/fast-forward preamble BEFORE the ancestor guard (fetch=%d, ff=%d, guard=%d)", fetchIdx, ffIdx, guardIdx)
	}

	// The tracker-closure bullet must document the mechanical backstop:
	// the reachability check that refuses a --by-commit SHA not on HEAD.
	if !strings.Contains(step, "reachable from `HEAD`") {
		t.Error("step 9's tracker-closure bullet must note the mechanical guard (a --by-commit SHA must be reachable from `HEAD`)")
	}

	mergeIdx := strings.Index(step, "Merge to mainline.")
	integrateIdx := strings.Index(step, "integrate current mainline into the patch branch")
	gateIdx := strings.Index(step, "re-run the full local CI gate")
	if mergeIdx < 0 {
		t.Fatal("step 9 must retain the existing `Merge to mainline.` bullet lead")
	}
	if integrateIdx < 0 || gateIdx < 0 {
		t.Fatal("step 9 must document integrating mainline into the patch branch and re-running the full local CI gate")
	}
	// The mechanism-neutral "Merge to mainline." lead comes first
	// (unchanged framing), then the reconcile-first addition orders
	// integrate-mainline before re-run-the-gate.
	if mergeIdx >= integrateIdx || integrateIdx >= gateIdx {
		t.Errorf("step 9 must order the merge-to-mainline lead -> integrate mainline -> re-run gate (got indices merge=%d, integrate=%d, gate=%d)", mergeIdx, integrateIdx, gateIdx)
	}

	// The reconcile-first addition must not have dropped the
	// existing repo-chooses-the-mechanism framing.
	if !strings.Contains(step, "The skill does not prescribe the mechanism; the project does.") {
		t.Error(`step 9 must retain the mechanism-neutral framing ("The skill does not prescribe the mechanism; the project does.")`)
	}
}
