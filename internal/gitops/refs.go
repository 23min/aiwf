package gitops

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrRefNotFound reports that the requested ref does not resolve in
// workdir's git repository. Wrapped by HasRef and LsTreePaths so
// callers can distinguish "ref absent" (potentially a sandbox repo)
// from "git failed for some other reason."
var ErrRefNotFound = errors.New("ref not found")

// HasRemotes reports whether workdir has any configured git remote.
// A repo with no remotes has no possible cross-branch coordination
// surface, so the trunk-aware allocator skips its check there.
func HasRemotes(ctx context.Context, workdir string) (bool, error) {
	out, err := output(ctx, workdir, "remote")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// HasAnyRemoteTrackingRefs reports whether workdir has any
// refs/remotes/* ref recorded locally. Used by the trunk-awareness
// policy to distinguish "remote configured but never populated"
// (e.g., a clone of an empty bare repo, before the first push) from
// "remote configured and the trunk ref just doesn't match what's
// fetched" (a real misconfiguration).
//
// Returns (false, nil) when no tracking refs exist; (true, nil) when
// at least one does. Other git failures propagate as wrapped errors.
func HasAnyRemoteTrackingRefs(ctx context.Context, workdir string) (bool, error) {
	out, err := output(ctx, workdir, "for-each-ref", "--count=1", "--format=%(refname)", "refs/remotes/")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// AddCommitSHA returns the SHA of the commit that introduced
// relPath into the repo. Returns ("", nil) when the file has no add
// commit visible from HEAD (newly staged but never committed).
// Wraps git failures.
//
// `git log --diff-filter=A --pretty=%H -- <path>` is git's "when
// did this exact path first appear" query. We deliberately do NOT
// pass `--follow`: it traces *content* across renames as a
// heuristic, which produces wrong answers in the duplicate-id case
// the reallocate tiebreaker cares about — two entity files of the
// same kind have nearly-identical frontmatter/body shapes, and
// `--follow` will frequently mis-attribute one's add commit to the
// other's. The exact-path query is what we actually want: the
// commit that first put bytes at this exact path.
func AddCommitSHA(ctx context.Context, workdir, relPath string) (string, error) {
	out, err := output(ctx, workdir, "log", "--diff-filter=A", "--pretty=%H", "--", relPath)
	if err != nil {
		return "", fmt.Errorf("finding add commit for %s: %w", relPath, err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	// `git log` lists newest first; with --diff-filter=A the *last*
	// line is the original add. That's what callers want when
	// ranking two entities by birth order.
	for i := len(lines) - 1; i >= 0; i-- {
		s := strings.TrimSpace(lines[i])
		if s != "" {
			return s, nil
		}
	}
	return "", nil
}

// IsAncestor reports whether commit is an ancestor of ref (i.e.
// `git merge-base --is-ancestor <commit> <ref>` succeeds). Returns
// (false, nil) when commit is not an ancestor; (true, nil) when it
// is; an error only on real git failures (bad refs, missing repo).
//
// The reallocate tiebreaker uses this to ask "which side already
// exists on trunk?" — the side that does keeps the id; the side
// that doesn't gets renumbered.
func IsAncestor(ctx context.Context, workdir, commit, ref string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "merge-base", "--is-ancestor", commit, ref)
	cmd.Dir = workdir
	cmd.Env = gitEnv()
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Exit 1 = not an ancestor. Exit 128 = bad ref / repo issue.
			if exitErr.ExitCode() == 1 {
				return false, nil
			}
			return false, fmt.Errorf("git merge-base --is-ancestor %s %s: %w", commit, ref, err)
		}
		return false, fmt.Errorf("git merge-base --is-ancestor %s %s: %w", commit, ref, err)
	}
	return true, nil
}

// ShortSHA returns the first n hex characters of the commit SHA that
// ref resolves to, via `git rev-parse --short=n ref`. Returns ""
// (and a wrapped error) when the ref does not resolve or git fails.
// Used by the doctor binary-staleness check (G-0176) to compare a
// pseudo-version's 12-char SHA prefix against the trunk-ref HEAD.
func ShortSHA(ctx context.Context, workdir, ref string, n int) (string, error) {
	out, err := output(ctx, workdir, "rev-parse", fmt.Sprintf("--short=%d", n), ref)
	if err != nil {
		return "", fmt.Errorf("git rev-parse --short=%d %s: %w", n, ref, err)
	}
	return strings.TrimSpace(out), nil
}

// HasRef reports whether ref resolves to an object in workdir's repo.
// Returns (false, nil) when the ref is absent — distinguishing it
// from any other git failure, which propagates as a wrapped error.
func HasRef(ctx context.Context, workdir, ref string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	cmd.Dir = workdir
	cmd.Env = gitEnv()
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("git rev-parse --verify %s: %w", ref, err)
	}
	return true, nil
}

