package policies

import (
	"os"
	"path/filepath"
	"strings"
)

// PolicyM0137AC3BatchedWalker asserts the fsm-history-consistent
// rule (`internal/check/fsm_history_consistent.go`) has been
// retrofitted to use the batched gitops helpers landed in
// M-0137/AC-1 (BulkRevwalk) and AC-2 (BlobReader), and that the
// per-entity walker helpers shipped with M-0130 (walkOneEntity,
// listCommitPathPairs, commitParents, statusAtCommitPath,
// commitTrailers) no longer exist in the file.
//
// Mechanical evidence for M-0137/AC-3 ("fsm-history-consistent: no
// per-entity exec.Command — routes through helpers"). The five
// deleted helpers each carried one or more `exec.Command(...)` calls
// per entity invocation. Their absence — combined with the presence
// of gitops.BulkRevwalk / gitops.NewBlobReader references — proves
// the hot path no longer fans out per-entity subprocesses.
//
// Why a source-check rather than a perf test: the perf claim is
// AC-7's mechanical evidence (a budget assertion on real timings).
// AC-3 is the structural claim that the helpers are in use; a
// presence/absence check on the source file is the cleanest
// mechanical assertion for "routes through helpers". A perf test
// alone wouldn't catch a refactor that re-introduces per-entity
// exec.Command while happening to stay under the budget.
func PolicyM0137AC3BatchedWalker(root string) ([]Violation, error) {
	target := filepath.Join(root, "internal", "check", "fsm_history_consistent.go")
	src, err := os.ReadFile(target)
	if err != nil {
		return nil, err
	}
	content := string(src)
	var out []Violation

	// (1) Must reference gitops.BulkRevwalk — the per-commit batched
	// walker from M-0137/AC-1. Its presence proves the rule's commit
	// walk uses the long-lived subprocess pattern, not the per-entity
	// `git log --follow` fan-out.
	if !strings.Contains(content, "gitops.BulkRevwalk") {
		out = append(out, Violation{
			Policy: "m0137-ac3-batched-walker",
			File:   "internal/check/fsm_history_consistent.go",
			Detail: "does not reference gitops.BulkRevwalk — the M-0137/AC-1 batched commit walker is not in use; the hot path is still per-entity",
		})
	}

	// (2) Must reference gitops.NewBlobReader (or BlobReader as a
	// type) — the cat-file batch pump from M-0137/AC-2. Its presence
	// proves status reads at (commit, path) go through the long-lived
	// cat-file subprocess, not per-call `git show`.
	if !strings.Contains(content, "gitops.NewBlobReader") && !strings.Contains(content, "gitops.BlobReader") {
		out = append(out, Violation{
			Policy: "m0137-ac3-batched-walker",
			File:   "internal/check/fsm_history_consistent.go",
			Detail: "does not reference gitops.NewBlobReader / gitops.BlobReader — the M-0137/AC-2 cat-file batch pump is not in use; status reads still fan out per (commit, path) via git show",
		})
	}

	// (3) Must NOT define the deleted per-entity walker helpers. Each
	// of these shipped in M-0130 carrying one or more direct
	// exec.Command calls; the M-0137 retrofit replaces them with the
	// batched helpers above.
	bannedDefs := []string{
		"func walkOneEntity(",
		"func listCommitPathPairs(",
		"func commitParents(",
		"func statusAtCommitPath(",
		"func commitTrailers(",
	}
	for _, banned := range bannedDefs {
		if strings.Contains(content, banned) {
			name := strings.TrimSuffix(strings.TrimPrefix(banned, "func "), "(")
			out = append(out, Violation{
				Policy: "m0137-ac3-batched-walker",
				File:   "internal/check/fsm_history_consistent.go",
				Detail: "still defines " + name + " — the M-0130 per-entity helper should be deleted by the M-0137/AC-3 retrofit (its callers route through gitops.BulkRevwalk + gitops.BlobReader instead)",
			})
		}
	}
	return out, nil
}
