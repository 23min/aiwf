package verb

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/severity"
	"github.com/23min/aiwf/internal/tree"
)

// validateUserBodyBytes refuses user-supplied body content that begins
// with a YAML frontmatter delimiter (`---\n`). Concatenating such
// content with the verb's serialized frontmatter would produce a
// malformed double-block file the loader can't parse — better to
// refuse early with a clear message than to silently strip and
// surprise the user. Leading whitespace is trimmed before the check
// so users can't smuggle frontmatter past with a couple of newlines.
//
// Used by `aiwf add --body-file` (resolveAddBody) and `aiwf edit-body`
// (M-058) so both routes apply the same rule against the same shape.
func validateUserBodyBytes(body []byte) error {
	trimmed := bytes.TrimLeft(body, " \t\r\n")
	if bytes.HasPrefix(trimmed, []byte("---\n")) || bytes.HasPrefix(trimmed, []byte("---\r\n")) {
		return fmt.Errorf("body content begins with a frontmatter delimiter (---); pass body content only, not a full markdown file with its own frontmatter")
	}
	return nil
}

// pathInside reports whether the repo-relative path p is the directory
// dir or lives somewhere underneath it. Comparison is forward-slash so
// callers don't need to normalize.
func pathInside(p, dir string) bool {
	p = filepath.ToSlash(p)
	dir = filepath.ToSlash(dir)
	if p == dir {
		return true
	}
	return strings.HasPrefix(p, dir+"/")
}

// initialStatus is the status `aiwf add` assigns to a freshly-created
// entity. Each kind starts at the leftmost state of its FSM.
func initialStatus(k entity.Kind) entity.Status {
	switch k {
	case entity.KindEpic:
		return "proposed"
	case entity.KindMilestone:
		return "draft"
	case entity.KindADR:
		return "proposed"
	case entity.KindGap:
		return "open"
	case entity.KindDecision:
		return "proposed"
	case entity.KindContract:
		return "proposed"
	}
	return ""
}

// projectAdd returns a new tree value that includes e alongside all of
// t's existing entities. plannedPaths lists repo-relative
// (forward-slash) paths that the verb plans to write but hasn't yet,
// so disk-consulting checks can treat them as present. The original
// tree is not mutated.
func projectAdd(t *tree.Tree, e *entity.Entity, plannedPaths ...string) *tree.Tree {
	proj := *t
	proj.Entities = make([]*entity.Entity, len(t.Entities), len(t.Entities)+1)
	copy(proj.Entities, t.Entities)
	proj.Entities = append(proj.Entities, e)
	proj.PlannedFiles = withPlanned(t.PlannedFiles, plannedPaths)
	return &proj
}

// projectReplace returns a new tree value where the entity matching
// modified.ID is replaced with modified. If the id is not present,
// projectReplace returns the original tree unchanged.
func projectReplace(t *tree.Tree, modified *entity.Entity, plannedPaths ...string) *tree.Tree {
	proj := *t
	proj.Entities = make([]*entity.Entity, len(t.Entities))
	for i, e := range t.Entities {
		if e.ID == modified.ID {
			proj.Entities[i] = modified
			continue
		}
		proj.Entities[i] = e
	}
	proj.PlannedFiles = withPlanned(t.PlannedFiles, plannedPaths)
	return &proj
}

// withPlanned merges existing planned paths with new ones into a fresh
// map. Returns nil only when both inputs are empty.
func withPlanned(existing map[string]struct{}, additions []string) map[string]struct{} {
	if len(existing) == 0 && len(additions) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(existing)+len(additions))
	for k := range existing {
		out[k] = struct{}{}
	}
	for _, p := range additions {
		out[p] = struct{}{}
	}
	return out
}

// projectionFindings returns the findings introduced by going from
// `original` to `projected`: any finding present on `projected` whose
// equivalent does not appear on `original` is considered "introduced
// by this verb." Pre-existing tree problems unrelated to the verb's
// change do not block it; the user can see them via `aiwf check`.
//
// Equivalence is by code + subcode + path + entity-id + message.
// That's strict enough that "same kind of problem on a different
// entity" is treated as a new finding (which is the right call:
// adding an entity that triggers a new ids-unique conflict, even when
// the tree already had unrelated ids-unique conflicts, is still the
// verb's responsibility).
//
// Both sides carry the consumer's aiwf.yaml severity policy before the
// diff, so the guard decides against the same severities `aiwf check`
// will apply at the push boundary. Without it the guard reads every
// finding at its config-agnostic default, and a knob that escalates one
// to error is invisible to every verb — the verb reports success and
// commits a state the pre-push hook then refuses.
//
// Only the `post` side is load-bearing for the refusal, since severity
// is what HasErrors reads and findingKey does not carry it. The `pre`
// side matters for the *diff*: a pass that rewrites a finding's message
// changes its key, so escalating one side alone would make an unchanged
// finding read as introduced. Every such pass today covers a code
// skipDuringProjection excludes, which is what makes symmetry
// unobservable rather than unnecessary — lift that exclusion, or add a
// message-rewriting pass over a code the diff sees, and this line is
// the difference between a correct guard and one refusing verbs for
// findings they did not introduce.
func projectionFindings(original, projected *tree.Tree) []check.Finding {
	pre := check.Run(original, nil)
	post := check.Run(projected, nil)
	policy := severity.Load(original.Root)
	severity.Apply(pre, policy, original)
	severity.Apply(post, policy, projected)
	seen := make(map[string]bool, len(pre))
	for i := range pre {
		seen[findingKey(&pre[i])] = true
	}
	var introduced []check.Finding
	for i := range post {
		if skipDuringProjection(post[i].Code) {
			// Codes the diff cannot attribute to this verb; see
			// skipDuringProjection for each one's reason and its gate.
			continue
		}
		if check.IsCrossBranchClassification(post[i]) {
			// ADR-0041: which refs carry a target is a fact about the
			// repository's refs, not about this verb's change, so it
			// never counts as introduced by it. A verb that renames or
			// moves the REFERENCING entity would otherwise look like
			// the cause: the finding's key carries the entity's path,
			// so the pre-change and post-change copies of one
			// unchanged cross-branch condition compare as different
			// findings. Enforcement is the push boundary's
			// (`aiwf check`), which sees the condition either way.
			continue
		}
		if !seen[findingKey(&post[i])] {
			introduced = append(introduced, post[i])
		}
	}
	return introduced
}

