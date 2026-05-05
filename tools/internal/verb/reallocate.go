package verb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/23min/ai-workflow-v2/tools/internal/check"
	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
)

// Reallocate gives an entity a new id of the same kind, renames its
// file/dir to reflect the new id, and rewrites every reference to
// the old id — both in frontmatter and in body prose — across the
// whole tree, including the entity's own body.
//
// The reference grammar (E-NN, M-NNN, ADR-NNNN, G-NNN, D-NNN, C-NNN)
// is regular and unambiguous, so prose rewriting is mechanical and
// safe. Word boundaries prevent false matches against longer ids
// (e.g., reallocating M-001 leaves M-0010 untouched).
//
// The argument may be an id (e.g., "M-007") when unambiguous, or a
// repo-relative path (e.g., "work/epics/E-01-platform/M-007-cache.md")
// when the id is duplicated — required after a merge collision where
// two files share the same id.
//
// The commit gets an aiwf-prior-entity: <old-id> trailer in addition
// to the standard three, so `aiwf history <old-id>` continues to find
// the entity's lifecycle even after the renumber.
func Reallocate(ctx context.Context, t *tree.Tree, idOrPath, actor string) (*Result, error) {
	target, err := resolveReallocateTarget(ctx, t, idOrPath)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, fmt.Errorf("entity %q not found by id or path", idOrPath)
	}

	oldID := target.ID
	newID := entity.AllocateID(target.Kind, t.Entities, t.TrunkIDStrings())

	source, dest, err := reallocatePaths(target, newID)
	if err != nil {
		return nil, err
	}
	newEntityPath := newEntityPathAfterRename(target, source, dest)

	// Modified entity: new id and new path. The lineage list grows
	// by appending the OLD id (not the new one), oldest-first; tree-
	// only readers (HTML render, aiwf show, future projections) can
	// read the chain straight from frontmatter without shelling out
	// to git log.
	modified := *target
	modified.ID = newID
	modified.Path = newEntityPath
	modified.PriorIDs = appendPriorID(target.PriorIDs, oldID)

	// Rewrite frontmatter references in every entity that points at
	// the old id.
	rewrites := rewriteReferences(t.Entities, target, oldID, newID)

	// For entities that live inside the moved directory (e.g., a
	// milestone inside the epic dir we're renumbering), their file
	// arrives at a new path after `git mv`. Update the rewritten
	// entity's Path so the subsequent write lands at the new location;
	// reads still come from the original path.
	for _, rw := range rewrites {
		if pathInside(rw.original.Path, source) {
			rw.entity.Path = newEntityPathAfterRename(rw.entity, source, dest)
		}
	}

	// Discover every entity whose body prose mentions the old id.
	// These bodies will be rewritten in this same commit. The set
	// may include entities not in `rewrites` (which only covers
	// frontmatter changes), and may include the target itself.
	proseTouched, err := findProseMentions(t, oldID)
	if err != nil {
		return nil, err
	}

	// Plan paths (every file the verb intends to land):
	//   - the moved entity's new path
	//   - the contents of any directory it dragged along
	//   - the rewritten reference files (at their post-move paths)
	//   - any prose-only files (no frontmatter rewrite, body changes only)
	planned, err := plannedDestinations(t.Root, source, dest, newEntityPath)
	if err != nil {
		return nil, err
	}
	for _, rw := range rewrites {
		planned = append(planned, filepath.ToSlash(rw.entity.Path))
	}
	for _, e := range proseTouched {
		// pre-move path is enough for projection — pathInside checks
		// use the original path; final write uses the post-move path
		// computed below.
		planned = append(planned, filepath.ToSlash(e.Path))
	}

	proj := projectReallocate(t, target, &modified, rewrites, planned)
	projFindings := projectionFindings(t, proj)
	if check.HasErrors(projFindings) {
		return findings(projFindings), nil
	}

	// Track which entities will already be written for frontmatter
	// reasons, so the prose-only pass doesn't double-write.
	frontmatterRewrites := make(map[string]*rewriteRecord, len(rewrites))
	for i := range rewrites {
		frontmatterRewrites[rewrites[i].original.ID] = &rewrites[i]
	}

	idPattern := proseRewritePattern(oldID)

	// Build the file ops:
	//   1. move the entity's file/dir (git mv preserves rename history)
	//   2. write the moved entity's file with the new id in frontmatter
	//      and prose
	//   3. write each frontmatter-rewrite file (with prose rewritten too)
	//   4. write each prose-only file
	ops := []FileOp{{Type: OpMove, Path: source, NewPath: dest}}

	movedBody, err := readBody(t.Root, target.Path)
	if err != nil {
		return nil, err
	}
	movedBody = idPattern.ReplaceAll(movedBody, []byte(newID))
	movedContent, err := entity.Serialize(&modified, movedBody)
	if err != nil {
		return nil, fmt.Errorf("serializing reallocated %s: %w", newID, err)
	}
	ops = append(ops, FileOp{Type: OpWrite, Path: newEntityPath, Content: movedContent})

	for _, rw := range rewrites {
		// Read from the pre-move path; write to the post-move path.
		// They differ only when the rewritten entity lives inside the
		// moved directory.
		body, err := readBody(t.Root, rw.original.Path)
		if err != nil {
			return nil, err
		}
		body = idPattern.ReplaceAll(body, []byte(newID))
		content, err := entity.Serialize(rw.entity, body)
		if err != nil {
			return nil, fmt.Errorf("serializing %s after reference rewrite: %w", rw.entity.ID, err)
		}
		ops = append(ops, FileOp{Type: OpWrite, Path: rw.entity.Path, Content: content})
	}

	// Prose-only pass: entities with no frontmatter ref to oldID but
	// with body mentions. Read existing entity, rewrite body, serialize.
	for _, e := range proseTouched {
		if e == target {
			continue // target already handled above
		}
		if _, alreadyHandled := frontmatterRewrites[e.ID]; alreadyHandled {
			continue
		}
		body, err := readBody(t.Root, e.Path)
		if err != nil {
			return nil, err
		}
		body = idPattern.ReplaceAll(body, []byte(newID))
		writePath := e.Path
		if pathInside(e.Path, source) {
			writePath = newEntityPathAfterRename(e, source, dest)
		}
		content, err := entity.Serialize(e, body)
		if err != nil {
			return nil, fmt.Errorf("serializing %s after prose rewrite: %w", e.ID, err)
		}
		ops = append(ops, FileOp{Type: OpWrite, Path: writePath, Content: content})
	}

	subject := fmt.Sprintf("aiwf reallocate %s -> %s", oldID, newID)
	return &Result{
		Plan: &Plan{
			Subject: subject,
			Trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "reallocate"},
				{Key: gitops.TrailerEntity, Value: newID},
				{Key: gitops.TrailerPriorEntity, Value: oldID},
				{Key: gitops.TrailerActor, Value: actor},
			},
			Ops: ops,
		},
	}, nil
}

