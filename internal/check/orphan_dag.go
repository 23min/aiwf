package check

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CommitDAG is an in-memory parent map of the repository's commit graph,
// including reflog-only (force-pushed-away) commits, built from a single
// `git rev-list --all --reflog --parents` subprocess. It answers ancestry
// queries in memory, replacing the per-pair `git merge-base --is-ancestor`
// fan-out the orphaned-AI-commit walk previously spawned (683 subprocesses
// on the kernel's own repo at the M-0215 baseline), and — via
// [CommitDAG.FirstParentChain] — the per-branch `git rev-list
// --first-parent` fan-out the isolation-escape oracle spawned (M-0216
// AC-6). Built ONCE per check invocation and shared across both
// consumers (E-0053 / M-0216).
type CommitDAG struct {
	parents map[string][]string
}

// BuildCommitDAG runs one `git rev-list --all --reflog --parents` and
// parses it into a SHA→parents map. The `--reflog` flag is load-bearing:
// the orphaned-AI-commit walk asks about commits that were force-pushed
// away and are no longer reachable from any ref, so a plain `--all` DAG
// would omit exactly the commits the walk inspects (`--all` is a superset
// of what the oracle's first-parent index needs, so the one DAG serves
// both). Returns an error only on a genuine git failure; callers treat
// that as "cannot determine ancestry / first-parent" and fall back to
// their per-call git path, matching the prior behavior.
func BuildCommitDAG(ctx context.Context, root string) (*CommitDAG, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--all", "--reflog", "--parents")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git rev-list --all --reflog --parents: %w", err) //coverage:ignore git rev-list fails only on a corrupt repo; callers fall back to their per-call git path
	}
	return parseCommitDAG(string(out)), nil
}

// parseCommitDAG parses `git rev-list --parents` output — one line per
// commit, "<commit> <parent1> <parent2> ..." (a root commit has no
// parents listed) — into the parent map.
func parseCommitDAG(out string) *CommitDAG {
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
	return &CommitDAG{parents: parents}
}

// FirstParentChain returns the SHAs along the first-parent path from tip
// back to a root — tip, then tip's first parent, then that commit's first
// parent, and so on — matching `git rev-list --first-parent <tip>`
// (newest-first). First parent is parents[0] because `git rev-list
// --parents` lists parents first-parent-first. A commit absent from the
// map (or a root) ends the chain. The seen-guard is defensive against a
// hypothetical cycle; a real git DAG is acyclic. Empty tip yields nil.
func (d *CommitDAG) FirstParentChain(tip string) []string {
	if tip == "" {
		return nil
	}
	var chain []string
	seen := make(map[string]bool)
	for cur := tip; cur != "" && !seen[cur]; {
		seen[cur] = true
		chain = append(chain, cur)
		ps := d.parents[cur]
		if len(ps) == 0 {
			break
		}
		cur = ps[0]
	}
	return chain
}

// isAncestor reports whether old is an ancestor of newer — reachable from
// newer by following parent edges — matching `git merge-base --is-ancestor
// old newer` semantics, including reflexivity (old == newer is true). The
// walk is an iterative DFS over the in-memory parent map.
func (d *CommitDAG) isAncestor(old, newer string) bool {
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
