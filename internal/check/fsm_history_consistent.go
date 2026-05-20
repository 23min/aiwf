package check

import (
	"bytes"
	"context"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// FSMHistoryConsistent is the kernel chokepoint that makes the per-
// entity status FSM a tree-invariant rather than just a verb-
// precondition (closes G-0132 when all of M-0130 lands). The rule
// walks every entity's commit history in DAG order, observes every
// status-change commit, and — once AC-2/3/4 land the per-subcode
// predicates — emits findings per violation.
//
// M-0130 lands the rule incrementally:
//
//   - AC-1 (this file): walker scaffolding. walkStatusChanges
//     enumerates status-change observations across every entity in
//     the tree via DAG-aware per-parent comparison. FSMHistoryConsistent
//     returns no findings yet — the per-subcode predicates land in
//     AC-2/3/4. The walker's correctness (per-parent comparison,
//     rename tracking via --follow, single-commit and no-change
//     short-circuits, multi-entity independence, branched-history
//     phantom-transition avoidance) is pinned by the test suite in
//     fsm_history_consistent_test.go.
//   - AC-2: illegal-transition subcode — observation's (Prior, Next)
//     is outside entity.AllowedTransitions and Trailers lacks
//     aiwf-force.
//   - AC-3: forced-untrailered subcode — sovereign-act-shape
//     transition (per entity.IsSovereignActShape) without aiwf-force
//     trailer.
//   - AC-4: manual-edit subcode — catch-all: legal-in-FSM AND not
//     sovereign-act-shape AND no aiwf-verb trailer. Includes audit-
//     only suppression.
//   - AC-5: hint table entries + SKILL.md rows (already landed).
//   - AC-6: audit catalog update (legal-workflows-audit.md).
//
// The rule is wired in internal/cli/check/check.go's Run() alongside
// RunProvenanceCheck, NOT in this package's Run(). The per-entity
// git-walk is too expensive for the per-commit pre-commit hook's
// policy-test path; pre-push and explicit `aiwf check` invocations
// get the full audit.
//
// Walker design contract (the load-bearing correctness pin for AC-1):
//
// The walker is DAG-aware, not linearization-aware. For each commit
// C that touched an entity's file, the walker compares C's status
// against the status at each of C's actual git parents (not against
// the linearization-neighbor commit in `git log --follow` output).
// The original AC-1 design walked linearization adjacency, which
// silently produced phantom transitions across branch boundaries —
// e.g., a retitle commit on a feature branch with status=proposed
// followed in `git log` order by a promote-to-active on a parallel
// branch would emit an "active → proposed" observation that
// corresponds to no real edit. Per-parent comparison eliminates the
// phantom by structurally restricting comparisons to actual parent-
// child edges in the DAG.
func FSMHistoryConsistent(ctx context.Context, root string, t *tree.Tree) []Finding {
	if t == nil || root == "" {
		return nil
	}
	_, err := walkStatusChanges(ctx, root, t)
	if err != nil {
		// AC-3/4 may route walker errors into the finding stream
		// (e.g., a "history-walk-error" subcode). For now the rule is
		// a clean no-op for trees where the per-entity git log fails
		// (rare; usually permission issues or transient cancellation).
		return nil
	}
	// AC-1 emits no findings — the walker's job is just to enumerate
	// observations. AC-2 / AC-3 / AC-4 will append per-subcode
	// predicate results here.
	return nil
}

// statusChange records one observed status-change for an entity at
// one commit, relative to one of the commit's parents. Multi-parent
// (merge) commits may yield one observation per parent where the
// status differs at that parent's path-at-commit.
//
// Unexported because it's an intermediate value passed from the
// walker to the per-subcode predicates that land in AC-2/3/4 — not a
// public surface of the check package.
//
// Fields:
//   - EntityID, EntityKind, Path: identify the entity and where its
//     file lives at the observed commit. Path may differ across
//     observations of the same entity when `aiwf rename` has moved
//     the file (--follow tracks the history).
//   - Commit: full SHA of the status-change commit (not the parent).
//   - Parent: full SHA of the parent commit this observation is
//     relative to. For single-parent commits, this is the only
//     parent. For multi-parent (merge) commits, the same Commit may
//     appear in multiple observations — one per parent where the
//     status differed.
//   - Prior: the status field value at Parent (read at Path).
//   - Next: the status field value at Commit (read at Path).
//   - Trailers: the aiwf-* trailers parsed from Commit's message.
//     Keys are bare (no "aiwf-" prefix stripping). Used by AC-2/3/4
//     predicates to classify the change.
type statusChange struct {
	EntityID   string
	EntityKind entity.Kind
	Commit     string
	Parent     string
	Path       string
	Prior      string
	Next       string
	Trailers   map[string]string
}

// walkStatusChanges enumerates DAG-aware status-change observations
// across every entity in t. Returns one observation per (entity,
// commit, parent) tuple where the entity's status at the parent
// (under the same on-disk path) differs from its value at the commit
// itself.
//
// Returns (nil, nil) when t is nil, root is empty, or root is not a
// git repo with at least one commit reachable from HEAD. Per-entity
// walker errors propagate as (nil, err) — callers can choose to
// swallow or route them as findings (FSMHistoryConsistent swallows
// for AC-1).
func walkStatusChanges(ctx context.Context, root string, t *tree.Tree) ([]statusChange, error) {
	if t == nil || root == "" {
		return nil, nil
	}
	if !hasGitCommits(ctx, root) {
		return nil, nil
	}
	var out []statusChange
	for _, e := range t.Entities {
		if e == nil || e.Path == "" {
			continue
		}
		changes, err := walkOneEntity(ctx, root, e)
		if err != nil {
			return nil, err
		}
		out = append(out, changes...)
	}
	return out, nil
}

// walkOneEntity returns DAG-aware status-change observations for a
// single entity.
//
// For each commit C in the entity's `git log --follow` history:
//
//  1. Enumerate C's parents (`git log -1 --pretty=format:%P C`).
//  2. For each parent P, read the entity file at (P, path-at-C) via
//     `git show P:<path>`. If the read fails — the file doesn't
//     exist at P under path-at-C (P pre-dates an add, or C is a
//     rename and P has the file under the OLD name), or the file
//     has no parseable frontmatter — the (P, C) pair is silently
//     skipped.
//  3. Read C's own status at path-at-C.
//  4. Emit one observation when both statuses are non-empty and
//     differ.
//
// Rename handling: `git log --follow` traverses the rename, so the
// entity's pre-rename commits appear in the touches list with their
// OLD path, and post-rename commits with their NEW path. Each commit
// is compared against its actual parent at the parent's own path
// (which matches the file's path AT that parent). The (rename
// commit, its parent) pair itself produces no observation — the
// parent has the file at the OLD name, `git show P:<NEW-name>`
// fails, the pair is skipped. Pure renames don't change status, so
// no observation is lost. The rare commit that both renames AND
// changes status is unobserved — accepted as a known non-handling.
//
// Multi-parent (merge) commits emit per-parent observations: if M
// has parents P1 and P2 with different statuses, M produces up to
// two observations. Whether merges count as "real" predicate events
// is left to the AC-2/3/4 predicates' filtering. This avoids
// baking merge semantics into the walker.
func walkOneEntity(ctx context.Context, root string, e *entity.Entity) ([]statusChange, error) {
	pairs, err := listCommitPathPairs(ctx, root, e.Path)
	if err != nil {
		return nil, err
	}
	if len(pairs) == 0 {
		return nil, nil
	}
	var out []statusChange
	for _, p := range pairs {
		parents := commitParents(ctx, root, p.Commit)
		if len(parents) == 0 {
			// Root commit: no parent to compare against. The file may
			// have appeared at this commit (the initial import) — no
			// prior status to compute a delta against. Skip.
			continue
		}
		next := statusAtCommitPath(ctx, root, p.Commit, p.Path)
		if next == "" {
			continue
		}
		var trailers map[string]string
		for _, parent := range parents {
			prior := statusAtCommitPath(ctx, root, parent, p.Path)
			if prior == "" || prior == next {
				continue
			}
			if trailers == nil {
				trailers = commitTrailers(ctx, root, p.Commit)
			}
			out = append(out, statusChange{
				EntityID:   e.ID,
				EntityKind: e.Kind,
				Commit:     p.Commit,
				Parent:     parent,
				Path:       p.Path,
				Prior:      prior,
				Next:       next,
				Trailers:   trailers,
			})
		}
	}
	return out, nil
}

// commitPathPair couples a commit SHA with the file path at that
// commit. The path may differ across pairs when --follow has
// traversed a rename; reading the file's content at a given commit
// requires both values (`git show <sha>:<path>`).
type commitPathPair struct {
	Commit string
	Path   string
}

// listCommitPathPairs returns (commit SHA, path-at-commit) pairs for
// every commit that touched currentPath. Uses --follow so renames
// are tracked across the path's history; --name-only gives the
// path-at-each-commit, parsed alongside the COMMIT-prefixed SHA
// lines.
//
// The custom "COMMIT %H" prefix distinguishes the SHA lines from the
// path lines without relying on whitespace heuristics (git's default
// --pretty output mixes blank lines and metadata in ways that break
// naive parsing).
//
// Order doesn't matter to the DAG-aware walker — each pair is
// compared against its commit's actual git parents independently —
// so we deliberately do not reverse the result. (The original AC-1
// design relied on adjacency-in-list semantics and needed reverse-
// ordering; the bug that triggered M-0130's AC-1 redo originated in
// that very adjacency assumption.)
func listCommitPathPairs(ctx context.Context, root, currentPath string) ([]commitPathPair, error) {
	cmd := exec.CommandContext(ctx, "git", "log",
		"--follow", "--name-only",
		"--pretty=format:COMMIT %H",
		"--", currentPath)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var pairs []commitPathPair
	var pendingCommit string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if rest, ok := strings.CutPrefix(line, "COMMIT "); ok {
			pendingCommit = rest
			continue
		}
		if pendingCommit == "" {
			continue
		}
		pairs = append(pairs, commitPathPair{Commit: pendingCommit, Path: line})
		pendingCommit = ""
	}
	return pairs, nil
}

