// Package areamatch answers whether a repo-relative path matches an area's
// path glob. It is the single source of truth for glob semantics across the
// area-path checks (dead-glob and overlap; M-0180) and the Tier-2 consumers
// that reuse it (mistag M-0181, auto-derive M-0182).
package areamatch

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/bmatcuk/doublestar/v4"
)

// errStopWalk is the sentinel MatchesAny returns from its GlobWalk callback to
// halt on the first match. doublestar.GlobWalk stops and returns it verbatim,
// so MatchesAny recognizes it via errors.Is to mean "matched", distinct from a
// real walk error.
var errStopWalk = errors.New("areamatch: match found")

// Match reports whether the repo-relative path matches the area path glob.
// Both arguments are '/'-separated and repo-relative (no leading or trailing
// slash); the glob may use doublestar ('**') semantics that the standard
// library's filepath.Match cannot evaluate. A malformed glob returns an error
// wrapping doublestar.ErrBadPattern.
func Match(glob, path string) (bool, error) {
	return doublestar.Match(glob, path)
}

// MatchFS returns the repo-relative paths under fsys that the glob matches —
// files and directories alike. An empty result means the glob locates
// nothing (a dead glob). The glob uses the same doublestar ('**') semantics
// as Match; a malformed glob returns an error wrapping
// doublestar.ErrBadPattern, and a filesystem walk error is returned as-is.
// Callers that must never fail on IO (the check rules) treat any error as
// "indeterminate" and skip.
func MatchFS(fsys fs.FS, glob string) ([]string, error) {
	return doublestar.Glob(fsys, glob)
}

// MatchesAny reports whether the glob matches at least one real path under
// fsys, short-circuiting on the first match. It is the boolean-any primitive
// the dead-glob check uses (and the M-0185 scoped-coverage check will reuse):
// unlike MatchFS it does not enumerate the entire subtree of a '**' glob, so
// it stays cheap on the large monorepo trees the areas feature targets. Same
// '**' semantics as Match. A malformed glob or a filesystem walk error is
// returned; callers that must never fail on IO (the check rules) treat any
// error as "indeterminate" and skip.
func MatchesAny(fsys fs.FS, glob string) (bool, error) {
	err := doublestar.GlobWalk(fsys, glob, func(string, fs.DirEntry) error {
		return errStopWalk
	})
	switch {
	case err == nil:
		return false, nil
	case errors.Is(err, errStopWalk):
		return true, nil
	default:
		return false, err
	}
}

// Validate reports whether the glob is syntactically well-formed. It is the
// Tier-1 (config-load) gate: config.Areas.validate calls it so a malformed
// area path glob is a hard error at load — naming the bad glob — rather than
// being silently skipped by the path-axis checks at runtime. Routing the
// syntax check through the SSOT keeps config-load from importing doublestar
// directly. Returns an error wrapping doublestar.ErrBadPattern for a malformed
// glob, nil otherwise.
func Validate(glob string) error {
	if !doublestar.ValidatePattern(glob) {
		return fmt.Errorf("%w: %q", doublestar.ErrBadPattern, glob)
	}
	return nil
}
