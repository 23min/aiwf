package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// g0293StartMilestoneFixturePath, g0293WrapMilestoneFixturePath, and
// g0293TddCycleFixturePath are the canonical authoring locations for the
// three ritual skills G-0293's Model 1 decision touches.
const (
	g0293StartMilestoneFixturePath = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md"
	g0293WrapMilestoneFixturePath  = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md"
	g0293TddCycleFixturePath       = "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-tdd-cycle/SKILL.md"
)

// TestG0293_StartMilestoneCommitsImplementationPerAC asserts G-0293 Facet 2:
// `aiwfx-start-milestone`'s per-AC implementation loop (`## Workflow` step 6)
// commits the AC's implementation code on the milestone branch as each AC
// completes, rather than leaving it uncommitted for the wrap to bundle.
//
// Structural per CLAUDE.md *Substring assertions are not structural
// assertions*: scoped to the step-6 `### ` subsection, not grepped file-wide.
func TestG0293_StartMilestoneCommitsImplementationPerAC(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), g0293StartMilestoneFixturePath))
	if err != nil {
		t.Fatalf("reading %s: %v", g0293StartMilestoneFixturePath, err)
	}
	body := string(data)

	step := findWorkflowSubsection(body, "implementation")
	if step == "" {
		t.Fatal("AC: aiwfx-start-milestone `## Workflow` must contain a `### …Implementation…` step for the per-AC loop")
	}
	lower := strings.ToLower(step)

	if !strings.Contains(lower, "commit the ac's implementation code") {
		t.Error("AC: the per-AC loop must instruct committing the AC's implementation code on the milestone branch")
	}
	if !strings.Contains(step, "milestone branch") {
		t.Error("AC: the per-AC commit must be scoped to the milestone branch")
	}
	if !strings.Contains(lower, "before starting the next ac") {
		t.Error("AC: the commit must land before the next AC starts (per-AC cadence, not a batch)")
	}

	// The live phase-promote framing (facet 1) is stated in the same step.
	for _, w := range []string{"live", "never deferred or bursted at wrap"} {
		if !strings.Contains(lower, w) {
			t.Errorf("AC: the implementation step must reaffirm phase promotes fire live, not bursted — missing %q", w)
		}
	}
}

// TestG0293_StartMilestoneNoLongerDefersImplementationToWrap asserts the old
// "do not commit the implementation yet — wrap bundles it" instruction (the
// root cause G-0293 names) is gone from `aiwfx-start-milestone`'s hand-off
// step, replaced with a statement that the implementation is already
// committed by the time wrap runs.
func TestG0293_StartMilestoneNoLongerDefersImplementationToWrap(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), g0293StartMilestoneFixturePath))
	if err != nil {
		t.Fatalf("reading %s: %v", g0293StartMilestoneFixturePath, err)
	}
	body := string(data)

	handoff := findWorkflowSubsection(body, "hand off to wrap")
	if handoff == "" {
		t.Fatal("AC: aiwfx-start-milestone `## Workflow` must contain a `### …Hand off to wrap…` step")
	}

	if strings.Contains(handoff, "Do not commit the implementation yet") {
		t.Error("AC: the stale 'do not commit the implementation yet' instruction (G-0293's root cause) must be gone from the hand-off step")
	}
	if strings.Contains(body, "wrap bundles the implementation") {
		t.Error("AC: no reference to the wrap 'bundling the implementation' may remain anywhere in the file")
	}
	if !strings.Contains(strings.ToLower(handoff), "already committed") {
		t.Error("AC: the hand-off step must state the implementation is already committed per-AC")
	}
}

// TestG0293_WrapMilestoneDoesNotBundleImplementation asserts G-0293 Facet 2
// on the wrap side: `aiwfx-wrap-milestone`'s staging step stages only the
// milestone spec (wrap-side prose), and says so explicitly — it does not
// bundle the implementation, because the implementation already landed
// per-AC during `aiwfx-start-milestone`.
func TestG0293_WrapMilestoneDoesNotBundleImplementation(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), g0293WrapMilestoneFixturePath))
	if err != nil {
		t.Fatalf("reading %s: %v", g0293WrapMilestoneFixturePath, err)
	}
	body := string(data)

	stage := findWorkflowSubsection(body, "stage all changes")
	if stage == "" {
		t.Fatal("AC: aiwfx-wrap-milestone `## Workflow` must contain a `### …Stage all changes…` step")
	}
	lower := strings.ToLower(stage)
	if !strings.Contains(lower, "already committed, per-ac") {
		t.Error("AC: the staging step must state the implementation is already committed per-AC")
	}
	if !strings.Contains(lower, "does not bundle any source or test files") {
		t.Error("AC: the staging step must explicitly disclaim bundling source/test files")
	}

	commitGate := findWorkflowSubsection(body, "commit gate")
	if commitGate == "" {
		t.Fatal("AC: aiwfx-wrap-milestone `## Workflow` must retain a `### …Commit gate…` step")
	}
	if !strings.Contains(strings.ToLower(commitGate), "per-ac implementation commits") {
		t.Error("AC: the commit gate summary must point at the per-AC implementation commits already on the branch")
	}
}

// TestG0293_TddCycleDefersMetAndWorkLogToCaller closes the residual half of
// G-0293 Facet 2: `wf-tdd-cycle`'s RECORD step used to promote the AC to
// `met` and append a Work log entry citing `commit <SHA>` itself — before
// any implementation commit existed, since wf-tdd-cycle never commits code.
// That's the exact "SHA doesn't exist yet" bug G-0293 names. RECORD must
// stop at `phase: done` and explicitly hand `met` + the commit + the Work
// log entry to the calling milestone ritual, which commits the
// implementation before citing its SHA (per the other two tests above).
func TestG0293_TddCycleDefersMetAndWorkLogToCaller(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), g0293TddCycleFixturePath))
	if err != nil {
		t.Fatalf("reading %s: %v", g0293TddCycleFixturePath, err)
	}
	body := string(data)

	record := sectionUnder(body, "RECORD")
	if record == "" {
		t.Fatal("AC: wf-tdd-cycle must have a 'RECORD' section")
	}
	lower := strings.ToLower(record)

	if strings.Contains(lower, "mark the acceptance criterion `met`") {
		t.Error("AC: RECORD must no longer promote the AC to `met` itself — that belongs to the calling milestone ritual, which has the implementation commit's SHA")
	}
	if strings.Contains(lower, "append a work log entry") {
		t.Error("AC: RECORD must no longer append the Work log entry itself — it never commits code, so citing `commit <SHA>` here was always premature")
	}
	if !strings.Contains(lower, "calling") || !strings.Contains(lower, "does not exist until this cycle returns") {
		t.Error("AC: RECORD must explicitly defer `met` + the commit + the Work log entry to the calling ritual, and say why (the SHA doesn't exist yet)")
	}

	// Phase promotion to `done` is still RECORD's own job.
	if !strings.Contains(record, "aiwf promote M-NNN/AC-<N> --phase done") {
		t.Error("AC: RECORD must still advance the AC's tdd_phase to done — only the met/commit/work-log handoff changed")
	}
}