// commitParents returns the parent SHAs of the named commit. Returns
// nil for the root commit (no parents) and for any read failure.
// Multi-parent (merge) commits return all parents in git's declared
// order — first-parent is conventionally the mainline-being-merged-
// into; the walker treats all parents uniformly.
func commitParents(ctx context.Context, root, commit string) []string {
	cmd := exec.CommandContext(ctx, "git", "log", "-1",
		"--pretty=format:%P", commit)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil
	}
	return strings.Fields(trimmed)
}

// statusAtCommitPath reads the entity file at the named commit +
// path via `git show <commit>:<path>` and parses the status field
// from its YAML frontmatter. Returns "" when the file doesn't exist
// at that commit, has no frontmatter delimiter, has no status field,
// or fails YAML parsing — all four "I can't determine the status
// here" cases collapse to the same skip-this-pair signal.
func statusAtCommitPath(ctx context.Context, root, commit, path string) string {
	cmd := exec.CommandContext(ctx, "git", "show", commit+":"+path)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return parseStatusFromFrontmatter(out)
}

// parseStatusFromFrontmatter extracts the status field from an
// entity file's YAML frontmatter (the block between leading `---`
// and the next `---`). Returns "" for any failure mode: missing
// frontmatter, unterminated frontmatter, YAML parse error, or
// absent status field.
//
// Accepts both `---\n` and `---\r\n` opening sequences so files
// written on Windows hosts still parse.
func parseStatusFromFrontmatter(content []byte) string {
	var rest []byte
	switch {
	case bytes.HasPrefix(content, []byte("---\n")):
		rest = content[4:]
	case bytes.HasPrefix(content, []byte("---\r\n")):
		rest = content[5:]
	default:
		return ""
	}
	end := bytes.Index(rest, []byte("\n---"))
	if end < 0 {
		return ""
	}
	var meta struct {
		Status string `yaml:"status"`
	}
	if err := yaml.Unmarshal(rest[:end], &meta); err != nil {
		return ""
	}
	return meta.Status
}

