package stresstest

import (
	"fmt"
	"path/filepath"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
)

// parallel_branch_reallocate.go — M-0243/AC-1: ParallelBranchReallocateScenario
// reproduces CLAUDE.md's own "Id-collision resolution at merge time"
// story literally: two operators, each working from an independent
// clone of one bare origin, allocate an id from the identical,
// unadvanced trunk state. Unlike M-0241/AC-3's
// CrossWorktreeIDRaceScenario (real concurrent goroutines racing a
// probabilistic timing window within sibling worktrees of ONE repo),
// the collision here is deterministic by construction — AllocateID
// (internal/entity/allocate.go) is a pure max(existing-ids)+1
// function, so two isolated clones computing it from the same
// pre-push state always land on the same id, no timing dependency at
// all. The scenario drives the actual merge/push sequence — the
// first push succeeds, the second is rejected non-fast-forward, the
// operator fetches and merges, and the merged tree now carries the
// collision — through to resolution: `aiwf check` surfaces it, `aiwf
// reallocate` clears it, and the final push, now fast-forward,
// succeeds.

const (
	operatorATitle = "operatora"
	operatorBTitle = "operatorb"
)

// ParallelBranchReallocateScenario implements Scenario.
type ParallelBranchReallocateScenario struct {
	aiwfBin    string
	kind       entity.Kind
	violations []Violation
}

// NewParallelBranchReallocateScenario builds a scenario driving one
// `aiwf add <kind>` per operator clone, then the merge/push/reallocate
// sequence between them.
func NewParallelBranchReallocateScenario(aiwfBin string, kind entity.Kind) *ParallelBranchReallocateScenario {
	return &ParallelBranchReallocateScenario{aiwfBin: aiwfBin, kind: kind}
}

// Setup creates a bare origin repo and clones it into dir/operator-a
// and dir/operator-b.
func (s *ParallelBranchReallocateScenario) Setup(dir string) error {
	return newBareOriginWithClonesFixture(dir, "operator-a", "operator-b")
}

