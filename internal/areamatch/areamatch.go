// Package areamatch answers whether a repo-relative path matches an area's
// path glob. It is the single source of truth for glob semantics across the
// area-path checks (dead-glob and overlap; M-0180) and the Tier-2 consumers
// that reuse it (mistag M-0181, auto-derive M-0182).
package areamatch

import "github.com/bmatcuk/doublestar/v4"

// Match reports whether the repo-relative path matches the area path glob.
// Both arguments are '/'-separated and repo-relative (no leading or trailing
// slash); the glob may use doublestar ('**') semantics that the standard
// library's filepath.Match cannot evaluate. A malformed glob returns an error
// wrapping doublestar.ErrBadPattern.
func Match(glob, path string) (bool, error) {
	return doublestar.Match(glob, path)
}
