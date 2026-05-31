package check

import (
	"context"
	"errors"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// walkError records one read failure during the batched walk. The
// rule turns each into a `fsm-history-consistent/history-walk-error`
// finding so a transient subprocess error against one entity's blob
// surfaces visibly rather than silently wiping the rule's output
// (per CLAUDE.md §Engineering principles — "Errors are findings, not
// parse failures.").
//
// Side is "commit" when the failure was reading the status at the
// touched commit's path; "parent" when reading the parent's path
// for the prior-status comparison. Parent is the parent SHA being
// compared against (empty for commit-side errors). EntityID names
// the entity whose walk hit the failure.
type walkError struct {
	EntityID string
	Commit   string
	Parent   string
	Path     string
	Side     string
	Err      error
}

// batchedWalkStatusChanges enumerates DAG-aware status-change
// observations across every entity in t via the M-0137 batched
// helpers (gitops.BulkRevwalk + the blobReader dep). Returns:
//
//   - observations: per (entity, commit, parent) tuples where the
//     entity's status differs between the parent and the commit
//   - walkErrors: per-blob-read failures the rule should surface as
//     history-walk-error findings without aborting the walk
//   - fatalErr: walker-level failure (BulkRevwalk subprocess crash,
//     context cancelled before any commit was processed). The
//     observations and walkErrors collected before the fatal are
//     still returned — partial results survive.
//
// Returns (nil, nil, nil) for nil tree, empty root, or a repo with
// no commits — the same "nothing to walk" semantic the M-0130
// per-entity walker used.
//
// Rename-chain tracking: BulkRevwalk emits commits newest-first by
// default. The walker maintains a pathToEntity map seeded from the
// tree's CURRENT paths; when a rename touch (Status="R") is
// processed, the SrcPath is added to the map (the entity used to
// live there). Older commits referencing the entity at its
// pre-rename path then resolve correctly. Same imperfection as the
// M-0130 walker: a commit that both renames AND changes status is
// unobserved — the parent has the file at SrcPath, the rule reads
// parent:t.Path (the new name) which doesn't exist, the pair is
// skipped. Pure renames don't change status, so no observation is
// lost on the typical path.
func batchedWalkStatusChanges(ctx context.Context, root string, t *tree.Tree, br blobReader) ([]statusChange, []walkError, error) {
	if t == nil || root == "" {
		return nil, nil, nil
	}
	if !hasGitCommits(ctx, root) {
		return nil, nil, nil
	}

	pathToEntity := make(map[string]*entity.Entity, len(t.Entities))
	for _, e := range t.Entities {
		if e == nil || e.Path == "" {
			continue
		}
		pathToEntity[e.Path] = e
	}
	if len(pathToEntity) == 0 {
		return nil, nil, nil
	}

	var (
		observations []statusChange
		walkErrors   []walkError
		// Dedup by (commit, parent, path). BulkRevwalk emits one
		// CommitRecord per parent-diff under -m, so a merge commit
		// whose touched paths differ from BOTH parents appears in
		// multiple records. Each record's rec.Parents lists ALL
		// parents — so iterating per-parent inside the callback would
		// re-emit each (commit, parent, path) tuple once per record.
		// Dedup at this layer collapses to one observation per real
		// (commit, parent, path) triple.
		seen = make(map[string]struct{})
		// Dedup walk-errors similarly: parent-side read failures for
		// the same (commit, path) shouldn't be double-counted across
		// per-parent CommitRecord emissions.
		seenErr = make(map[string]struct{})
	)

	walkErr := gitops.BulkRevwalk(ctx, root, func(rec gitops.CommitRecord) error {
		// Single-pass per commit-record: for each path touched,
		// attribute it to an entity (if known), then read commit-side
		// + per-parent statuses and emit observations when they
		// differ.
		isMerge := len(rec.Parents) > 1
		for _, touch := range rec.Paths {
			e, ok := pathToEntity[touch.Path]
			if !ok {
				// Path not associated with any known entity (yet).
				// Skip; if a later (older) rename brings it back into
				// scope via SrcPath, future iterations of older
				// commits will see it.
				continue
			}

			commitStatus, readErr := readStatusAt(rec.Commit, touch.Path, br)
			if readErr != nil {
				key := rec.Commit + "\x00" + touch.Path + "\x00commit"
				if _, dup := seenErr[key]; !dup {
					seenErr[key] = struct{}{}
					walkErrors = append(walkErrors, walkError{
						EntityID: e.ID,
						Commit:   rec.Commit,
						Path:     touch.Path,
						Side:     "commit",
						Err:      readErr,
					})
				}
				// Can't compare without commit-side status; skip per-
				// parent reads but DON'T abort the walk.
				continue
			}
			if commitStatus == "" {
				// File deleted at this commit, or no frontmatter
				// status — nothing to compare against.
				continue
			}

			if len(rec.Parents) == 0 {
				// Root commit: no parent to compare against.
				continue
			}

			for _, parent := range rec.Parents {
				priorStatus, readErr := readStatusAt(parent, touch.Path, br)
				if readErr != nil {
					key := rec.Commit + "\x00" + parent + "\x00" + touch.Path + "\x00parent"
					if _, dup := seenErr[key]; !dup {
						seenErr[key] = struct{}{}
						walkErrors = append(walkErrors, walkError{
							EntityID: e.ID,
							Commit:   rec.Commit,
							Parent:   parent,
							Path:     touch.Path,
							Side:     "parent",
							Err:      readErr,
						})
					}
					continue
				}
				if priorStatus == "" || priorStatus == commitStatus {
					continue
				}
				obsKey := rec.Commit + "\x00" + parent + "\x00" + touch.Path
				if _, dup := seen[obsKey]; dup {
					continue
				}
				seen[obsKey] = struct{}{}
				observations = append(observations, statusChange{
					EntityID:      e.ID,
					EntityKind:    e.Kind,
					Commit:        rec.Commit,
					Parent:        parent,
					Path:          touch.Path,
					Prior:         priorStatus,
					Next:          commitStatus,
					Trailers:      rec.Trailers,
					IsMergeCommit: isMerge,
				})
			}

			// Rename: the entity lived at SrcPath before this commit.
			// Add to the map so older commits' touches at SrcPath
			// resolve to this entity.
			if touch.Status == "R" && touch.SrcPath != "" {
				pathToEntity[touch.SrcPath] = e
			}
		}
		return nil
	})

	return observations, walkErrors, walkErr
}

// readStatusAt reads the entity file's frontmatter status field at
// (commit, path) via the blobReader dep. Returns:
//
//   - ("", nil) when the path doesn't exist at the commit (the
//     blobReader returns ErrBlobMissing) or when the frontmatter has
//     no status field — the "skip this pair" signal that
//     statusAtCommitPath returned via empty string in M-0130
//   - ("", err) for real failure modes the walker should surface
//     (subprocess crash, protocol violation, injected test failure)
//   - (status, nil) on success
func readStatusAt(commit, path string, br blobReader) (string, error) {
	content, err := br.Read(commit, path)
	if errors.Is(err, gitops.ErrBlobMissing) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return parseStatusFromFrontmatter(content), nil
}

// historyWalkErrorFindings turns the walker's per-blob-read errors
// into fsm-history-consistent/history-walk-error findings (severity
// error). One finding per walkError so the operator sees which
// (entity, commit) read failed — and partial findings for healthy
// entities still emerge alongside.
//
// Dedupes per (EntityID, Commit, Side) so a multi-parent merge with
// the same parent-side read failing N times doesn't inflate the
// finding count.
func historyWalkErrorFindings(walkErrors []walkError) []Finding {
	if len(walkErrors) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(walkErrors))
	out := make([]Finding, 0, len(walkErrors))
	for _, we := range walkErrors {
		key := we.EntityID + "\x00" + we.Commit + "\x00" + we.Side
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		var detail strings.Builder
		detail.WriteString("entity ")
		detail.WriteString(we.EntityID)
		detail.WriteString(": walker failed reading ")
		detail.WriteString(we.Side)
		detail.WriteString(" status at ")
		detail.WriteString(shortHash(we.Commit))
		detail.WriteString(":")
		detail.WriteString(we.Path)
		detail.WriteString(": ")
		detail.WriteString(we.Err.Error())
		out = append(out, Finding{
			Code:     CodeFSMHistoryConsistent,
			Subcode:  "history-walk-error",
			Severity: SeverityError,
			Message:  detail.String(),
			Path:     we.Path,
			EntityID: we.EntityID,
			Field:    "status",
		})
	}
	return out
}