// RenamesFromRef returns the set of file renames committed on HEAD
// since it diverged from ref — i.e., renames in commits reachable from
// HEAD but not from ref. Keys are pre-rename paths, values are post-
// rename paths (both repo-relative, slash-separated).
//
// Used by `aiwf check`'s ids-unique trunk-collision rule (G-0109) so a
// feature-branch slug rename of an existing entity is recognized as
// the same entity moved, not a duplicate id allocation. Without this,
// any rename-heavy cleanup on a feature branch produces a finding per
// renamed entity and blocks `git push` via the pre-push hook — the
// catch-22 the gap documents.
//
// The scope is deliberately **merge-base(HEAD, ref)..HEAD**, not
// `ref..HEAD` or `ref` vs the working tree. The merge-base scoping
// matters for the G37 case the trunk-collision rule was originally
// designed to catch: two parallel clones each independently allocate
// the same id at different slug-derived paths. Comparing ref's tree
// to HEAD's tree (or to the working tree) sees both sides' add+delete
// pair and git's similarity heuristic matches them as a rename, even
// though no rename ever happened. Scoping to merge-base..HEAD only
// surfaces the renames *this branch* committed; the other clone's
// add isn't in this branch's history at all and can't be misread as
// a rename.
//
// Returns an empty map (not nil) when no renames are detected. Returns
// (nil, nil) when ref does not resolve, when HEAD has no commits, or
// when ref and HEAD share no common ancestor — in each case the
// trunk-collision rule already degrades to "no cross-tree view" so
// the empty answer is the correct one.
//
// `-z` is required for safe parsing: file paths can legally contain
// any byte except NUL, and the default newline-separated output
// breaks on paths with embedded tabs or newlines.
func RenamesFromRef(ctx context.Context, workdir, ref string) (map[string]string, error) {
	exists, err := HasRef(ctx, workdir, ref)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	headExists, err := HasRef(ctx, workdir, "HEAD")
	if err != nil {
		return nil, err
	}
	if !headExists {
		return nil, nil
	}
	mbOut, err := output(ctx, workdir, "merge-base", "HEAD", ref)
	if err != nil {
		// No common ancestor (unrelated histories) is a legitimate
		// "no cross-tree view" — return an empty map rather than
		// erroring. Other failures (bad workdir, etc.) propagate.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("finding merge-base of HEAD and %s: %w", ref, err)
	}
	mergeBase := strings.TrimSpace(mbOut)
	if mergeBase == "" {
		return map[string]string{}, nil
	}
	renames := make(map[string]string)

	// Pass 1: trailer-driven rename detection (G-0167).
	//
	// Walk commits on merge-base..HEAD with rename-shaped aiwf-verb
	// trailers (retitle, rename, reallocate, archive, move) and
	// record their file moves. The kernel's verbs stamp explicit
	// trailers naming the operator's intent; that intent is ground
	// truth for "this was a rename" — strictly better than git's
	// similarity heuristic, which can miss real renames when body
	// changes pile up (e.g., retitle + 3× body enrichment in
	// separate commits drops cumulative similarity below the -M50
	// threshold).
	//
	// Chains forward: if commit X renames A→B and a later commit Y
	// renames B→C, the cumulative map records A→C.
	trailerRenames, err := renamesFromAiwfVerbTrailers(ctx, workdir, mergeBase)
	if err != nil {
		return nil, fmt.Errorf("detecting trailer-driven renames since merge-base with %s: %w", ref, err)
	}
	for k, v := range trailerRenames {
		renames[k] = v
	}

	// Pass 2: cumulative -M default similarity detection.
	//
	// Catches non-trailered renames (legacy git-mv operations,
	// externally committed file moves, third-party-tool moves). The
	// default 50% threshold is preserved here to keep the G-0109
	// parallel-collision behavior intact — two unrelated entities
	// that share frontmatter shape but distinctive body content
	// stay below 50% similarity and are correctly NOT paired as a
	// rename.
	out, err := output(ctx, workdir, "diff", "-M", "--diff-filter=R", "--name-status", "-z", mergeBase, "HEAD")
	if err != nil {
		return nil, fmt.Errorf("detecting renames since merge-base with %s: %w", ref, err)
	}
	if out == "" {
		return renames, nil
	}
	// With -z, each rename entry serializes as three NUL-separated
	// fields: "R<score>", oldPath, newPath. A trailing NUL after the
	// last newPath is typical but not guaranteed; TrimRight handles
	// either form.
	fields := strings.Split(strings.TrimRight(out, "\x00"), "\x00")
	for i := 0; i+2 < len(fields); i += 3 {
		status := fields[i]
		if status == "" || status[0] != 'R' {
			continue
		}
		oldPath := fields[i+1]
		newPath := fields[i+2]
		if oldPath == "" || newPath == "" {
			continue
		}
		// Trailer-driven entries take precedence — they reflect the
		// operator's stated intent. The cumulative diff fills in
		// non-trailered renames only.
		if _, alreadyTrailered := renames[oldPath]; alreadyTrailered {
			continue
		}
		renames[oldPath] = newPath
	}
	return renames, nil
}

