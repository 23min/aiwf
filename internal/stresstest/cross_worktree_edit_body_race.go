package stresstest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/check"
)

// cross_worktree_edit_body_race.go — M-0243/AC-2:
// CrossWorktreeEditBodyRaceScenario reproduces G-0212 item 2 literally:
// two operators, each in a sibling worktree of one repo, run `aiwf
// edit-body` on the SAME pre-existing entity with different content —
// concurrent in git-history terms (two sibling commits off the same
// parent), though the AC's own framing allows for them to be minutes
// apart in wall-clock terms. Empirically confirmed (manual git
// experiment, not a guess): merging one operator's branch into the
// other's ALWAYS produces a genuine git conflict, never a silent
// last-writer-wins overwrite — `edit-body` replaces the whole body
// field, so two different edits to the same field are, structurally,
// two changes to the same lines. This is a better outcome than G-0212
// feared (maximally observable, not silent), per this milestone's own
// "assert what the scenario actually observes" constraint. The
// scenario's oracle: whichever way the merge resolves, some trace of
// BOTH operators' edits must survive — the finding this AC exists to
// make, not an assumption baked into it, is what that oracle records.

const editBodyRaceEntityID = "G-0001"

// crossWorktreeEditBodyRaceExpectedWarnings is the baseline of finding
// codes this scenario's post-merge check is expected to carry
// (M-0257/AC-1), beyond the merge-outcome assertion
// classifyCrossWorktreeEditBodyRace already pins directly:
//
//   - provenance-untrailered-scope-undefined: sibling worktrees of one
//     repo never configure a separate upstream remote.
//
// Any OTHER finding — any error-severity finding, or a warning with a
// code not in this set — is a real violation. Holds regardless of
// whether the merge conflicted (a conflict marker sits entirely within
// the entity's body prose, never its frontmatter, so `aiwf check`
// still loads and validates the entity normally either way).
var crossWorktreeEditBodyRaceExpectedWarnings = map[string]bool{
	check.CodeProvenanceUntrailedScopeUndefined: true,
}

// CrossWorktreeEditBodyRaceScenario implements Scenario.
type CrossWorktreeEditBodyRaceScenario struct {
	aiwfBin    string
	violations []Violation

	// skipOperatorBEdit, when true, has operator B never edit the
	// shared entity, so the real merge Run drives is a genuine clean
	// (non-conflicting) one instead of the always-conflicting default
	// — the wiring's other real branch. Test-only: every
	// registered/production use leaves this at its zero value.
	skipOperatorBEdit bool
}

// NewCrossWorktreeEditBodyRaceScenario builds a scenario that races
// two `aiwf edit-body` invocations against one pre-existing entity
// from sibling worktrees.
func NewCrossWorktreeEditBodyRaceScenario(aiwfBin string) *CrossWorktreeEditBodyRaceScenario {
	return &CrossWorktreeEditBodyRaceScenario{aiwfBin: aiwfBin}
}

// Setup creates a main repo seeded with one gap entity, then adds two
// sibling worktrees (actor-a, actor-b) off that same seeded commit —
// dir/main, dir/wt-a, dir/wt-b.
func (s *CrossWorktreeEditBodyRaceScenario) Setup(dir string) error {
	mainDir := filepath.Join(dir, "main")
	if err := os.MkdirAll(mainDir, 0o755); err != nil { //coverage:ignore defensive: mainDir is a fresh subdirectory of RunScenario's own os.MkdirTemp result, no realistic failure mode short of filesystem sabotage
		return fmt.Errorf("creating main repo dir: %w", err)
	}
	if err := gitInitAndConfig(mainDir); err != nil { //coverage:ignore defensive: gitInitAndConfig's own internal branch already carries this rationale
		return err
	}
	addEnv, err := runAiwfJSON(s.aiwfBin, mainDir, "add", "gap", "--title", "race", "--body", "original body before the cross-worktree edit race")
	if err != nil { //coverage:ignore defensive: covered by the same launch-failure class other scenarios pin at runAiwfJSON's own source
		return fmt.Errorf("seeding the shared entity: %w", err)
	}
	if addEnv.Status != "ok" {
		return fmt.Errorf("seeding the shared entity: aiwf did not report ok (status=%s, error=%+v)", addEnv.Status, addEnv.Error)
	}
	if err := runGit(mainDir, "worktree", "add", "-q", "-b", "actor-a", filepath.Join(dir, "wt-a")); err != nil { //coverage:ignore defensive: adding a worktree at a fresh, never-before-used path has no realistic failure mode
		return err
	}
	if err := runGit(mainDir, "worktree", "add", "-q", "-b", "actor-b", filepath.Join(dir, "wt-b")); err != nil { //coverage:ignore defensive: see the actor-a worktree add above
		return err
	}
	return nil
}

