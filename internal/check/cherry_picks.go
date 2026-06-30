package check

import (
	"regexp"
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
//
// M-0216/AC-5: derives from the shared HEAD walk (head) — the
// author/committer emails and body it needs are carried on each
// HeadCommit, so the dedicated `git log HEAD` it used to spawn is
// gone. A nil/empty head yields nil (the prior "no commits" signal).
func WalkCherryPicks(head []HeadCommit) map[string]bool {
	if len(head) == 0 {
		return nil
	}
	cherryPicks := map[string]bool{}
	for i := range head {
		c := &head[i]
		// Both signals required: (b) identity gap AND (a) marker.
		if c.SHA == "" || c.AuthorEmail == "" || c.CommitterEmail == "" {
			continue
		}
		if c.AuthorEmail == c.CommitterEmail {
			continue
		}
		if !cherryPickedMarkerRE.MatchString(c.Body) {
			// Gap exists (e.g., an amended commit with --reset-author
			// or an upstream rewrite) but no marker line: not a
			// `cherry-pick -x` shape per the rule's docstring. The
			// rule continues to fire.
			continue
		}
		cherryPicks[c.SHA] = true
	}
	return cherryPicks
}
