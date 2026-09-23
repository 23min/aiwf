package projectguidance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/23min/aiwf/internal/gitops"
)

// Snapshot holds selected documents and catalogue metadata from one exact commit.
type Snapshot struct {
	Commit    string
	Catalogue Catalogue
	Documents map[string][]byte
}

// Retrieve downloads and validates the current source without retaining a cache.
// Only selected documents are read; all catalogue metadata is validated.
func Retrieve(ctx context.Context, source string, selected []string) (*Snapshot, error) {
	return retrieve(ctx, source, selected, "")
}

func retrieve(ctx context.Context, source string, selected []string, tempParent string) (*Snapshot, error) {
	return retrieveWithSelection(ctx, source, func(Catalogue) ([]string, error) { return selected, nil }, tempParent)
}

// RetrieveWithSelection keeps one source revision available while the caller
// chooses packs. Only the chosen documents are validated and returned.
func RetrieveWithSelection(ctx context.Context, source string, selectPacks func(Catalogue) ([]string, error)) (*Snapshot, error) {
	return retrieveWithSelection(ctx, source, selectPacks, "")
}

func retrieveWithSelection(ctx context.Context, source string, selectPacks func(Catalogue) ([]string, error), tempParent string) (snapshot *Snapshot, err error) {
	temp, err := os.MkdirTemp(tempParent, "aiwf-guidance-")
	if err != nil {
		return nil, fmt.Errorf("creating temporary guidance directory: %w", err)
	}
	defer func() {
		if cleanupErr := os.RemoveAll(temp); cleanupErr != nil {
			err = errors.Join(err, fmt.Errorf("removing temporary guidance directory %s: %w", temp, cleanupErr))
			snapshot = nil
		}
		if ctx.Err() != nil {
			err = errors.Join(err, ctx.Err())
			snapshot = nil
		}
	}()
	repo := filepath.Join(temp, "source")
	if err = gitops.CloneDefault(ctx, source, repo); err != nil {
		return nil, fmt.Errorf("fetching guidance from %q; check the source and Git access: %w", source, err)
	}
	commit, err := gitops.ResolveCommitSHA(ctx, repo, "HEAD")
	if err != nil {
		return nil, fmt.Errorf("resolving guidance source revision: %w", err)
	}
	raw, err := gitops.ReadRegularFromHEAD(ctx, repo, "catalogue.json")
	if err != nil {
		return nil, fmt.Errorf("%w: reading catalogue.json: %w", ErrInvalidSource, err)
	}
	catalogue, err := parseCatalogue(raw)
	if err != nil {
		return nil, fmt.Errorf("validating catalogue.json; repair the source catalogue: %w", err)
	}
	selected, err := selectPacks(catalogue)
	if err != nil {
		return nil, fmt.Errorf("selecting project guidance: %w", err)
	}
	available := make(map[string]Pack, len(catalogue.Packs))
	for _, pack := range catalogue.Packs {
		available[pack.ID] = pack
	}
	documents := make(map[string][]byte)
	seen := make(map[string]bool)
	for _, id := range selected {
		pack, ok := available[id]
		if !ok || seen[id] {
			return nil, fmt.Errorf("%w: selected pack %q is missing or repeated; choose each catalogue id once", ErrInvalidSource, id)
		}
		seen[id] = true
		for _, name := range pack.Files {
			content, err := gitops.ReadRegularFromHEAD(ctx, repo, name)
			if err != nil {
				return nil, fmt.Errorf("%w: reading pack %q document %q: %w", ErrInvalidSource, id, name, err)
			}
			if len(bytes.TrimSpace(content)) == 0 || !utf8.Valid(content) || bytes.ContainsRune(content, 0) {
				return nil, fmt.Errorf("%w: document %q must contain nonempty UTF-8 Markdown text; repair the source document", ErrInvalidSource, name)
			}
			documents[name] = content
		}
	}
	return &Snapshot{Commit: commit, Catalogue: catalogue, Documents: documents}, nil
}
