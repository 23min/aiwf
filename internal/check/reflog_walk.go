package check

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/23min/aiwf/internal/gitops"
)

// reflog_walk.go — M-0161/AC-5 (G-0205) gather-layer helper.
//
// WalkOrphanedAICommits walks the reflog of every ritual
// branch and surfaces AI-actor commits orphaned by non-fast-
// forward updates (the canonical "force-push" shape).
//
// The kernel cannot determine from the orphan alone whether the
// AI commit was on the correct branch at the time of force-push
// — the rewrite removed that audit trail. So the rule surfaces
// the orphan for operator review (isolation-escape-orphaned-ai-
// commit warning) rather than auto-classifying as escape.

// OrphanedAICommit is one AI-actor commit that was orphaned by
// a non-fast-forward update on a ritual branch.
//
// SHA is the orphan's commit SHA (preserved in the local object
// store regardless of the ref's current value).
// Branch is the ritual branch whose reflog records the
// orphaning update.
// ReflogDate is the human-readable date of the reflog entry
// recording the orphaning update.
// EntityID and Actor are the aiwf-entity: and aiwf-actor:
// trailer values from the orphan's commit body.
type OrphanedAICommit struct {
	SHA        string
	Branch     string
	ReflogDate string
	EntityID   string
	Actor      string
}

// WalkOrphanedAICommits walks the reflog of every ritual
// branch under refs/heads/, identifies non-fast-forward updates
// (the previous tip was not an ancestor of the new tip — the
// canonical force-push shape), reads trailers from each
// orphaned tip, and returns one OrphanedAICommit per orphan
// that carries aiwf-actor: ai/... + aiwf-entity: <id>.
//
// Algorithm (per ritual ref):
//
//  1. `git reflog show <ref> --pretty=format:%H %gd` lists the
//     reflog entries newest-first.
//  2. Walk consecutive pairs (newer, older). older was the
//     ref's tip BEFORE the update that landed at newer.
//  3. If older is NOT an ancestor of newer (`git merge-base
//     --is-ancestor older newer` exits non-zero), the update
//     was a non-fast-forward and older was orphaned.
//  4. Read older's trailers via `git log -1 --pretty=%B older`.
//     If aiwf-actor starts with "ai/" AND aiwf-entity is
//     non-empty, surface the orphan.
//
// Returns nil on non-git directories, empty repos, and repos
// with no ritual refs. Reflog absence (e.g. core.logAllRefUpdates
// =false) is handled at the oracle layer (M-0161/AC-3 typed-
// error contract emits OracleErr with Capability "reflog-
// disabled" → isolation-escape-oracle-failure advisory); this
// walker simply finds no entries to inspect.
//
// Per-SHA deduplication: an orphan that appears across multiple
// ref reflogs (rare; would require the orphan tip to have been
// the tip on multiple refs at different times) surfaces once
// per (SHA, Branch) pair; the rule consumer deduplicates by
// SHA when emitting findings.
func WalkOrphanedAICommits(ctx context.Context, root string) []OrphanedAICommit {
	if root == "" {
		return nil
	}
	refs, err := listRitualHeads(ctx, root)
	if err != nil || len(refs) == 0 {
		return nil
	}
	var out []OrphanedAICommit
	for _, ref := range refs {
		entries, err := reflogEntries(ctx, root, ref)
		if err != nil || len(entries) < 2 {
			continue
		}
		// Walk consecutive (newer, older) pairs. entries[0] is
		// the most recent; entries[i+1] was the ref's tip
		// before entries[i] landed.
		for i := 0; i < len(entries)-1; i++ {
			newer := entries[i]
			older := entries[i+1]
			if older.SHA == "" || newer.SHA == "" || older.SHA == newer.SHA {
				continue
			}
			if isAncestor(ctx, root, older.SHA, newer.SHA) {
				continue // fast-forward; not orphaned
			}
			actor, entity := readActorAndEntity(ctx, root, older.SHA)
			if !strings.HasPrefix(actor, "ai/") || entity == "" {
				continue
			}
			out = append(out, OrphanedAICommit{
				SHA:        older.SHA,
				Branch:     ref,
				ReflogDate: older.Date,
				EntityID:   entity,
				Actor:      actor,
			})
		}
	}
	return out
}

// listRitualHeads returns the local-branch set under
// refs/heads/ filtered to ritual shapes (epic/M-NNNN-...,
// milestone/M-NNNN-..., patch/...) plus main. Matches the
// existing oracle's filter (intentionally; same set of refs
// the rule polices).
func listRitualHeads(ctx context.Context, root string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "for-each-ref", "refs/heads/", "--format=%(refname:short)")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var ritual []string
	for _, line := range strings.Split(string(out), "\n") {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		// Same filter the oracle uses (internal/cli/check/
		// isolation_escape_oracle.go::listRitualBranches).
		// Duplicated here to keep this helper standalone — the
		// canonical filter is the oracle's; if the shape
		// changes, both update together.
		if name == "main" || ritualShape(name) {
			ritual = append(ritual, name)
		}
	}
	return ritual, nil
}