// draftAText / draftBText are the two operators' independent body
// edits — distinct, plain text (no id-shaped tokens) so the classify
// step can confirm each one's literal survival in the merge outcome.
const (
	draftAText = "operator A's independent edit to the shared entity"
	draftBText = "operator B's independent edit to the shared entity"
)

// Run drives operator A's edit-body call against their own worktree
// (and, unless skipOperatorBEdit is set, operator B's too), then
// merges actor-b into actor-a's worktree and classifies however that
// merge resolves.
func (s *CrossWorktreeEditBodyRaceScenario) Run(dir string) error {
	wtA := filepath.Join(dir, "wt-a")
	wtB := filepath.Join(dir, "wt-b")

	draftAPath := filepath.Join(dir, "draft-a.txt")
	if err := os.WriteFile(draftAPath, []byte(draftAText+"\n"), 0o644); err != nil { //coverage:ignore defensive: writing a fresh file under this scenario's own os.MkdirTemp dir has no realistic failure mode short of filesystem sabotage
		return fmt.Errorf("writing operator A's draft: %w", err)
	}
	envA, err := runAiwfJSON(s.aiwfBin, wtA, "edit-body", editBodyRaceEntityID, "--body-file", draftAPath)
	if err != nil { //coverage:ignore defensive: covered by the same launch-failure class other scenarios pin at runAiwfJSON's own source
		return fmt.Errorf("operator A edit-body: %w", err)
	}

	envBStatus := "ok" // no B edit attempted when skipped, so nothing to fail
	if !s.skipOperatorBEdit {
		draftBPath := filepath.Join(dir, "draft-b.txt")
		if writeErr := os.WriteFile(draftBPath, []byte(draftBText+"\n"), 0o644); writeErr != nil { //coverage:ignore defensive: see operator A's draft above
			return fmt.Errorf("writing operator B's draft: %w", writeErr)
		}
		envB, editErr := runAiwfJSON(s.aiwfBin, wtB, "edit-body", editBodyRaceEntityID, "--body-file", draftBPath)
		if editErr != nil { //coverage:ignore defensive: see operator A above
			return fmt.Errorf("operator B edit-body: %w", editErr)
		}
		envBStatus = envB.Status
	}
	if envA.Status != "ok" || envBStatus != "ok" {
		return fmt.Errorf("operator edit-body did not report ok: a=%+v b-status=%s", envA, envBStatus)
	}

	conflicted := runGit(wtA, "merge", "--no-edit", "actor-b") != nil

	entityPath := filepath.Join(wtA, "work", "gaps", editBodyRaceEntityID+"-race.md")
	mergedBytes, err := os.ReadFile(entityPath)
	if err != nil { //coverage:ignore defensive: the shared entity's file is never deleted or renamed by either operator's edit-body call — only its body content changes
		return fmt.Errorf("reading the merged entity file: %w", err)
	}

	s.violations = classifyCrossWorktreeEditBodyRace(conflicted, string(mergedBytes), draftAText, draftBText)

	// M-0257/AC-1: alongside the merge-outcome assertion above, confirm
	// the merged tree stays check-clean beyond baseline noise — this
	// scenario never ran `aiwf check` at all before.
	checkEnv, err := runAiwfJSON(s.aiwfBin, wtA, "check")
	if err != nil { //coverage:ignore defensive: covered by the same launch-failure class other scenarios pin at runAiwfJSON's own source
		return fmt.Errorf("running aiwf check after the merge: %w", err)
	}
	s.violations = append(s.violations, classifyAgainstBaseline(checkEnv.Findings, crossWorktreeEditBodyRaceExpectedWarnings)...)
	return nil
}

// Verify returns every violation Run collected.
func (s *CrossWorktreeEditBodyRaceScenario) Verify(_ string) []Violation {
	return s.violations
}

// classifyCrossWorktreeEditBodyRace judges one cross-worktree
// edit-body race: whichever way the merge resolved, some trace of
// BOTH operators' intended content must survive somewhere recoverable
// — a conflicted merge must show both sides in its conflict markers; a
// clean (non-conflicting) merge must land on exactly one operator's
// content, never a third, unexplained value that matches neither.
func classifyCrossWorktreeEditBodyRace(conflicted bool, mergedContent, draftA, draftB string) []Violation {
	if conflicted {
		var violations []Violation
		if !strings.Contains(mergedContent, draftA) {
			violations = append(violations, Violation{Message: "the conflicted merge lost operator A's content — not even a conflict marker preserved it"})
		}
		if !strings.Contains(mergedContent, draftB) {
			violations = append(violations, Violation{Message: "the conflicted merge lost operator B's content — not even a conflict marker preserved it"})
		}
		return violations
	}
	if !strings.Contains(mergedContent, draftA) && !strings.Contains(mergedContent, draftB) {
		return []Violation{{Message: "a clean (non-conflicting) merge produced content matching neither operator's edit — silent, untraceable data loss"}}
	}
	return nil
}