// Run drives both operators' adds, the merge/push sequence to the
// point of collision, and the reallocate-based resolution.
func (s *ParallelBranchReallocateScenario) Run(dir string) error {
	opA := filepath.Join(dir, "operator-a")
	opB := filepath.Join(dir, "operator-b")

	envA, err := runAiwfJSON(s.aiwfBin, opA, "add", string(s.kind), "--title", operatorATitle, "--body", "parallel-branch reallocate stress operator")
	if err != nil { //coverage:ignore defensive: covered by the same launch-failure class other scenarios pin at runAiwfJSON's own source
		return fmt.Errorf("operator A add: %w", err)
	}
	envB, err := runAiwfJSON(s.aiwfBin, opB, "add", string(s.kind), "--title", operatorBTitle, "--body", "parallel-branch reallocate stress operator")
	if err != nil { //coverage:ignore defensive: see operator A above
		return fmt.Errorf("operator B add: %w", err)
	}
	if envA.Status != "ok" || envB.Status != "ok" {
		return fmt.Errorf("operator add did not report ok: a=%+v b=%+v", envA, envB)
	}
	if envA.Metadata.EntityID != envB.Metadata.EntityID { //coverage:ignore defensive: two independent clones computing "next free id" (internal/entity/allocate.go's pure max+1 function) from an identical, unadvanced origin state always allocate the same id; reaching this branch would mean the allocator started incorporating per-clone-unique input, which it doesn't
		return fmt.Errorf("operator A and operator B allocated different ids (%s vs %s) — the scenario's deterministic-collision premise did not hold", envA.Metadata.EntityID, envB.Metadata.EntityID)
	}

	if pushAErr := runGit(opA, "push", "-q", "origin", "HEAD:main"); pushAErr != nil { //coverage:ignore defensive: operator A's push is always the first push after the seed commit to a freshly-created origin, no realistic failure mode
		return fmt.Errorf("operator A pushing to origin: %w", pushAErr)
	}

	if pushErr := runGit(opB, "push", "-q", "origin", "HEAD:main"); pushErr == nil { //coverage:ignore defensive: operator B's local main always diverges from origin/main at this point (operator A just advanced it moments ago) — git always rejects this as non-fast-forward
		return fmt.Errorf("operator B's naive push unexpectedly succeeded — expected a non-fast-forward rejection")
	}

	if fetchErr := runGit(opB, "fetch", "-q", "origin"); fetchErr != nil { //coverage:ignore defensive: fetching from a reachable bare origin this scenario itself created has no realistic failure mode
		return fmt.Errorf("operator B fetching origin: %w", fetchErr)
	}
	if mergeErr := runGit(opB, "merge", "-q", "--no-edit", "origin/main"); mergeErr != nil { //coverage:ignore defensive: operator A and operator B's adds touch disjoint paths (distinct fixed slugs), so this merge is always a clean fast path with no realistic conflict
		return fmt.Errorf("operator B merging origin/main: %w", mergeErr)
	}

	checkEnv, err := runAiwfJSON(s.aiwfBin, opB, "check")
	if err != nil { //coverage:ignore defensive: covered by the same launch-failure class other scenarios pin at runAiwfJSON's own source
		return fmt.Errorf("running aiwf check after the merge: %w", err)
	}

	bPath, err := findEntityFile(opB, envB.Metadata.EntityID, operatorBTitle)
	if err != nil { //coverage:ignore defensive: findEntityFile's own not-found branch is unit-tested directly at its source (cross_worktree_id_race_classify_test.go); a real collision always leaves operator B's file in place under its known, fixed slug
		return fmt.Errorf("locating operator B's colliding entity file: %w", err)
	}
	reallocEnv, err := runAiwfJSON(s.aiwfBin, opB, "reallocate", bPath)
	if err != nil { //coverage:ignore defensive: covered by the same launch-failure class other scenarios pin at runAiwfJSON's own source
		return fmt.Errorf("running aiwf reallocate: %w", err)
	}

	postCheckEnv, err := runAiwfJSON(s.aiwfBin, opB, "check")
	if err != nil { //coverage:ignore defensive: covered by the same launch-failure class other scenarios pin at runAiwfJSON's own source
		return fmt.Errorf("running aiwf check after reallocate: %w", err)
	}

	pushedClean := runGit(opB, "push", "-q", "origin", "HEAD:main") == nil

	s.violations = classifyParallelBranchReallocate(checkEnv.Findings, reallocEnv.Status, postCheckEnv.Findings, pushedClean)
	return nil
}

// Verify returns every violation Run collected.
func (s *ParallelBranchReallocateScenario) Verify(_ string) []Violation {
	return s.violations
}

// classifyParallelBranchReallocate judges one parallel-branch
// collision attempt: aiwf check must have surfaced the collision as
// CodeIDsUnique, the reallocate must have reported "ok", a follow-up
// check must no longer carry CodeIDsUnique, and the final push (now
// fast-forward) must have succeeded.
func classifyParallelBranchReallocate(checkFindings []verbEnvelopeFinding, reallocateStatus string, postCheckFindings []verbEnvelopeFinding, pushedClean bool) []Violation {
	var violations []Violation
	if !hasFindingCode(checkFindings, check.CodeIDsUnique) {
		violations = append(violations, Violation{Message: "a real parallel-branch id collision occurred but aiwf check did not surface it as " + check.CodeIDsUnique})
	}
	if reallocateStatus != "ok" {
		violations = append(violations, Violation{Message: fmt.Sprintf("aiwf reallocate did not cleanly resolve the collision (status=%s)", reallocateStatus)})
	}
	if hasFindingCode(postCheckFindings, check.CodeIDsUnique) {
		violations = append(violations, Violation{Message: check.CodeIDsUnique + " finding still present after aiwf reallocate"})
	}
	if !pushedClean {
		violations = append(violations, Violation{Message: "operator B's final push after reallocate did not succeed cleanly"})
	}
	return violations
}