// ritualShape reports whether the branch name sits in an aiwf
// ritual namespace (epic/, milestone/, patch/). This is a
// deliberately loose prefix-only check — looser than
// branchparse.ParseEntityFromBranch, which since G-0198 also
// requires a coherent id segment (epic/E-, milestone/M-, patch/g-).
// The reflog orphan-walk wants to scan every ritual-namespace branch
// for force-pushed-away AI commits, including malformed ones
// (e.g. epic/typo), so it intentionally does not require a parseable
// id here. Kept inline rather than importing branchparse so the
// helper stays leaf relative to the kernel rule.
func ritualShape(branch string) bool {
	for _, prefix := range []string{"epic/", "milestone/", "patch/"} {
		if strings.HasPrefix(branch, prefix) {
			return true
		}
	}
	return false
}

// reflogEntry is one record from `git reflog show <ref>`.
type reflogEntry struct {
	SHA  string
	Date string
}

// reflogEntries returns the reflog of ref newest-first. The
// "%gd" placeholder emits the reflog selector (e.g.
// "epic/E-0001-engine@{0}") with the date appended by
// `--date=iso`; we extract the date portion for the hint text.
//
// Empty reflog (e.g., a freshly-created branch with no update
// history yet, OR core.logAllRefUpdates=false) returns
// (nil, nil) — the helper proceeds to the next ref.
func reflogEntries(ctx context.Context, root, ref string) ([]reflogEntry, error) {
	cmd := exec.CommandContext(ctx, "git", "reflog", "show", ref, "--date=iso", "--pretty=format:%H%x1f%gd")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		// git reflog show returns 0 even on empty reflog; a
		// non-zero exit here is a real error (e.g., ref not
		// found). Treat as "no reflog data" and continue.
		return nil, err
	}
	var entries []reflogEntry
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\x1f", 2)
		if len(parts) < 2 {
			continue
		}
		sha := strings.TrimSpace(parts[0])
		// parts[1] is the reflog-selector with the date appended
		// by --date=iso, e.g.:
		//   "epic/E-0001-engine@{2026-06-04 14:30:00 +0000}"
		// Extract the date from inside the braces.
		date := ""
		if openBrace := strings.Index(parts[1], "{"); openBrace >= 0 {
			if closeBrace := strings.LastIndex(parts[1], "}"); closeBrace > openBrace {
				date = parts[1][openBrace+1 : closeBrace]
			}
		}
		entries = append(entries, reflogEntry{SHA: sha, Date: date})
	}
	return entries, nil
}

// isAncestor reports whether old is reachable from new via
// any path. Used to distinguish fast-forward (old IS ancestor;
// not orphaned) from non-fast-forward (old NOT ancestor;
// orphaned).
func isAncestor(ctx context.Context, root, old, newer string) bool {
	cmd := exec.CommandContext(ctx, "git", "merge-base", "--is-ancestor", old, newer)
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// readActorAndEntity reads the aiwf-actor: and aiwf-entity:
// trailer values from the commit at sha. Returns ("", "") if
// the commit is unreadable or if the trailers are absent.
//
// gitops.ParseTrailers is the canonical trailer parser; we
// pull the full commit message and feed it through.
func readActorAndEntity(ctx context.Context, root, sha string) (actor, entity string) {
	cmd := exec.CommandContext(ctx, "git", "log", "-1", "--pretty=%B", sha)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "", ""
	}
	trailers := gitops.ParseTrailers(string(out))
	for _, tr := range trailers {
		switch tr.Key {
		case gitops.TrailerActor:
			actor = tr.Value
		case gitops.TrailerEntity:
			entity = tr.Value
		}
	}
	return actor, entity
}

// RunOrphanedAICommits emits one CodeIsolationEscapeOrphanedAICommit
// warning per orphaned AI-actor commit, honoring per-SHA acks.
//
// orphans is the typed gather output from WalkOrphanedAICommits.
// ackedSHAs is the M-0159/AC-3 acknowledgment set; an entry in
// the map exempts that SHA from this rule the same way it
// exempts isolation-escape, id-rename-untrailered, etc.
//
// Per-SHA dedup: an orphan that surfaced on multiple branches
// (uncommon; would require the commit to have been the tip on
// more than one branch over time) fires once. The first
// (branch, date) pair seen names the finding.
func RunOrphanedAICommits(orphans []OrphanedAICommit, ackedSHAs map[string]bool) []Finding {
	if len(orphans) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var findings []Finding
	for _, o := range orphans {
		if o.SHA == "" {
			continue
		}
		if seen[o.SHA] {
			continue
		}
		seen[o.SHA] = true
		if ackedSHAs[o.SHA] {
			continue
		}
		findings = append(findings, Finding{
			Code:     CodeIsolationEscapeOrphanedAICommit.ID,
			Severity: SeverityWarning,
			Message: fmt.Sprintf(
				"AI-actor commit %s was orphaned by a non-fast-forward update on %q at %s; entity %s, actor %q",
				shortHash(o.SHA), o.Branch, o.ReflogDate, o.EntityID, o.Actor,
			),
			Hint: fmt.Sprintf(
				"inspect with `git reflog show %s | grep %s` and either restore the commit (`git update-ref refs/heads/%s <pre-push-sha>` or cherry-pick onto the correct branch) or, if the force-push was deliberate sovereign cleanup, document with `aiwf acknowledge illegal %s --reason \"...\"`",
				o.Branch, shortHash(o.SHA), o.Branch, shortHash(o.SHA),
			),
			EntityID: o.EntityID,
		})
	}
	return findings
}
