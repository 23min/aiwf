package check

import (
	"context"
	"os/exec"
	"regexp"
	"strings"
)

// cherry_picks.go — M-0159/AC-6: the canonical home for the
// sovereign-human cherry-pick gather-side. Mirrors the structure
// of acks.go (the M-0159/AC-3 retroactive-acknowledgment gather)
// since both feed map[string]bool exemptions into rules consumed
// from the CLI gather layer at internal/cli/check/.
//
// Closes G-0202 (the parked gather-side that left
// `internal/cli/check/provenance.go:67` passing `nil` for
// cherryPicked under RunIsolationEscape). With nil, the M-0106
// isolation-escape rule's cherry-pick suppression arm at
// internal/check/isolation_escape.go:269 (the
// `if cherryPicked[c.SHA]` block — distinct from the M-0136/AC-3
// ackedSHAs arm immediately above it at line 257) could not fire
// end-to-end; sovereign human re-authors via `git cherry-pick -x`
// of an AI commit landed on a non-bound branch were spuriously
// flagged as escapes.

// cherryPickedMarkerRE matches the canonical
// `(cherry picked from commit <sha>)` line that `git cherry-pick
// -x` writes by default. The SHA token allows 4+ hex chars to
// accommodate any reasonable abbreviation (git's default
// core.abbrev minimum is 4); in practice `git cherry-pick -x`
// writes the full 40-char form, but the relaxed pattern keeps the
// gather robust against `--abbrev-commit`-style upstream rewrites
// of historical commits.
var cherryPickedMarkerRE = regexp.MustCompile(`\(cherry picked from commit [0-9a-fA-F]{4,}\)`)

// WalkCherryPicks walks HEAD's reachable history for commits that
// are sovereign-human cherry-pick re-authors per the both-signals
// contract pinned in the RunIsolationEscape docstring at
// internal/check/isolation_escape.go:67-89 (the canonical contract;
// keep edits there, not duplicated here). This helper is the
// gather-side derivation that produces the cherryPicked map the
// rule consumes; the contract is rule-side.
//
// Derivation steps (one `git log` subprocess + an in-memory filter):
//
//  1. `git log --pretty=format:%H<US>%ae<US>%ce<US>%B<RS> HEAD`
//     emits one record per HEAD-reachable commit (SHA, author email,
//     committer email, full body), null-byte-free, separated by
//     ASCII unit (US) and record (RS) control chars.
//  2. For each record, the commit qualifies iff author email AND
//     committer email AND author email != committer email (the
//     identity gap) AND the body matches cherryPickedMarkerRE (the
//     `(cherry picked from commit <sha>)` marker line). Both
//     signals required — see rule docstring for the rationale.
//  3. Qualifying SHAs land in the returned map.
//
// Returns nil for non-git directories and empty histories; the
// rule treats nil and an empty map identically (no exemptions).
// Performance: one git subprocess, O(reachable-commits) parse;
// for kernel-tree-sized repos under a second.
//
// AC-6 caller convention: the CLI gather layer at
// internal/cli/check/provenance.go::RunProvenanceCheck calls this
// exactly once per check invocation and passes the resulting map
// to RunIsolationEscape's cherryPicked parameter (replacing the
// G-0202 nil-placeholder).
func WalkCherryPicks(ctx context.Context, root string) map[string]bool {
	if root == "" || !hasGitCommits(ctx, root) {
		return nil
	}
	const fieldSep = "\x1f"
	const recSep = "\x1e"
	// One `git log` invocation; per-commit emit:
	//   <SHA><US><author-email><US><committer-email><US><body><RS>
	// Using ASCII control chars (US=\x1f, RS=\x1e) that don't
	// appear in commit content. The recSep needs no trailing
	// newline because we split on the raw \x1e and never depend on
	// surrounding whitespace.
	cmd := exec.CommandContext(ctx, "git", "log",
		"--pretty=format:%H"+fieldSep+"%ae"+fieldSep+"%ce"+fieldSep+"%B"+recSep,
		"HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	cherryPicks := map[string]bool{}
	for _, rec := range strings.Split(string(out), recSep) {
		// Trim only whitespace that could land between records via
		// git log's own formatting (no leading newline after the
		// first record because `format` adds no terminators of its
		// own; defensive trim anyway).
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		fields := strings.SplitN(rec, fieldSep, 4)
		if len(fields) < 4 {
			continue
		}
		sha := strings.TrimSpace(fields[0])
		authorEmail := strings.TrimSpace(fields[1])
		committerEmail := strings.TrimSpace(fields[2])
		body := fields[3]
		// Both signals required: (b) gap AND (a) marker.
		if sha == "" || authorEmail == "" || committerEmail == "" {
			continue
		}
		if authorEmail == committerEmail {
			continue
		}
		if !cherryPickedMarkerRE.MatchString(body) {
			// Gap exists (e.g., an amended commit with --reset-author
			// or an upstream rewrite) but no marker line: not a
			// `cherry-pick -x` shape per the rule's docstring. The
			// rule continues to fire.
			continue
		}
		cherryPicks[sha] = true
	}
	return cherryPicks
}
