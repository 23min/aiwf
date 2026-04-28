package verb

import (
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
// file/dir to reflect the new id, and rewrites every reference field
// in every other entity that pointed to the old id. Body-prose
// references to the old id are surfaced as warnings (not rewritten).
//
// The argument may be an id (e.g., "M-007") when unambiguous, or a
// repo-relative path (e.g., "work/epics/E-01-platform/M-007-cache.md")
// when the id is duplicated — required after a merge collision where
// two files share the same id.
//
// The commit gets an aiwf-prior-entity: <old-id> trailer in addition
// to the standard three, so `aiwf history <old-id>` continues to find
// the entity's lifecycle even after the renumber.
func Reallocate(t *tree.Tree, idOrPath, actor string) (*Result, error) {
	target := resolveTarget(t, idOrPath)
	if target == nil {
		return nil, fmt.Errorf("entity %q not found by id or path", idOrPath)
	}

	oldID := target.ID
	newID := entity.AllocateID(target.Kind, t.Entities)

	source, dest, err := reallocatePaths(target, newID)
	if err != nil {
		return nil, err
	}
	newEntityPath := newEntityPathAfterRename(target, source, dest)

	// Modified entity: new id and new path.
	modified := *target
	modified.ID = newID
	modified.Path = newEntityPath

	// Rewrite references in every entity that points at the old id.
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

	// Body-prose mentions of the old id become warnings (not auto-rewrite).
	bodyFindings := scanBodyProse(t, target, oldID)

	// Plan paths (every file the verb intends to land):
	//   - the moved entity's new path
	//   - the contents of any directory it dragged along
	//   - the rewritten reference files (at their post-move paths)
	planned, err := plannedDestinations(t.Root, source, dest, newEntityPath)
	if err != nil {
		return nil, err
	}
	for _, rw := range rewrites {
		planned = append(planned, filepath.ToSlash(rw.entity.Path))
	}

	proj := projectReallocate(t, target, &modified, rewrites, planned)
	projFindings := projectionFindings(t, proj)
	if check.HasErrors(projFindings) {
		// Body-prose warnings are also worth surfacing alongside the projection errors.
		all := append([]check.Finding{}, projFindings...)
		all = append(all, bodyFindings...)
		return findings(all), nil
	}

	// Build the file ops:
	//   1. move the entity's file/dir (git mv preserves rename history)
	//   2. write the moved entity's file with the new id in frontmatter
	//   3. write each rewritten reference file
	ops := []FileOp{{Type: OpMove, Path: source, NewPath: dest}}

	movedBody, err := readBody(t.Root, target.Path)
	if err != nil {
		return nil, err
	}
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
		content, err := entity.Serialize(rw.entity, body)
		if err != nil {
			return nil, fmt.Errorf("serializing %s after reference rewrite: %w", rw.entity.ID, err)
		}
		ops = append(ops, FileOp{Type: OpWrite, Path: rw.entity.Path, Content: content})
	}

	subject := fmt.Sprintf("aiwf reallocate %s -> %s", oldID, newID)
	return &Result{
		Findings: bodyFindings, // warnings travel with a successful plan
		Plan: &Plan{
			Subject: subject,
			Trailers: []gitops.Trailer{
				{Key: "aiwf-verb", Value: "reallocate"},
				{Key: "aiwf-entity", Value: newID},
				{Key: "aiwf-prior-entity", Value: oldID},
				{Key: "aiwf-actor", Value: actor},
			},
			Ops: ops,
		},
	}, nil
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

// scanBodyProse walks every entity (except target) looking for the
// old id mentioned in body prose. Reports each as a warning so the
// human can decide whether to update the prose; the verb does not
// auto-rewrite text outside frontmatter.
func scanBodyProse(t *tree.Tree, target *entity.Entity, oldID string) []check.Finding {
	pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(oldID) + `\b`)
	var out []check.Finding
	for _, e := range t.Entities {
		if e == target {
			continue
		}
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
			out = append(out, check.Finding{
				Code:     "reallocate-body-reference",
				Severity: check.SeverityWarning,
				Message:  fmt.Sprintf("body still mentions %s; consider updating prose for clarity", oldID),
				Path:     e.Path,
				EntityID: e.ID,
				Hint:     check.HintFor("reallocate-body-reference", ""),
			})
		}
	}
	return out
}