// renameVerbs is the closed set of aiwf-verb trailer values whose
// commits move entity files. Used by renamesFromAiwfVerbTrailers to
// scope the trailer walk. The values are checked against the literal
// strings emitted by the corresponding verbs in internal/verb/.
//
// Closed set intentionally — adding a new rename-shaped verb to the
// kernel without adding it here would silently regress G-0167's
// trailer-driven detection for that verb's renames.
var renameVerbs = map[string]bool{
	"retitle":    true, // changes slug + frontmatter title
	"rename":     true, // changes slug only
	"reallocate": true, // changes id + slug (entity renumber)
	"archive":    true, // sweeps to per-kind archive/ subdir
	"move":       true, // milestone changes parent epic
}

// renamesFromAiwfVerbTrailers walks commits on mergeBase..HEAD and
// returns the cumulative file-rename map produced by aiwf-verb
// commits whose verb value is in renameVerbs. Chains forward: an
// A→B rename in commit X followed by a B→C rename in commit Y
// collapses to A→C in the returned map.
//
// Returns an empty map when no rename-shaped trailers exist in the
// range.
func renamesFromAiwfVerbTrailers(ctx context.Context, workdir, mergeBase string) (map[string]string, error) {
	// Walk commits oldest-first so per-commit renames apply in
	// chronological order; chain-forward updates downstream entries
	// when a later commit re-renames an earlier rename's destination.
	//
	// Format: each commit produces "COMMIT <sha>" followed by trailer
	// lines (one per line, "Key: value"), terminated by a literal
	// "END_COMMIT" marker. The marker pattern is the simplest way to
	// frame multi-line trailer blocks without ambiguity around blank
	// lines inside trailers (which `unfold=true` already collapses).
	const recordSeparator = "END_COMMIT"
	out, err := output(ctx, workdir, "log", "--reverse",
		"--format=COMMIT %H%n%(trailers:only=true,unfold=true)"+recordSeparator,
		mergeBase+"..HEAD")
	if err != nil {
		return nil, fmt.Errorf("walking aiwf-verb trailers: %w", err)
	}
	renames := map[string]string{}
	for _, record := range strings.Split(out, recordSeparator) {
		record = strings.TrimSpace(record)
		if record == "" {
			continue
		}
		lines := strings.Split(record, "\n")
		if len(lines) == 0 || !strings.HasPrefix(lines[0], "COMMIT ") {
			continue
		}
		sha := strings.TrimSpace(strings.TrimPrefix(lines[0], "COMMIT "))
		if sha == "" {
			continue
		}
		trailers := ParseTrailers(strings.Join(lines[1:], "\n"))
		verb := ""
		for _, tr := range trailers {
			if tr.Key == TrailerVerb {
				verb = tr.Value
				break
			}
		}
		if !renameVerbs[verb] {
			continue
		}
		perCommitRenames, err := renamesInCommit(ctx, workdir, sha)
		if err != nil {
			return nil, fmt.Errorf("reading per-commit renames for %s: %w", sha, err)
		}
		for src, dst := range perCommitRenames {
			// Chain forward: if any existing entry's value equals
			// the new src, that entry now points to dst instead.
			// Without chaining, a multi-step rename (A→B in commit
			// X, B→C in commit Y) would be recorded as {A:B, B:C}
			// instead of {A:C}, and the consumer (trunk-collision
			// rule) would look up A and find B (which doesn't
			// exist on the branch).
			chained := false
			for oldKey, oldDst := range renames {
				if oldDst == src {
					renames[oldKey] = dst
					chained = true
					// Don't break — a later rename could be the
					// destination of multiple earlier renames if
					// two entities were merged into one, though
					// the kernel verbs don't do that today. Safe
					// to continue scanning.
				}
			}
			if !chained {
				renames[src] = dst
			}
		}
	}
	return renames, nil
}

