package verb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
)

// Apply executes a verb's Plan against the consumer repo at root: it
// runs every OpMove via `git mv`, every OpWrite directly to disk
// (creating parent directories as needed), stages the writes with
// `git add`, then creates the single commit with the plan's subject
// and trailers.
//
// Moves run before writes so that when a verb (notably reallocate)
// renames a file/dir and also rewrites files inside that dir, the
// writes land at the new locations. The orchestrator does not validate
// the plan further — that already happened inside the verb.
func Apply(ctx context.Context, root string, p *Plan) error {
	movedPaths := []string{}
	writtenPaths := []string{}

	// Phase 1: moves.
	for _, op := range p.Ops {
		if op.Type != OpMove {
			continue
		}
		if err := gitops.Mv(ctx, root, op.Path, op.NewPath); err != nil {
			return fmt.Errorf("git mv %s -> %s: %w", op.Path, op.NewPath, err)
		}
		movedPaths = append(movedPaths, op.NewPath)
	}

	// Phase 2: writes.
	for _, op := range p.Ops {
		if op.Type != OpWrite {
			continue
		}
		full := filepath.Join(root, op.Path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", filepath.Dir(op.Path), err)
		}
		if err := os.WriteFile(full, op.Content, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", op.Path, err)
		}
		writtenPaths = append(writtenPaths, op.Path)
	}

	// Stage writes (moves already staged themselves).
	if len(writtenPaths) > 0 {
		if err := gitops.Add(ctx, root, writtenPaths...); err != nil {
			return fmt.Errorf("git add: %w", err)
		}
	}

	// Re-stage moved paths in case write-after-move modified the moved file.
	if len(movedPaths) > 0 {
		if err := gitops.Add(ctx, root, movedPaths...); err != nil {
			return fmt.Errorf("git add (moved): %w", err)
		}
	}

	if err := gitops.Commit(ctx, root, p.Subject, p.Body, p.Trailers); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	return nil
}