// blocksWrite reports whether fs carries an error the verb layer must
// refuse a write for: an error-severity finding that is not a
// cross-branch classification (see check.IsCrossBranchClassification).
//
// It is the gate for the body-prose scans each body-supplying verb runs
// against its planned-write bytes. projectionFindings applies the same
// exclusion itself, so its callers can keep using check.HasErrors.
func blocksWrite(fs []check.Finding) bool {
	for i := range fs {
		if fs[i].Severity == check.SeverityError && !check.IsCrossBranchClassification(fs[i]) {
			return true
		}
	}
	return false
}

// entityWrite carries the commit-shaping fields of a single-entity
// write — everything the Plan needs that isn't derived from the entity
// bytes themselves. Bundled into a struct so planEntityWrite reads at
// its call sites without a long positional parameter list.
type entityWrite struct {
	subject  string           // commit subject line
	body     string           // commit-message body paragraph; "" for none
	trailers []gitops.Trailer // audit-trail trailers
	metadata map[string]any   // Result.Metadata (JSON envelope fields)
}

// planEntityWrite is the shared tail of every single-entity, single-file
// mutating verb: it serializes `modified` with the already-computed
// `fileBody`, runs the projection safety-net (refusing with findings if
// the change would introduce an error-severity check finding), and
// returns a Plan writing exactly one OpWrite to `path`.
//
// The body-reading and body-transforming steps above it vary per verb
// (a plain readBody, an AC-heading rewrite, a batch upsert), so they
// stay at the call sites; this helper begins at Serialize.
//
// Verbs that deliberately skip the projection net — set-priority and
// set-area, whose single-scalar edits are already guarded at verb time
// and whose --clear paths must be allowed to write a state the standing
// check then flags (see their own notes) — do NOT route through here.
func planEntityWrite(t *tree.Tree, modified *entity.Entity, path string, fileBody []byte, w entityWrite) (*Result, error) {
	content, err := entity.Serialize(modified, fileBody)
	if err != nil {
		//coverage:ignore defensive: Serialize fails only on a malformed entity; `modified` round-tripped through the loader, so no realistic unit-test trigger. Consolidates the per-verb serialize-error guards this helper replaced.
		return nil, fmt.Errorf("serializing %s: %w", modified.ID, err)
	}
	proj := projectReplace(t, modified, filepath.ToSlash(path))
	if fs := projectionFindings(t, proj); check.HasErrors(fs) {
		return findings(fs), nil
	}
	result := plan(&Plan{
		Subject:  w.subject,
		Body:     w.body,
		Trailers: w.trailers,
		Ops:      []FileOp{{Type: OpWrite, Path: path, Content: content}},
	})
	result.Metadata = w.metadata
	return result, nil
}

// skipDuringProjection reports whether a finding code should be
// filtered out of the projectionFindings diff. Each code here is one
// the diff cannot attribute to the verb under it, for a reason of its
// own:
//
//   - body-prose-id reads body content from disk; projection models
//     update in-memory entity frontmatter but don't reflect verb-side
//     body content changes (e.g. add --body-file's not-yet-written
//     file, edit-body's new content, reallocate's prose rewrites).
//     Each body-supplying verb runs its own verb-time ScanBodyProseID
//     against the planned-write bytes directly (see add.go /
//     editbody.go / import.go / reallocate.go), which IS the gate.
//   - archive-sweep-pending is the per-tree aggregate, and its message
//     names the pending count. Since the diff keys findings by message,
//     every verb that moves an entity to or from a terminal status
//     re-keys the aggregate, and on a tree already past its declared
//     `archive.sweep_threshold` each subsequent terminal promote reads
//     as the mutation that introduced the breach. That is an instance
//     of the message-keyed identity defect G-0574 states in general;
//     excluding this one code is a point mitigation, not the decided
//     identity G-0574 asks for. It is safe here on its own terms: the
//     sweep ceiling is a tree-level drift control whose remedy is
//     `aiwf archive --apply`, a different verb over a different entity
//     set, and enforcement stays with `aiwf check`, which sees the
//     condition whichever verb preceded it. Same unattributability
//     ADR-0041 settles for the cross-branch classifications the diff
//     excludes just below.
func skipDuringProjection(code string) bool {
	return code == check.CodeBodyProseID || code == check.CodeArchiveSweepPending
}

func findingKey(f *check.Finding) string {
	return f.Code + "|" + f.Subcode + "|" + f.Path + "|" + f.EntityID + "|" + f.Message
}