// renamesInCommit returns the file renames recorded by a single
// commit. Uses git's per-commit `-M` similarity at the default
// threshold; per-commit diffs are typically very high-similarity
// (the kernel verbs that move files don't simultaneously rewrite
// the body), so the default threshold reliably catches them.
func renamesInCommit(ctx context.Context, workdir, sha string) (map[string]string, error) {
	out, err := output(ctx, workdir, "show", "-M", "--diff-filter=R", "--name-status", "-z",
		"--format=", // suppress commit header — we only want the diff
		sha)
	if err != nil {
		return nil, err
	}
	renames := map[string]string{}
	if out == "" {
		return renames, nil
	}
	fields := strings.Split(strings.TrimRight(out, "\x00"), "\x00")
	for i := 0; i+2 < len(fields); i += 3 {
		status := fields[i]
		if status == "" || status[0] != 'R' {
			continue
		}
		oldPath := fields[i+1]
		newPath := fields[i+2]
		if oldPath == "" || newPath == "" {
			continue
		}
		renames[oldPath] = newPath
	}
	return renames, nil
}

// LsTreePaths returns the file paths under ref's tree, optionally
// filtered to those whose slash-normalized path begins with any of the
// supplied prefixes. Pass no prefixes to list every path. Paths are
// repo-relative and slash-separated; ordering is git's (sorted).
//
// Returns ErrRefNotFound (wrapped) when ref does not resolve. Other
// git failures propagate as wrapped errors. An existing but empty
// ref tree returns ([]string{}, nil).
func LsTreePaths(ctx context.Context, workdir, ref string, prefixes ...string) ([]string, error) {
	exists, err := HasRef(ctx, workdir, ref)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrRefNotFound, ref)
	}
	out, err := output(ctx, workdir, "ls-tree", "--full-tree", "-r", "--name-only", "-z", ref)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return []string{}, nil
	}
	parts := strings.Split(strings.TrimRight(out, "\x00"), "\x00")
	paths := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		if len(prefixes) == 0 {
			paths = append(paths, p)
			continue
		}
		for _, prefix := range prefixes {
			if strings.HasPrefix(p, prefix) {
				paths = append(paths, p)
				break
			}
		}
	}
	return paths, nil
}