// proseRewritePattern returns a regex that matches the literal
// id at word boundaries. Word boundaries prevent false matches
// against longer ids (M-001 must not match M-0010 or M-0011).
func proseRewritePattern(id string) *regexp.Regexp {
	return regexp.MustCompile(`\b` + regexp.QuoteMeta(id) + `\b`)
}

// appendPriorID returns prior with id appended, unless id is already
// the last element (idempotent under double-reallocate edge cases or
// repaired hand-edits). Returns a fresh slice so callers don't share
// backing storage with the loaded entity.
func appendPriorID(prior []string, id string) []string {
	if len(prior) > 0 && prior[len(prior)-1] == id {
		out := make([]string, len(prior))
		copy(out, prior)
		return out
	}
	out := make([]string, len(prior), len(prior)+1)
	copy(out, prior)
	return append(out, id)
}

// resolveTarget interprets an argument as either an id or a
// repo-relative path. ID match takes priority; falls back to path
// match. Returns nil if neither matches.
func resolveTarget(t *tree.Tree, idOrPath string) *entity.Entity {
	if e := t.ByID(idOrPath); e != nil {
		return e
	}
	want := filepath.ToSlash(idOrPath)
	for _, e := range t.Entities {
		if filepath.ToSlash(e.Path) == want {
			return e
		}
	}
	return nil
}

