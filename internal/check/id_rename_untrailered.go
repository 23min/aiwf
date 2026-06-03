package check

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	codespkg "github.com/23min/aiwf/internal/codes"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
)

// CodeIDRenameUntrailered is the typed kernel-code descriptor for
// the id-rename-untrailered finding (M-0160/AC-4). The finding
// fires when a commit between merge-base(HEAD, trunk) and HEAD
// renames an id-bearing entity file AND lacks an `aiwf-verb`
// trailer in the rename-class closed set (retitle / rename /
// reallocate / archive / move per `renameClassVerbs` below,
// mirrored from internal/gitops/refs.go::renameVerbs).
//
// Catches the operator-discipline gap CLAUDE.md
// §"Id-collision resolution at merge time" documents: an operator
// resolves a trunk-collision via inline `git mv` instead of
// `aiwf reallocate <id-or-path>`. The immediate trunk-collision
// finding clears (git's rename detection paired the move via
// G-0167's trailer-driven path or G-0109's cumulative-similarity
// fallback), but the kernel trailer history misses the renumber
// event: `aiwf history G-old` doesn't bridge to the new id,
// cross-references in body prose aren't rewritten, and any future
// check rule keyed on `aiwf-verb: reallocate` doesn't see the
// rename.
//
// The code carries [codes.ClassBranchChoreography] — the layer-4
// kernel carve-out per ADR-0011 — so the rule joins the
// branch-policing finding set alongside isolation-escape, which
// polices the AI-actor-on-wrong-branch shape of the same
// trunk-collision-resolution discipline surface.
//
// Severity is warning at first land (per the M-0106 / G-0150
// chokepoint-rule precedent); a future decision (recorded as a
// D-NNN) may tighten to error after one epic of usage. This
// milestone does not pre-commit the tightening timing.
//
// Closes the CLAUDE.md §Id-collision chokepoint hint.
var CodeIDRenameUntrailered = codespkg.Code{
	ID:    "id-rename-untrailered",
	Class: codespkg.ClassBranchChoreography,
}

// UntrailedIDRename describes a single id-bearing file rename
// occurring on a commit between merge-base(HEAD, trunk) and HEAD
// whose `aiwf-verb:` trailer is NOT in the rename-class closed
// set. Pre-computed by the CLI gather layer via
// WalkUntrailedIDRenames; consumed by RunIDRenameUntrailered.
//
// OldID and NewID are extracted from the respective paths via
// entity.PathKind + entity.IDFromPath. For the typical
// slug-rename shape (retitle/rename done outside the verb path)
// OldID == NewID; for the rare hand-edited-frontmatter shape
// (operator manually changed `id:` AND did `git mv`) they may
// differ. The rule fires the same way either way; carrying both
// fields preserves the information for future tightening (e.g.,
// a future rule that distinguishes "untrailered slug rename" from
// "untrailered renumber").
type UntrailedIDRename struct {
	SHA     string
	OldPath string
	NewPath string
	OldID   string
	NewID   string
}

// The rename-class verb membership lives at gitops.IsRenameVerb
// (the M-0160/AC-4 REFACTOR export of internal/gitops/refs.go::
// renameVerbs). commitHasRenameClassVerb below consumes it. Before
// the export, this file carried a duplicated map by value — the
// reviewer N-2 drift hazard that the gitops export closed.

// RunIDRenameUntrailered emits one warning finding per record in
// renames, minus those whose SHA appears in ackedSHAs.
//
// The rule is pure: the gather layer (WalkUntrailedIDRenames) does
// all git work and filtering; the rule itself just maps records to
// findings. Same shape as the M-0159/AC-3 ack-helper-lift pattern:
// `ackedSHAs map[string]bool` carries the set of commits
// retroactively acknowledged via `aiwf acknowledge-illegal`; per-
// SHA closed-set scoping; nil or empty map is "no acknowledgments."
//
// Each record produces its own finding (no per-entity aggregation,
// mirrors M-0106/AC-10's per-commit firing). The finding's Path
// is the new (post-rename) path so the operator's editor can jump
// to the file that needs aiwf-reallocate retroactively; EntityID
// is the new id when known.
func RunIDRenameUntrailered(renames []UntrailedIDRename, ackedSHAs map[string]bool) []Finding {
	if len(renames) == 0 {
		return nil
	}
	var out []Finding
	for _, r := range renames {
		if ackedSHAs[r.SHA] {
			// M-0159/AC-3 — retroactive acknowledgment exempts this
			// commit. Same per-SHA closed-set semantics as the other
			// three ack-consuming rules (fsm-history-consistent,
			// isolation-escape, trailer-verb-unknown).
			continue
		}
		entityID := r.NewID
		if entityID == "" {
			entityID = r.OldID
		}
		out = append(out, Finding{
			Code:     CodeIDRenameUntrailered.ID,
			Severity: SeverityWarning,
			Message: fmt.Sprintf(
				"commit %s renames id-bearing entity file %s -> %s without an aiwf-verb trailer in the rename-class set (retitle/rename/reallocate/archive/move); inline `git mv` was likely used instead of the kernel verb. See CLAUDE.md §\"Id-collision resolution at merge time\".",
				shortHash(r.SHA), r.OldPath, r.NewPath),
			Path:     r.NewPath,
			EntityID: entityID,
		})
	}
	return out
}

