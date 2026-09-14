package check

import (
	"context"
	"fmt"
	"os/exec"
	"slices"

	codespkg "github.com/23min/aiwf/internal/codes"
	"github.com/23min/aiwf/internal/entity"
)

// CodeEntityBodySectionDropped is the typed kernel-code descriptor for the
// push-seam membership gate (ADR-0048 seam two).
//
// The write seams hold a body-supplying verb to the same rule, but a body can
// reach a commit without passing one: the wrap-milestone ritual commits the
// milestone spec with plain `git commit`. That commit carries the ritual's
// `aiwf-verb`, `aiwf-entity` and `aiwf-actor` trailers, so the untrailered
// audit beside this rule passes it — which is why membership cannot be
// inferred from trailer presence and needs a rule that reads the bytes.
//
// The code names a dropped section rather than an absent one. It fires only
// where a commit removed a section the body carried, never on an omission
// already present at the range base. Holding a push to completeness instead
// would refuse an author over an omission they did not introduce — the cost
// ADR-0048 measured at the edit seam and rejected — and would refuse the very
// commit `aiwf edit-body` had just made, since that verb permits an edit which
// keeps an existing omission. Nothing here converges the bodies committed
// without a section before the gate landed.
//
// Absence and emptiness keep separate codes per D-0090: the remedies are
// mutually escaping, so one hint cannot serve both.
var CodeEntityBodySectionDropped = codespkg.Code{
	ID:    "entity-body-section-dropped",
	Class: codespkg.ClassStructural,
}

// DroppedBodySection describes one required section an entity's body carried
// before a commit in the range and no longer carries at HEAD.
//
// Pre-computed by WalkDroppedBodySections; consumed by
// RunEntityBodySectionDropped. SHA is the commit that dropped the section, so
// a deliberate removal can be exempted through `aiwf acknowledge illegal` —
// the escape ADR-0048 names, and the only one, since no verb on this path
// carries `--force`.
type DroppedBodySection struct {
	SHA      string
	Path     string
	EntityID string
	Section  string
}

// RunEntityBodySectionDropped emits one error finding per record in dropped,
// minus those whose SHA appears in ackedSHAs.
//
// The rule is pure: WalkDroppedBodySections does the git work and the
// comparison; this maps records to findings, the same split
// RunIDRenameUntrailered uses.
func RunEntityBodySectionDropped(dropped []DroppedBodySection, ackedSHAs map[string]bool) []Finding {
	var out []Finding
	for _, d := range dropped {
		if ackedSHAs[d.SHA] {
			continue
		}
		out = append(out, Finding{
			Code:     CodeEntityBodySectionDropped.ID,
			Severity: SeverityError,
			Message: fmt.Sprintf(
				"commit %s drops required section %q from %s; the body carried it before and no longer does",
				shortHash(d.SHA), "## "+d.Section, d.EntityID,
			),
			Path:     d.Path,
			EntityID: d.EntityID,
		})
	}
	return out
}

// WalkDroppedBodySections compares each commit in the range against its first
// parent and returns the required sections it removed from an entity body,
// confirmed still absent at HEAD.
//
// Three shapes are deliberately not reported, each one a way a range carries a
// body its author did not write:
//
//   - An omission already present at the commit's parent. The author did not
//     introduce it, and refusing there is the cost ADR-0048 rejected.
//   - A body absorbed by an ordinary `--no-ff` merge. Its content traces to
//     commits on another branch, judged by that branch's own push. The
//     untrailered audit skips the same shape for the same reason.
//   - An entity the commit created. `aiwf add` already holds a create to
//     completeness and carries the sovereign `--force` that ADR-0048 makes a
//     standing exemption; re-judging the body here would revoke it.
//
// A drop a later commit in the range repaired is also not reported: the rule
// judges what the push publishes, and refusing over an intermediate state
// would leave the author rewriting history to satisfy it.
//
// Returns nil for every error condition — a missing blob, an unparseable
// entity file, a git subprocess failure. The benign fail-shut precedent is
// WalkUntrailedIDRenames': the gate is one rule among many, and a git hiccup
// should surface the others rather than abort the pass.
func WalkDroppedBodySections(ctx context.Context, root string, commits []UntrailedCommit) []DroppedBodySection {
	type candidate struct {
		sha, path, id, section string
	}
	var candidates []candidate
	for i := range commits {
		c := &commits[i]
		if len(c.ParentSHAs) == 0 || isAbsorbingMerge(c.ParentSHAs, c.Subject) {
			continue
		}
		for _, path := range c.Paths {
			kind, ok := entity.PathKind(path)
			if !ok {
				continue
			}
			// Every kind PathKind yields declares a non-empty set, so
			// there is no empty-set arm to guard; SectionsAbsent over an
			// empty want would return nothing regardless.
			want := entity.RequiredSections(kind)
			id, ok := entityIDFromPath(path)
			if !ok {
				continue
			}
			before, ok := bodyAtRev(ctx, root, c.ParentSHAs[0], path)
			if !ok {
				continue
			}
			after, ok := bodyAtRev(ctx, root, c.SHA, path)
			if !ok {
				continue
			}
			absentBefore := SectionsAbsent(before, want)
			for _, section := range SectionsAbsent(after, want) {
				if slices.Contains(absentBefore, section) {
					continue
				}
				candidates = append(candidates, candidate{c.SHA, path, id, section})
			}
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	// Confirm against HEAD, once per distinct path: a drop a later commit in
	// the range restored is not what the push publishes.
	absentAtHEAD := map[string][]string{}
	var out []DroppedBodySection
	for _, c := range candidates {
		still, seen := absentAtHEAD[c.path]
		if !seen {
			kind, _ := entity.PathKind(c.path)
			if body, ok := bodyAtRev(ctx, root, "HEAD", c.path); ok {
				still = SectionsAbsent(body, entity.RequiredSections(kind))
			}
			absentAtHEAD[c.path] = still
		}
		if !slices.Contains(still, c.section) {
			continue
		}
		out = append(out, DroppedBodySection{SHA: c.sha, Path: c.path, EntityID: c.id, Section: c.section})
	}
	return out
}

// bodyAtRev returns the post-frontmatter bytes of relPath as of rev. It
// reports false when the path does not exist there, or when what it holds is
// not an entity file — comparing frontmatter would make a status promote look
// like a body change, which is the scope this gate must stay out of.
func bodyAtRev(ctx context.Context, root, rev, relPath string) ([]byte, bool) {
	cmd := exec.CommandContext(ctx, "git", "show", rev+":"+relPath)
	cmd.Dir = root
	raw, err := cmd.Output()
	if err != nil {
		return nil, false
	}
	_, body, ok := entity.Split(raw)
	if !ok {
		return nil, false
	}
	return body, true
}
