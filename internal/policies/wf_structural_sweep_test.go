package policies

import (
	"strings"
	"testing"
)

// wf-structural-sweep structural tests (G-0450). The skill lives under
// internal/skills/embedded-rituals/**, the canonical authoring location,
// so these tests assert against the bytes the binary embeds. Each assertion is
// scoped to a named markdown section — heading count or section-local
// content — never a flat body grep, per CLAUDE.md §"Substring assertions
// are not structural assertions".
const wfStructuralSweepFixturePath = "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-structural-sweep/SKILL.md"

// TestWfStructuralSweep_HasFourNamedLenses pins the load-bearing shape of
// the skill: a "## The four lenses" section carrying exactly four `###`
// sub-lenses, one each for dead paths, textual clones, convergent
// duplication, and data flow. Dropping a lens (e.g. shipping only the
// mechanical ones and losing the convergence or data-flow lens that
// motivate the pass) fails here.
func TestWfStructuralSweep_HasFourNamedLenses(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfStructuralSweepFixturePath)

	lenses := extractMarkdownSection(body, 2, "The four lenses")
	if lenses == "" {
		t.Fatal("wf-structural-sweep must have a `## The four lenses` section")
	}
	if got := countSubHeadings(lenses, 3); got != 4 {
		t.Errorf("`## The four lenses` has %d `###` sub-lenses; want exactly 4", got)
	}
	for _, want := range []string{"Dead paths", "Textual clones", "Convergent duplication", "Data flow"} {
		if !strings.Contains(lenses, want) {
			t.Errorf("the lenses section is missing a lens named %q", want)
		}
	}
}

// TestWfStructuralSweep_DataFlowLensNamesProducedButUnconsumed pins the
// distinguishing property of Lens 4: it traces producer→consumer and flags a
// value that is "used" in the call-graph sense yet consumed nowhere — the
// class no reachability or clone tool reaches.
func TestWfStructuralSweep_DataFlowLensNamesProducedButUnconsumed(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfStructuralSweepFixturePath)

	lens4 := extractMarkdownSection(body, 3, "Lens 4")
	if lens4 == "" {
		t.Fatal("wf-structural-sweep must have a `### Lens 4 …` data-flow section")
	}
	low := strings.ToLower(lens4)
	if !strings.Contains(low, "consume") {
		t.Error("Lens 4 must frame the trace as producer→consumer")
	}
	if !strings.Contains(low, "unconsumed") {
		t.Error("Lens 4 must flag produced-but-unconsumed values (the class reachability calls live)")
	}
}

// TestWfStructuralSweep_DeadPathLensRequiresReachabilityWithTestRoots pins
// that Lens 1 describes a whole-program reachability analysis with tests as
// roots — the property that distinguishes it from a package-scoped unused
// check and that prevents test-only helpers being mistaken for rot.
func TestWfStructuralSweep_DeadPathLensRequiresReachabilityWithTestRoots(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfStructuralSweepFixturePath)

	lens1 := extractMarkdownSection(body, 3, "Lens 1")
	if lens1 == "" {
		t.Fatal("wf-structural-sweep must have a `### Lens 1 …` dead-path section")
	}
	low := strings.ToLower(lens1)
	if !strings.Contains(low, "reachab") {
		t.Error("Lens 1 must describe a reachability analysis")
	}
	if !strings.Contains(low, "root") {
		t.Error("Lens 1 must specify tests-as-roots so test-only helpers are not flagged as dead")
	}
}

// TestWfStructuralSweep_TriageBeforeDeleteIsInstructed pins the pass's
// sharpest safety step: a dedicated section instructing the invoker to
// check ownership (open issue / accepted decision / coupled spec) before
// proposing any deletion, so a confident removal cannot contradict a
// recorded decision to retain the code.
func TestWfStructuralSweep_TriageBeforeDeleteIsInstructed(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfStructuralSweepFixturePath)

	triage := extractMarkdownSection(body, 2, "Triage before you delete")
	if triage == "" {
		t.Fatal("wf-structural-sweep must have a `## Triage before you delete` section")
	}
	low := strings.ToLower(triage)
	ownership := strings.Contains(low, "owned") || strings.Contains(low, "owner") ||
		strings.Contains(low, "decision") || strings.Contains(low, "issue")
	if !ownership {
		t.Error("the triage section must instruct checking ownership (issue / decision / coupling) before removal")
	}
	if !strings.Contains(low, "delet") && !strings.Contains(low, "remov") {
		t.Error("the triage section must tie the ownership check to the deletion/removal decision")
	}
}

// TestWfStructuralSweep_IsStackAgnostic pins that the skill keeps the
// method stack-agnostic (like wf-codebase-health) with per-stack tool
// examples, rather than hardcoding one stack's tools as the only path —
// the ritual ships into consumer repos of any stack.
func TestWfStructuralSweep_IsStackAgnostic(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfStructuralSweepFixturePath)

	perStack := extractMarkdownSection(body, 2, "Per-stack tools")
	if perStack == "" {
		t.Fatal("wf-structural-sweep must have a `## Per-stack tools` section")
	}
	if !strings.Contains(perStack, "Go") {
		t.Error("the per-stack section should give a Go example")
	}
	low := strings.ToLower(perStack)
	if !strings.Contains(low, "other stack") && !strings.Contains(low, "equivalent") {
		t.Error("the per-stack section must name a non-Go path (other stacks / equivalents), not hardcode Go as the only option")
	}
}
