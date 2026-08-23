package verb

import (
	"bytes"
	"fmt"
	"path/filepath"
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
// moves is always non-empty at every call site — Rename always produces
// at least its own entity's move, and Retitle and Archive each call this
// helper only inside their own non-empty check — so there is no
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
		// e.Path is where the body's destinations were written to resolve
		// from; linkingPath is where they must resolve from once this plan
		// lands. They differ only for an entity that is itself moving,
		// which is the case ADR-0046 covers. Slash-normalized because the
		// primitive compares against markdown destinations, which are
		// forward-slash regardless of host, while e.Path carries whatever
		// separator the loader's filepath.Rel produced.
		newBody := RewriteLinkDestinationsForMove(body, filepath.ToSlash(e.Path), linkingPath, moves)
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
