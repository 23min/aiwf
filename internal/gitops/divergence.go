package gitops

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

// DivergenceKind names the way a path's working copy disagrees with the
// record at HEAD.
type DivergenceKind int

const (
	// DivergenceModified names a path recorded at HEAD and present on
	// disk whose working copy would not store as what the record holds —
	// different content, or a different kind of entry.
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

// headEntry is one path's record at HEAD: the object id git stores for
// it, and whether that entry is a symbolic link.
type headEntry struct {
	OID       string
	IsSymlink bool
}

// argvChunk bounds how many paths ride on one git command line, well
// under any platform's argument limit while keeping the subprocess count
// proportional to the set rather than to each path.
const argvChunk = 500

// DivergentPaths reports, for each requested repo-relative path, whether
// the working copy still equals the record at HEAD — and when it does
// not, in which of the three ways. Paths are '/'-separated and
// repo-relative; each names a file or a symbolic link. The result is
// sorted by path, and a path equal to its HEAD version is absent from it.
//
// It answers "what would this commit carry that the record does not",
// which is a different question from "what has the operator changed".
// The dirty-set queries answer the second: they read git's own report of
// the working tree, so a path git has been told to stop reporting — an
// ignored file, one carrying `assume-unchanged` or `skip-worktree`, one
// a sparse checkout omits — reads as clean there while differing on
// disk. Neither side of this comparison consults the index or
// `.gitignore`, so no such mechanism can hide a path from it (G-0492).
//
// The comparison is between object ids computed over raw bytes, which
// is what makes it predict what an aiwf commit would carry: the verb
// commit path stores the working copy's bytes verbatim, so the object id
// a path would land with is the one `git hash-object --no-filters`
// reports. Applying git's clean filter to the working-copy side instead
// would compare against a convention these commits do not follow, and
// call a path divergent whose bytes the record already holds.
//
// A repo configured with content filters (`core.autocrlf`, a
// clean/smudge driver) is the case where that distinction is visible,
// and where it exposes a limit that is not this comparison's: a checkout
// smudges the working copy away from a blob git itself normalised, so
// the two genuinely differ and a verb genuinely would rewrite them. The
// filter-awareness of the commit path is tracked separately.
//
// Paths ride on git's argument vector rather than through a
// line-oriented stdin protocol, so a path containing a space or a
// newline is compared correctly rather than desynchronising a batch.
//
// Callers consult HasHEAD first: with an unborn HEAD there is no record
// to compare against, and every path on disk would read as absent from
// it.
//
// A path that exists but cannot be read is an error rather than a silent
// pass, since the comparison genuinely cannot be made. A path naming a
// directory is a caller error and reported as one.
func DivergentPaths(ctx context.Context, workdir string, paths []string) ([]Divergence, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	head, err := headEntries(ctx, workdir, paths)
	if err != nil {
		return nil, err
	}

	// Partition by what is actually on disk before asking git anything
	// about content: a path absent from the working tree needs no object
	// id, and a symlink must not be read through.
	var regular []string
	var out []Divergence
	symlinkTargets := map[string]string{}
	for _, p := range paths {
		info, statErr := os.Lstat(filepath.Join(workdir, filepath.FromSlash(p)))
		rec, recorded := head[p]
		switch {
		case statErr != nil && !notPresentOnDisk(statErr):
			return nil, fmt.Errorf("inspecting %s: %w", p, statErr)
		case statErr != nil:
			// Absent from disk — including the case where a parent
			// component is a file, which is a path that exists nowhere.
			// Divergent only if HEAD recorded it; a path in neither place
			// is not a disagreement.
			if recorded {
				out = append(out, Divergence{Path: p, Kind: DivergenceAbsentFromDisk})
			}
		case info.IsDir():
			return nil, fmt.Errorf("comparing %s: is a directory, not a file", p)
		case info.Mode()&os.ModeSymlink != 0:
			target, linkErr := os.Readlink(filepath.Join(workdir, filepath.FromSlash(p)))
			if linkErr != nil { //coverage:ignore defensive: the Lstat above just reported this path as a symlink, so Readlink fails only if it is replaced mid-call
				return nil, fmt.Errorf("reading link %s: %w", p, linkErr)
			}
			symlinkTargets[p] = target
		default:
			if !recorded {
				out = append(out, Divergence{Path: p, Kind: DivergenceAbsentFromHEAD})
				continue
			}
			if rec.IsSymlink {
				// The record holds a link where the working tree holds a
				// file. No object id comparison can express that.
				out = append(out, Divergence{Path: p, Kind: DivergenceModified})
				continue
			}
			regular = append(regular, p)
		}
	}

	if len(symlinkTargets) > 0 {
		linkDiffs, linkErr := divergentSymlinks(ctx, workdir, symlinkTargets, head)
		if linkErr != nil { //coverage:ignore defensive: divergentSymlinks fails only where hashLinkTarget does, which is annotated unreachable for the same reason
			return nil, linkErr
		}
		out = append(out, linkDiffs...)
	}

	if len(regular) > 0 {
		diskOIDs, oidErr := hashOnDisk(ctx, workdir, regular)
		if oidErr != nil {
			return nil, oidErr
		}
		for _, p := range regular {
			if diskOIDs[p] != head[p].OID {
				out = append(out, Divergence{Path: p, Kind: DivergenceModified})
			}
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// divergentSymlinks compares each link's target against the record.
// git stores a symlink's target string as the blob's content and no
// filter applies to it, so hashing the target through git is exact and
// needs no assumption about the repo's hash algorithm.
func divergentSymlinks(ctx context.Context, workdir string, targets map[string]string, head map[string]headEntry) ([]Divergence, error) {
	paths := make([]string, 0, len(targets))
	for p := range targets {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var out []Divergence
	for _, p := range paths {
		rec, recorded := head[p]
		if !recorded {
			out = append(out, Divergence{Path: p, Kind: DivergenceAbsentFromHEAD})
			continue
		}
		if !rec.IsSymlink {
			// The record holds a file where the working tree holds a link.
			out = append(out, Divergence{Path: p, Kind: DivergenceModified})
			continue
		}
		oid, err := hashLinkTarget(ctx, workdir, targets[p])
		if err != nil { //coverage:ignore defensive: hashLinkTarget feeds git bytes on stdin and cannot be rejected for their content; reaching here needs the subprocess itself to break
			return nil, fmt.Errorf("hashing link %s: %w", p, err)
		}
		if oid != rec.OID {
			out = append(out, Divergence{Path: p, Kind: DivergenceModified})
		}
	}
	return out, nil
}

// hashLinkTarget returns the object id git would store for a symlink
// pointing at target. Symlinks are rare enough that one subprocess each
// is not worth batching away, and routing the target through git rather
// than hashing it here keeps the repo's own hash algorithm authoritative.
func hashLinkTarget(ctx context.Context, workdir, target string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "hash-object", "--stdin")
	cmd.Dir = workdir
	cmd.Env = gitEnv()
	cmd.Stdin = strings.NewReader(target)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) { //coverage:ignore defensive: `git hash-object --stdin` reads bytes and cannot reject them; a failure needs the subprocess itself to break
			return "", fmt.Errorf("git hash-object --stdin: %w\n%s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("git hash-object --stdin: %w", err) //coverage:ignore defensive: a non-ExitError from a started process means the exec machinery failed, which every other git call in this package would fail on first
	}
	return strings.TrimSpace(string(out)), nil
}

// notPresentOnDisk reports whether a stat error means the working tree
// simply does not hold the path. ENOTDIR is included deliberately: a
// path whose parent component is a file is one the working tree does not
// hold, and treating it as an unreadable path would turn an ordinary
// record-versus-disk disagreement into a hard failure.
func notPresentOnDisk(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR)
}

// hashOnDisk returns the object id each regular file's bytes would land
// with, unfiltered — matching how the verb commit path stores content,
// so the result is comparable to what HEAD actually holds rather than to
// what `git add` would have produced.
//
// Paths ride on the argument vector, so any byte a filesystem permits in
// a name survives the round trip.
func hashOnDisk(ctx context.Context, workdir string, paths []string) (map[string]string, error) {
	oids := make(map[string]string, len(paths))
	for start := 0; start < len(paths); start += argvChunk {
		end := min(start+argvChunk, len(paths))
		chunk := paths[start:end]
		args := append([]string{"hash-object", "--no-filters", "--"}, chunk...)
		out, err := output(ctx, workdir, args...)
		if err != nil {
			return nil, fmt.Errorf("hashing working-copy paths: %w", err)
		}
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		if len(lines) != len(chunk) { //coverage:ignore defensive: git emits exactly one object id per requested path, and a read error surfaces above rather than as a short answer
			return nil, fmt.Errorf("hashing working-copy paths: got %d object ids for %d paths", len(lines), len(chunk))
		}
		for i, p := range chunk {
			oids[p] = strings.TrimSpace(lines[i])
		}
	}
	return oids, nil
}

// headEntries returns what HEAD records for each requested path, keyed
// by path. A path HEAD does not record is absent from the map; a path
// HEAD records as anything but a blob (a directory, a submodule) is
// likewise absent, since no file comparison applies to it.
func headEntries(ctx context.Context, workdir string, paths []string) (map[string]headEntry, error) {
	entries := make(map[string]headEntry, len(paths))
	for start := 0; start < len(paths); start += argvChunk {
		end := min(start+argvChunk, len(paths))
		args := append([]string{"ls-tree", "-z", "HEAD", "--"}, paths[start:end]...)
		out, err := output(ctx, workdir, args...)
		if err != nil {
			return nil, fmt.Errorf("reading HEAD entries under %s: %w", workdir, err)
		}
		for _, rec := range strings.Split(strings.TrimRight(out, "\x00"), "\x00") {
			if rec == "" {
				continue
			}
			// "<mode> SP <type> SP <oid> TAB <path>"
			meta, path, found := strings.Cut(rec, "\t")
			if !found { //coverage:ignore defensive: ls-tree -z always separates metadata from path with a tab
				continue
			}
			fields := strings.Fields(meta)
			if len(fields) != 3 || fields[1] != "blob" {
				continue
			}
			entries[path] = headEntry{OID: fields[2], IsSymlink: fields[0] == symlinkMode}
		}
	}
	return entries, nil
}

// symlinkMode is git's file mode for a symbolic link, whose blob content
// is the link's target string rather than the target's bytes.
const symlinkMode = "120000"
