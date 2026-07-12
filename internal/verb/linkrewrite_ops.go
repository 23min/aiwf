package verb

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// planLinkRewriteWrites computes an OpWrite for every active entity
// outside exclude whose body contains a markdown link resolving to
// one of moves' From paths, recomputed against the final post-move
// layout via M-0245's RewriteLinkDestinations. exclude names entity
// paths (pre-move) the caller is already writing explicitly — e.g.
// `aiwf retitle` folds its own link rewrite into the same write that
// syncs its H1, so this helper must not also emit a competing write
// for that path. Already-archived entities are never linking-file
// candidates, mirroring `aiwf archive`'s own forget-by-default
// exclusion (ADR-0004).
//
// moves is always non-empty at both call sites — Rename always
// produces at least its own entity's move, and Retitle only calls
// this helper inside its own `len(moves) > 0` branch — so there is no
// empty-moves guard here.
func planLinkRewriteWrites(tr *tree.Tree, moves []EntityMove, exclude map[string]bool) ([]FileOp, error) {
	postMovePath := make(map[string]string, len(moves))
	for _, m := range moves {
		postMovePath[m.From] = m.To
	}

	var ops []FileOp
	for _, e := range tr.Entities {
		if exclude[e.Path] {
			continue
		}
		if entity.IsArchivedPath(e.Path) {
			continue
		}
		linkingPath := e.Path
		if to, ok := postMovePath[e.Path]; ok {
			linkingPath = to
		}
		body, err := readBody(tr.Root, e.Path)
		if err != nil { //coverage:ignore defensive: e.Path comes from the loaded tree, so the file is present; a read error needs the file to vanish mid-verb
			return nil, err
		}
		newBody := RewriteLinkDestinations(body, linkingPath, moves)
		if bytes.Equal(newBody, body) {
			continue
		}
		content, err := entity.Serialize(e, newBody)
		if err != nil { //coverage:ignore defensive: Serialize fails only on a malformed entity; e already round-tripped through the loader
			return nil, fmt.Errorf("serializing %s after link rewrite: %w", e.ID, err)
		}
		ops = append(ops, FileOp{Type: OpWrite, Path: linkingPath, Content: content})
	}
	sort.Slice(ops, func(i, j int) bool { return ops[i].Path < ops[j].Path })
	return ops, nil
}
