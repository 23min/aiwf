package check

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// commitDAG is an in-memory parent map of the repository's commit graph,
// including reflog-only (force-pushed-away) commits, built from a single
// `git rev-list --all --reflog --parents` subprocess. It answers ancestry
// queries in memory, replacing the per-pair `git merge-base --is-ancestor`
// fan-out the orphaned-AI-commit walk previously spawned — 683 subprocesses
// on the kernel's own repo at the M-0215 baseline (E-0053 / M-0216).
type commitDAG struct {
	parents map[string][]string
}

// buildCommitDAG runs one `git rev-list --all --reflog --parents` and
// parses it into a SHA→parents map. The `--reflog` flag is load-bearing:
// the orphaned-AI-commit walk asks about commits that were force-pushed
// away and are no longer reachable from any ref, so a plain `--all` DAG
// would omit exactly the commits the walk inspects. Returns an error only
// on a genuine git failure; the caller treats that as "cannot determine
// ancestry" and emits no findings, matching the prior per-call behavior.
func buildCommitDAG(ctx context.Context, root string) (*commitDAG, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--all", "--reflog", "--parents")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git rev-list --all --reflog --parents: %w", err) //coverage:ignore git rev-list fails only on a corrupt repo; WalkOrphanedAICommits then falls back to per-pair merge-base
	}
	return parseCommitDAG(string(out)), nil
}

// parseCommitDAG parses `git rev-list --parents` output — one line per
// commit, "<commit> <parent1> <parent2> ..." (a root commit has no
// parents listed) — into the parent map.
func parseCommitDAG(out string) *commitDAG {
	parents := make(map[string][]string)
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) == 1 {
			parents[fields[0]] = nil
			continue
		}
		parents[fields[0]] = fields[1:]
	}
	return &commitDAG{parents: parents}
}

// isAncestor reports whether old is an ancestor of newer — reachable from
// newer by following parent edges — matching `git merge-base --is-ancestor
// old newer` semantics, including reflexivity (old == newer is true). The
// walk is an iterative DFS over the in-memory parent map.
func (d *commitDAG) isAncestor(old, newer string) bool {
	if old == newer {
		return true
	}
	seen := make(map[string]bool)
	stack := []string{newer}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[n] {
			continue
		}
		seen[n] = true
		for _, p := range d.parents[n] {
			if p == old {
				return true
			}
			if !seen[p] {
				stack = append(stack, p)
			}
		}
	}
	return false
}