// resolveReallocateTarget resolves idOrPath to the entity reallocate
// should renumber, applying the trunk-ancestry tiebreaker when the
// argument is a bare id and two entities share it.
//
// The tiebreaker rule (per id-allocation.md):
//
//   - If exactly one of the colliding entities' add commits is an
//     ancestor of the configured trunk ref, the side already in
//     trunk keeps the id and the OTHER side is the renumber target.
//     The team has been calling the trunk-side entity by that name.
//   - If both add commits are in trunk, or neither is, ancestry can't
//     decide. Reallocate refuses with an error explaining the
//     diagnostic and asking the operator to pass a path.
//
// When the argument is a path, or when ByIDAll returns one match, or
// when the tree carries no TrunkRef (sandbox repos with no remote),
// the verb falls back to the existing single-target resolution and
// the existing "ambiguous, pass a path" error if needed.
func resolveReallocateTarget(ctx context.Context, t *tree.Tree, idOrPath string) (*entity.Entity, error) {
	matches := t.ByIDAll(idOrPath)
	switch len(matches) {
	case 0:
		// Either a path, or a non-existent id. Path matching falls
		// through to resolveTarget (which also handles single-id case
		// when ByIDAll happens to be empty, e.g. an exact path match
		// that didn't pass id format).
		return resolveTarget(t, idOrPath), nil
	case 1:
		return matches[0], nil
	}

	// Two or more entities share the id. Apply the trunk-ancestry
	// tiebreaker if a trunk ref is in scope; otherwise fall back to
	// the "ambiguous, pass a path" error.
	if t.TrunkRef == "" {
		return nil, ambiguousIDErr(idOrPath, matches, "no trunk ref configured for this repo")
	}

	type ranked struct {
		e        *entity.Entity
		addSHA   string
		inTrunk  bool
		hasError bool
	}
	rs := make([]ranked, len(matches))
	for i, e := range matches {
		sha, err := gitops.AddCommitSHA(ctx, t.Root, e.Path)
		if err != nil {
			rs[i] = ranked{e: e, hasError: true}
			continue
		}
		rs[i] = ranked{e: e, addSHA: sha}
		if sha == "" {
			continue
		}
		anc, err := gitops.IsAncestor(ctx, t.Root, sha, t.TrunkRef)
		if err != nil {
			rs[i].hasError = true
			continue
		}
		rs[i].inTrunk = anc
	}

	for _, r := range rs {
		if r.hasError {
			return nil, ambiguousIDErr(idOrPath, matches,
				"could not determine trunk ancestry for at least one side")
		}
	}

	inTrunk := 0
	var loser *entity.Entity
	for _, r := range rs {
		if r.inTrunk {
			inTrunk++
			continue
		}
		loser = r.e
	}
	if inTrunk != len(rs)-1 {
		// Either both/all are in trunk (inTrunk == len(rs)) or none
		// are (inTrunk == 0). Neither lets us pick a unique loser.
		var diag string
		switch inTrunk {
		case 0:
			diag = "neither side is on " + t.TrunkRef
		case len(rs):
			diag = "all sides are already on " + t.TrunkRef
		default:
			diag = fmt.Sprintf("%d of %d sides are on %s", inTrunk, len(rs), t.TrunkRef)
		}
		return nil, ambiguousIDErr(idOrPath, matches, diag)
	}
	return loser, nil
}

// ambiguousIDErr formats the operator-facing error when reallocate
// cannot pick a single target by id alone. The message lists every
// colliding path so the operator knows which paths to choose
// between, plus a one-line diagnostic explaining why ancestry didn't
// resolve it.
func ambiguousIDErr(id string, matches []*entity.Entity, diagnostic string) error {
	paths := make([]string, len(matches))
	for i, e := range matches {
		paths[i] = filepath.ToSlash(e.Path)
	}
	return fmt.Errorf("id %q is ambiguous (%s); pass a path instead. Candidates: %s",
		id, diagnostic, strings.Join(paths, ", "))
}

// reallocatePaths returns (source, dest) for the move that renames an
// entity to its new id. Slug is preserved; only the id portion of the
// path changes. For dir-based kinds the dir moves; for file-based kinds
// the file moves.
func reallocatePaths(e *entity.Entity, newID string) (source, dest string, err error) {
	switch e.Kind {
	case entity.KindEpic, entity.KindContract:
		dir := filepath.Dir(e.Path)
		parent, oldName := filepath.Split(dir)
		newName, err := substituteID(oldName, newID)
		if err != nil {
			return "", "", err
		}
		parent = strings.TrimRight(parent, "/")
		return dir, filepath.Join(parent, newName), nil
	default:
		dir, oldName := filepath.Split(e.Path)
		newName, err := substituteID(strings.TrimSuffix(oldName, ".md"), newID)
		if err != nil {
			return "", "", err
		}
		dir = strings.TrimRight(dir, "/")
		return e.Path, filepath.Join(dir, newName+".md"), nil
	}
}