// commitTrailers reads the commit's aiwf-* trailers (and any other
// trailer-shaped lines) and returns them as a key → value map.
// Returns nil when git emits no trailers or the read fails.
//
// Multiple trailers with the same key collapse to the last value,
// which is sufficient for the AC-2/3/4 predicates' boolean-ish use
// ("is aiwf-verb present?", "is aiwf-force present?"). If a future
// subcode needs multi-value-per-key semantics, switch to the slice
// form.
func commitTrailers(ctx context.Context, root, commit string) map[string]string {
	cmd := exec.CommandContext(ctx, "git", "log", "-1",
		"--pretty=%(trailers:only=true,unfold=true)", commit)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	parsed := gitops.ParseTrailers(string(out))
	if len(parsed) == 0 {
		return nil
	}
	m := make(map[string]string, len(parsed))
	for _, tr := range parsed {
		m[tr.Key] = tr.Value
	}
	return m
}

// hasGitCommits reports whether root is a git repo with at least one
// commit reachable from HEAD. Returns false for non-git directories,
// for git repos with no commits yet, and for any other condition
// that makes HEAD unresolvable.
func hasGitCommits(ctx context.Context, root string) bool {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--verify", "--quiet", "HEAD")
	cmd.Dir = root
	return cmd.Run() == nil
}
