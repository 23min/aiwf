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

// blobFrontmatter is what the walk reads out of one entity blob: the
// status recorded there, and the id saying whose status it is.
type blobFrontmatter struct {
	ID     string
	Status string
}

// names reports whether this blob's frontmatter id names e.
//
// A blob reached by object id has no path to identify it by, so the id
// it carries is the whole of what ties it to an entity. Widths are
// canonicalized because a narrower legacy id names the same entity
// (ADR-0008), and an `aiwf reallocate` is consulted through prior_ids,
// since it renames the file and rewrites the id in one commit — making
// the id the pre-image carries one e has since left behind.
//
// The prior_ids arm is as precise as prior_ids itself: an id a
// reallocation freed can be handed to a later entity, and both that
// entity and the one that vacated it then answer to it. The arm is
// reached only for a blob a rename already paired with e, and e's own
// id is consulted first, so the ambiguity costs at most a skipped
// observation on the entity that moved away.
//
// prior_ids is also what a renumbering has to leave behind to stay
// observable. A file renumbered by hand rather than by the verb
// records no lineage, so its history before the renumber is no longer
// attributable to it — which is what the tree already says about it.
func (fm blobFrontmatter) names(e *entity.Entity) bool {
	if fm.ID == "" {
		// Frontmatter without an id, or none at all — a file that the
		// renaming commit is what turns into an entity. Nothing ties it
		// to e, so it carries no prior status for e.
		return false
	}
	id := entity.Canonicalize(fm.ID)
	if id == entity.Canonicalize(e.ID) {
		return true
	}
	for _, prior := range e.PriorIDs {
		if id == entity.Canonicalize(prior) {
			return true
		}
	}
	return false
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
// pre-rename path then resolve correctly. The renaming commit itself
// is observed on the same footing: its parent-side status comes from
// the diff record's pre-image blob, which is the blob at SrcPath, so
// a commit that renames the file and rewrites its status in one step
// still reaches the FSM's legality verdict.
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
		// Dedup by (commit, parent, path). Historically (pre G-0372 Fix
		// 1) BulkRevwalk requested `-m`, emitting one CommitRecord per
		// parent-diff for a merge commit whose touched paths differed
		// from BOTH parents — dedup here collapsed those duplicate
		// (commit, parent, path) emissions to one observation. Without
		// -m, merge commits carry no Paths at all (see
		// gitops.CommitRecord's doc), so this dedup is now dormant for
		// merges specifically; kept as-is since it's still correct and
		// harmless for any future multi-record shape.
		seen = make(map[string]struct{})
		// Dedup walk-errors similarly: parent-side read failures for
		// the same (commit, path) shouldn't be double-counted across
		// multiple CommitRecord emissions for one commit.
		seenErr = make(map[string]struct{})
	)

	// M-0216 AC-2: read status by blob object id (the pre/post id
	// columns `git log --raw` puts on each PathTouch) instead of
	// resolving `<commit>:<path>` per read, which forces git to walk the
	// tree from the commit root to the blob on every call (~3× slower on
	// the kernel tree). Object ids dedupe across the walk — a commit's
	// PostSHA equals its child's PreSHA at the same path — so shaCache
	// reads each unique blob once. An all-zero id is the absent side of
	// an add or a delete and yields ("", nil) without a read; every id
	// that survives that check names a blob the walk needs, so failing
	// to read one is an error rather than a skip. gitops.ErrBlobMissing
	// arrives here for a real blob the local object store cannot produce
	// and git will not go and fetch — a damaged store, or a partial
	// clone whose lazy fetch is off or whose promisor remote is no
	// longer configured. A partial clone that can still fetch never gets
	// here, since git backfills the blob to answer the read; one whose
	// promisor is configured but unreachable does not either, because
	// the failed fetch kills the cat-file subprocess and the walk
	// reports that instead. Either way this is history the walk cannot
	// see, not history that says nothing.
	//
	// The id rides along with the status because a blob addressed by
	// object id carries no path, so its frontmatter id is the only
	// thing that says which entity the status belongs to.
	shaCache := make(map[string]blobFrontmatter)
	frontmatterBySHA := func(sha string) (blobFrontmatter, error) {
		if gitops.BlobAllZero(sha) {
			return blobFrontmatter{}, nil
		}
		if fm, ok := shaCache[sha]; ok {
			return fm, nil
		}
		content, err := br.ReadObject(sha)
		if err != nil {
			// Surface to the caller as a walk error; don't cache a
			// transient as authoritative.
			return blobFrontmatter{}, err
		}
		id, status := parseIDAndStatusFromFrontmatter(content)
		fm := blobFrontmatter{ID: id, Status: status}
		shaCache[sha] = fm
		return fm, nil
	}
	statusBySHA := func(sha string) (string, error) {
		fm, err := frontmatterBySHA(sha)
		return fm.Status, err
	}

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

			// Commit-side: PostSHA is by definition the blob at
			// touch.Path at this commit, for every status — a delete's
			// all-zero PostSHA reads as "" via statusBySHA, the same skip
			// the deleted-file branch below took. BulkRevwalk always emits
			// --raw, so PostSHA is populated.
			commitStatus, readErr := statusBySHA(touch.PostSHA)
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
				// Parent-side status at touch.Path. PreSHA is the blob at
				// the parent THIS diff record is against — but only when
				// the diff kept the same path is it the blob at
				// touch.Path, so the fast path is restricted to that case:
				//
				//   - merge: unreachable in practice since G-0372 Fix 1 —
				//     BulkRevwalk no longer requests -m, so a merge
				//     commit's rec.Paths is always empty and this loop
				//     body never runs for one. Kept as the documented
				//     fallback in case that ever changes: PreSHA can't be
				//     matched to a specific `parent` from a fan-out record
				//     that lists all parents, so the path-resolving read
				//     is the only correct option here.
				//   - rename ("R"): PreSHA is the blob at SrcPath, where
				//     the parent holds the file this touch renames.
				//     Resolving `parent:touch.Path` instead finds nothing,
				//     since this commit is what puts the file there, and
				//     the dropped pair would let a commit that renames the
				//     file and rewrites its status in one step escape the
				//     FSM's legality verdict. Read by object id — but only
				//     after checking whose file it is, because -M pairs a
				//     delete with an add by content *similarity*, not by
				//     identity: retiring one template-shaped entity and
				//     opening another in a single commit pairs exactly the
				//     same way, and taking that pre-image would report the
				//     retired entity's status as the opened one's prior
				//     state. On a mismatch the pair is skipped, which is
				//     what the destination's absence at the parent means.
				//   - copy ("C"): PreSHA is the source file's blob, but
				//     the source keeps its own life and the entity at
				//     touch.Path is new here — so its prior status is
				//     "absent", which only the path-resolving read
				//     reports. The explicit -M BulkRevwalk passes
				//     overrides any `diff.renames=copies` the consumer
				//     configured, so git emits no copy record; this is the
				//     correct handling if that flag list ever changes.
				//   - otherwise ("M"/"A"/"T"): touch.Path is unchanged, so
				//     PreSHA is exactly the parent's blob at touch.Path
				//     (an add's all-zero PreSHA reads as "", matching the
				//     parent-has-no-file case). Read by object id.
				//
				// All three fallback conditions are unreachable under the
				// flags BulkRevwalk passes today, so no test drives the
				// path-resolving arm: a merge carries no Paths without -m,
				// -M overrides any copy-detection config, and --raw always
				// fills PreSHA. The arm is what keeps each of them correct
				// if a flag changes.
				var priorStatus string
				var readErr error
				switch {
				case touch.PreSHA == "" || isMerge || touch.Status == "C":
					priorStatus, readErr = readStatusAt(parent, touch.Path, br) //coverage:ignore see the enumeration above — no flag BulkRevwalk passes can produce a touch that reaches this arm
				case touch.Status == "R":
					var fm blobFrontmatter
					if fm, readErr = frontmatterBySHA(touch.PreSHA); readErr == nil && fm.names(e) {
						priorStatus = fm.Status
					}
				default:
					priorStatus, readErr = statusBySHA(touch.PreSHA)
				}
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
					Prior:         entity.Status(priorStatus),
					Next:          entity.Status(commitStatus),
					Trailers:      rec.Trailers,
					IsMergeCommit: isMerge,
				})
			}

			// Rename: the entity lived at SrcPath before this commit, so
			// older commits' touches there resolve to it — but only once
			// the pre-image is confirmed to be this entity's own file.
			// A similarity-paired delete and add names a path that
			// belonged to a different entity, and seeding it would
			// attribute that entity's entire earlier history to this
			// one. The read is the same one the parent-side comparison
			// above made, served from shaCache.
			if touch.Status == "R" && touch.SrcPath != "" {
				if fm, err := frontmatterBySHA(touch.PreSHA); err == nil && fm.names(e) {
					pathToEntity[touch.SrcPath] = e
				}
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
//
// The ErrBlobMissing skip here is not the one statusBySHA declines to
// take, and the difference is what the reader can tell apart.
// `git cat-file --batch` answers a `<commit>:<path>` request that names
// no file and one whose blob the object store lacks with the same
// `missing`, so this route cannot separate "the parent had no such
// file" — the ordinary reading of an add — from an unreadable store,
// and reporting the pair would make an error of every add. A request
// by object id carries no path and so has no absent-file reading at
// all: an id `git log --raw` printed names a blob that must exist, and
// only a store that cannot produce it answers missing.
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