// substituteID replaces the "<prefix>-<digits>" portion of name with
// newID, preserving any trailing "-<slug>".
func substituteID(name, newID string) (string, error) {
	// Find the second hyphen — same shape as substituteSlug.
	first := strings.IndexByte(name, '-')
	if first < 0 {
		return "", fmt.Errorf("name %q has no id prefix", name)
	}
	second := strings.IndexByte(name[first+1:], '-')
	if second < 0 {
		return newID, nil
	}
	slug := name[first+1+second+1:]
	return newID + "-" + slug, nil
}

// rewriteRecord pairs the original entity with its updated copy so the
// projection and the file ops stay in sync.
type rewriteRecord struct {
	original *entity.Entity
	entity   *entity.Entity
}

// rewriteReferences walks every entity (except the one being
// reallocated) and rewrites any reference to oldID into newID. Only
// entities that actually changed appear in the result.
func rewriteReferences(entities []*entity.Entity, target *entity.Entity, oldID, newID string) []rewriteRecord {
	var out []rewriteRecord
	for _, e := range entities {
		if e == target {
			continue
		}
		modified, changed := rewriteEntityRefs(e, oldID, newID)
		if changed {
			out = append(out, rewriteRecord{original: e, entity: modified})
		}
	}
	return out
}

// rewriteEntityRefs returns a copy of e with every reference field
// (single or list) rewritten from oldID to newID. The bool reports
// whether any field actually changed; callers skip writes for entities
// that didn't reference the old id.
func rewriteEntityRefs(e *entity.Entity, oldID, newID string) (*entity.Entity, bool) {
	modified := *e
	changed := false

	if modified.Parent == oldID {
		modified.Parent = newID
		changed = true
	}
	if modified.SupersededBy == oldID {
		modified.SupersededBy = newID
		changed = true
	}
	if modified.DiscoveredIn == oldID {
		modified.DiscoveredIn = newID
		changed = true
	}
	if l, c := rewriteList(modified.DependsOn, oldID, newID); c {
		modified.DependsOn = l
		changed = true
	}
	if l, c := rewriteList(modified.Supersedes, oldID, newID); c {
		modified.Supersedes = l
		changed = true
	}
	if l, c := rewriteList(modified.AddressedBy, oldID, newID); c {
		modified.AddressedBy = l
		changed = true
	}
	if l, c := rewriteList(modified.RelatesTo, oldID, newID); c {
		modified.RelatesTo = l
		changed = true
	}
	return &modified, changed
}

// rewriteList substitutes every occurrence of oldID with newID inside
// a list field. Returns the (possibly new) slice and whether any
// element changed. The original slice is not mutated.
func rewriteList(s []string, oldID, newID string) ([]string, bool) {
	changed := false
	for _, v := range s {
		if v == oldID {
			changed = true
			break
		}
	}
	if !changed {
		return s, false
	}
	out := make([]string, len(s))
	for i, v := range s {
		if v == oldID {
			out[i] = newID
		} else {
			out[i] = v
		}
	}
	return out, true
}

// projectReallocate builds the projected tree for a reallocate verb:
// the renamed/renumbered entity replaces its original, and each
// rewriteRecord's modified entity replaces its original. PlannedFiles
// includes the moved entity's new location, any files swept along with
// a directory move, and the existing paths of every rewritten file.
func projectReallocate(t *tree.Tree, original, modified *entity.Entity, rewrites []rewriteRecord, planned []string) *tree.Tree {
	proj := *t
	proj.Entities = make([]*entity.Entity, len(t.Entities))
	for i, e := range t.Entities {
		if e == original {
			proj.Entities[i] = modified
			continue
		}
		proj.Entities[i] = e
	}
	for _, rw := range rewrites {
		for i, e := range proj.Entities {
			if e == rw.original {
				proj.Entities[i] = rw.entity
				break
			}
		}
	}
	proj.PlannedFiles = withPlanned(t.PlannedFiles, planned)
	return &proj
}

// findProseMentions returns every entity (including the target
// itself) whose body mentions oldID at a word boundary. The caller
// rewrites these bodies as part of the same commit.
//
// Read errors and frontmatter-split failures are silently skipped:
// the projection-level findings cover those cases, and a malformed
// file shouldn't stop the rewrite of the well-formed ones.
func findProseMentions(t *tree.Tree, oldID string) ([]*entity.Entity, error) {
	pattern := proseRewritePattern(oldID)
	var out []*entity.Entity
	for _, e := range t.Entities {
		full := filepath.Join(t.Root, e.Path)
		content, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		_, body, ok := entity.Split(content)
		if !ok {
			continue
		}
		if pattern.Find(body) != nil {
			out = append(out, e)
		}
	}
	return out, nil
}