// WalkUntrailedIDRenames walks merge-base(HEAD, ref)..HEAD for
// commits that rename id-bearing entity files WITHOUT an aiwf-verb
// trailer in the rename-class closed set. Returns one record per
// qualifying rename; multiple renames in one commit produce
// multiple records. Closes the gather-side seam for AC-4.
//
// Returns nil for all error conditions — ref empty, ref unresolved,
// no commits on HEAD, no common ancestor, transient git subprocess
// failure. Matches WalkCherryPicks's "benign fail-shut" precedent
// at internal/check/cherry_picks.go: the rule degrades to "no
// records" rather than blocking `aiwf check` on a git hiccup, since
// the chokepoint is one rule among many and a transient git
// failure should surface other findings rather than aborting the
// pass.
//
// Performance: one `git log` to enumerate the range with trailers,
// then one `git show -M --diff-filter=R` per untrailered commit
// to extract renames. Same shape as gitops.renamesFromAiwfVerbTrailers
// at the gather layer; for kernel-tree-sized branches the cost is
// well under a second.
func WalkUntrailedIDRenames(ctx context.Context, root, ref string) []UntrailedIDRename {
	if root == "" || ref == "" || !hasGitCommits(ctx, root) {
		return nil
	}
	// Find merge base; absence is benign.
	mbCmd := exec.CommandContext(ctx, "git", "merge-base", "HEAD", ref)
	mbCmd.Dir = root
	mbOut, err := mbCmd.Output()
	if err != nil {
		return nil
	}
	mergeBase := strings.TrimSpace(string(mbOut))
	if mergeBase == "" {
		return nil
	}

	// One git log to enumerate commits in mergeBase..HEAD with their
	// trailers. Format mirrors gitops.renamesFromAiwfVerbTrailers'
	// shape: COMMIT-marker + SHA + newline-separated trailer block,
	// terminated by END_COMMIT.
	const recordSeparator = "END_COMMIT"
	logCmd := exec.CommandContext(ctx, "git", "log", "--reverse",
		"--format=COMMIT %H%n%(trailers:only=true,unfold=true)"+recordSeparator,
		mergeBase+"..HEAD")
	logCmd.Dir = root
	logOut, err := logCmd.Output()
	if err != nil {
		return nil
	}

	var out []UntrailedIDRename
	for _, record := range strings.Split(string(logOut), recordSeparator) {
		record = strings.TrimSpace(record)
		if record == "" {
			continue
		}
		lines := strings.Split(record, "\n")
		if len(lines) == 0 || !strings.HasPrefix(lines[0], "COMMIT ") {
			continue
		}
		sha := strings.TrimSpace(strings.TrimPrefix(lines[0], "COMMIT "))
		if sha == "" {
			continue
		}
		trailers := gitops.ParseTrailers(strings.Join(lines[1:], "\n"))
		// Skip commits that have a rename-class aiwf-verb trailer —
		// they're the canonical-path renames the rule explicitly
		// exempts.
		if commitHasRenameClassVerb(trailers) {
			continue
		}
		// Get this commit's file renames via gitops.RenamesInCommit
		// (the M-0160/AC-4 REFACTOR export — replaces the duplicated
		// renamesInCommitForRule helper that mirrored the gitops
		// internal). Per-commit subprocess failure is silenced
		// consistently with the outer walker's fail-shut shape —
		// one transient hiccup in one commit shouldn't lose all
		// records from the range.
		commitRenames, rErr := gitops.RenamesInCommit(ctx, root, sha)
		if rErr != nil {
			continue
		}
		for oldPath, newPath := range commitRenames {
			// Filter to id-bearing entity files only. The rule's
			// claim is specifically about kernel entities; a
			// non-entity file rename (e.g., README.md -> DOCS.md)
			// is operator concern, not kernel concern.
			oldID, oldOK := entityIDFromPath(oldPath)
			newID, newOK := entityIDFromPath(newPath)
			if !oldOK && !newOK {
				continue
			}
			out = append(out, UntrailedIDRename{
				SHA:     sha,
				OldPath: oldPath,
				NewPath: newPath,
				OldID:   oldID,
				NewID:   newID,
			})
		}
	}
	return out
}

// commitHasRenameClassVerb returns true when the trailer set
// includes an `aiwf-verb:` entry whose value is in the
// rename-class closed set (gitops.IsRenameVerb). The closed-set
// authority lives in gitops; this helper just composes the
// trailer-key filter on top.
func commitHasRenameClassVerb(trailers []gitops.Trailer) bool {
	for _, tr := range trailers {
		if tr.Key != gitops.TrailerVerb {
			continue
		}
		if gitops.IsRenameVerb(tr.Value) {
			return true
		}
	}
	return false
}

// entityIDFromPath returns the kernel id for an id-bearing path,
// or "" + false for non-id-bearing paths. Bridges entity.PathKind
// + entity.IDFromPath into one call so callers don't carry both.
func entityIDFromPath(relPath string) (string, bool) {
	kind, ok := entity.PathKind(relPath)
	if !ok {
		return "", false
	}
	id, ok := entity.IDFromPath(relPath, kind)
	if !ok {
		return "", false
	}
	return id, true
}
