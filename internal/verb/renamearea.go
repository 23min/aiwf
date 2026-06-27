package verb

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// RenameArea renames a declared workstream area (E-0044, M-0177). It
// rewrites the `areas.members` entry in aiwf.yaml from oldName to
// newName AND rewrites the `area:` frontmatter of every entity tagged
// oldName to newName, in one atomic commit — the same referential-
// integrity discipline `aiwf reallocate` applies to ids. Renaming an
// area by hand-editing aiwf.yaml would orphan every entity still
// carrying the old value (the `area-unknown` finding flags them and the
// grouping view buckets them into the untagged complement); this verb
// closes that hole.
//
// members and defaultLabel are the consumer's declared areas (the
// validated single source of truth from config.Load, passed by the CLI
// layer); the verb never invents members. doc is the parsed aiwf.yaml
// the CLI loads for the comment-preserving splice, mirroring how
// ContractBind receives its aiwfyaml.Doc.
//
// Validation (no Plan on failure, so nothing is written):
//   - oldName and newName are non-empty and distinct;
//   - oldName is a declared member;
//   - newName is NOT already a declared member.
//
// The commit carries `aiwf-verb: rename-area`, one `aiwf-entity:`
// trailer per rewritten entity (sorted by id for determinism), and
// `aiwf-actor:`. The `aiwf-verb` trailer suppresses the untrailered-
// entity audit; the per-entity trailers make the rename appear in each
// affected entity's `aiwf history`. When no entity references oldName,
// only the verb+actor trailers ride along (an aiwf.yaml-only change).
//
// What undoes this? The same verb with swapped args: after
// `rename-area platform infra`, `rename-area infra platform` restores
// the prior member name and every entity tag.
func RenameArea(
	ctx context.Context,
	t *tree.Tree,
	doc *aiwfyaml.Doc,
	members []config.Member,
	defaultLabel, oldName, newName, actor string,
) (*Result, error) {
	_ = ctx
	if doc == nil {
		return nil, fmt.Errorf("aiwf.yaml not found; run 'aiwf init' first")
	}

	oldName = strings.TrimSpace(oldName)
	newName = strings.TrimSpace(newName)
	if oldName == "" || newName == "" {
		return nil, fmt.Errorf("rename-area requires a non-empty <old> and <new>")
	}
	if oldName == newName {
		return nil, fmt.Errorf("rename-area: <old> and <new> are identical (%q); nothing to rename", oldName)
	}

	names := make([]string, len(members))
	declared := make(map[string]bool, len(members))
	for i, m := range members {
		names[i] = m.Name
		declared[m.Name] = true
	}
	if !declared[oldName] {
		return nil, fmt.Errorf("area %q is not a declared member; declared areas: %s", oldName, declaredList(names))
	}
	if declared[newName] {
		return nil, fmt.Errorf("area %q is already a declared member; declared areas: %s", newName, declaredList(names))
	}

	// Rewrite the member set, preserving display order AND each member's
	// paths — only the renamed entry's name changes, in place. Map to the
	// aiwfyaml writer's local member shape at the splice (config stays the
	// single source of truth; aiwfyaml keeps its zero-dep-on-config layering).
	next := make([]aiwfyaml.AreaMember, len(members))
	for i, m := range members {
		name := m.Name
		if name == oldName {
			name = newName
		}
		next[i] = aiwfyaml.AreaMember{Name: name, Paths: m.Paths}
	}
	if err := doc.SetAreas(next, defaultLabel); err != nil {
		return nil, fmt.Errorf("updating aiwf.yaml: %w", err)
	}

	// One OpWrite for the rewritten aiwf.yaml, then one OpWrite per
	// entity whose effective area is oldName. Entities are sorted by id
	// so the trailer order (and the commit's file set) is deterministic.
	ops := []FileOp{{Type: OpWrite, Path: config.FileName, Content: doc.Bytes()}}

	var rewritten []*entity.Entity
	for _, e := range t.Entities {
		if e.Area == oldName {
			rewritten = append(rewritten, e)
		}
	}
	sort.Slice(rewritten, func(i, j int) bool { return rewritten[i].ID < rewritten[j].ID })

	trailers := []gitops.Trailer{{Key: gitops.TrailerVerb, Value: "rename-area"}}
	for _, e := range rewritten {
		modified := *e
		modified.Area = newName
		body, err := readBody(t.Root, e.Path)
		if err != nil {
			return nil, err
		}
		content, err := entity.Serialize(&modified, body)
		if err != nil { //coverage:ignore yaml.Marshal of a loaded, valid Entity does not fail; defensive, mirrors the reallocate serialize path
			return nil, fmt.Errorf("serializing %s after area rewrite: %w", e.ID, err)
		}
		ops = append(ops, FileOp{Type: OpWrite, Path: e.Path, Content: content})
		trailers = append(trailers, gitops.Trailer{Key: gitops.TrailerEntity, Value: entity.Canonicalize(e.ID)})
	}
	trailers = append(trailers, gitops.Trailer{Key: gitops.TrailerActor, Value: actor})

	return plan(&Plan{
		Subject:  fmt.Sprintf("aiwf rename-area %s -> %s", oldName, newName),
		Trailers: trailers,
		Ops:      ops,
	}), nil
}

// declaredList renders the declared member set for an operator-facing
// error. Empty (no areas declared) reads as "(none)" so the message is
// self-explaining when the consumer never declared an areas block.
func declaredList(members []string) string {
	if len(members) == 0 {
		return "(none)"
	}
	return strings.Join(members, ", ")
}
