package gitops

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// DivergenceKind names the way a path's working copy disagrees with the
// record at HEAD.
type DivergenceKind int

const (
	// DivergenceModified names a path present at HEAD and on disk,
	// holding different bytes.
	DivergenceModified DivergenceKind = iota
	// DivergenceAbsentFromHEAD names a path on disk with no version at
	// HEAD, so there is no record for it to contradict.
	DivergenceAbsentFromHEAD
	// DivergenceAbsentFromDisk names a path recorded at HEAD and missing
	// from the working tree.
	DivergenceAbsentFromDisk
)

// Divergence reports one path whose working copy differs from HEAD.
type Divergence struct {
	Path string
	Kind DivergenceKind
}

// DivergentPaths reports, for each requested repo-relative path, whether
// the working copy still equals the record at HEAD — and when it does
// not, in which of the three ways. Paths are '/'-separated and
// repo-relative; each names a file. The result is sorted by path, and a
// path equal to its HEAD version is absent from it.
//
// It answers "what does this commit carry that the record does not",
// which is a different question from "what has the operator changed".
// The dirty-set queries answer the second: they read git's own report of
// the working tree, so a path git has been told to stop reporting — an
// ignored file, one carrying `assume-unchanged` or `skip-worktree` —
// reads as clean there while differing on disk. This comparison reads
// HEAD's blob and the file, and neither side consults the index or
// `.gitignore`, so no such mechanism can hide a path from it (G-0492).
//
// The comparison is content against content. In a repo configured with
// content filters (`core.autocrlf`, a clean/smudge driver) the stored
// blob and the checked-out file differ by design for an untouched path,
// and it reports that path as modified — the same assumption the
// adoption check in internal/verb already rests on.
//
// Callers consult HasHEAD first: with an unborn HEAD there is no record
// to compare against, and every path on disk would read as absent from
// it.
func DivergentPaths(ctx context.Context, workdir string, paths []string) ([]Divergence, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	var out []Divergence
	for _, p := range paths {
		head, headErr := ReadFromHEAD(ctx, workdir, p)
		if headErr != nil {
			return nil, fmt.Errorf("reading %s at HEAD: %w", p, headErr)
		}
		disk, diskErr := os.ReadFile(filepath.Join(workdir, filepath.FromSlash(p))) //nolint:gosec // repo-relative path supplied by the caller's own plan or loaded tree
		switch {
		case diskErr != nil && !errors.Is(diskErr, fs.ErrNotExist):
			return nil, fmt.Errorf("reading %s: %w", p, diskErr)
		case diskErr != nil:
			// Absent from disk. Divergent only if HEAD recorded it —
			// a path in neither place is not a disagreement, it is a
			// caller asking about something that does not exist.
			if head != nil {
				out = append(out, Divergence{Path: p, Kind: DivergenceAbsentFromDisk})
			}
		case head == nil:
			out = append(out, Divergence{Path: p, Kind: DivergenceAbsentFromHEAD})
		case !bytes.Equal(head, disk):
			out = append(out, Divergence{Path: p, Kind: DivergenceModified})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}
