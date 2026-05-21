package gitops

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// CommitRecord is one commit observed by [BulkRevwalk]: the commit's
// SHA, its parent SHAs in git's declared order (first-parent first),
// the paths it touched (with rename info when -M detected one), and
// the aiwf-* trailers parsed from the commit message.
//
// For multi-parent (merge) commits, the underlying `git log -m` emits
// one record per parent-diff — the same merge SHA may appear in
// multiple records, each carrying that parent's diff in Paths. The
// Parents field is identical across the duplicate records (it lists
// all parents). Consumers needing one-record-per-commit semantics
// dedupe by Commit SHA.
//
// Trailers is keyed by the bare trailer name (no "aiwf-" prefix
// stripping). Multi-value trailers collapse to the last value, matching
// internal/cli/history's existing single-value-per-key shape; consumers
// needing multi-value semantics use the [Trailer] slice form via
// [HeadTrailers] / [ParseTrailers] instead.
type CommitRecord struct {
	Commit   string
	Parents  []string
	Paths    []PathTouch
	Trailers map[string]string
}

// PathTouch is one path touched by a commit. Status is the git
// --name-status code: "A" added, "M" modified, "D" deleted, "R"
// renamed (SrcPath set to the pre-rename path), "C" copied (SrcPath
// set to the source path). The "T" (type change) code is rare in
// the aiwf planning tree (no symlinks, no submodules) and passes
// through unchanged.
type PathTouch struct {
	Status  string
	Path    string
	SrcPath string
}

// Sentinels used to delimit the `git log` output into per-commit
// records and per-record format-vs-paths blocks. Printable markers
// (rather than control bytes) so the format is readable when dumping
// raw git output during debugging, and so they survive any future
// tweak that strips low bytes. The collision risk against legitimate
// commit-body content is negligible: aiwf-produced bodies never
// contain `===AIWF-REC===` / `===AIWF-PATHS===` and any consumer
// commit that did contain them would be misparsed in the same way
// `internal/cli/history`'s `\x1e` would be — accepted theoretical risk.
const (
	bulkRecordMarker = "===AIWF-REC==="
	bulkPathsMarker  = "===AIWF-PATHS==="
)

// bulkTrailerKeys lists the aiwf-* trailer keys BulkRevwalk extracts
// per commit, in the order they appear in the pretty format. Keep in
// sync with the pretty-format string in [BulkRevwalk] — the parser
// uses the slice length as the trailer-field count.
var bulkTrailerKeys = []string{
	"aiwf-verb",
	"aiwf-entity",
	"aiwf-actor",
	"aiwf-force",
	"aiwf-audit-only",
	"aiwf-principal",
	"aiwf-on-behalf-of",
	"aiwf-authorized-by",
	"aiwf-scope",
	"aiwf-reason",
	"aiwf-tests",
}

// BulkRevwalk streams [CommitRecord] values from a single
// `git log --all --name-status -M -m --pretty=...` subprocess, calling
// fn for each commit-diff record in walk order. The single-subprocess
// shape replaces the per-entity `git log --follow` fan-out used by
// callers that walk every entity (fsm-history-consistent, status
// worktree views, show scope views) — collapsing ~3,000 fork/execs on
// the kernel tree into one long-lived process.
//
// If fn returns a non-nil error, BulkRevwalk halts the walk and
// returns that error verbatim (`errors.Is` works). Use this to
// short-circuit when the consumer has found what it needs.
//
// Returns nil (no error, no callbacks) when root is empty, is not a
// git repo, or is a repo with no commits — the same "nothing to walk"
// semantic as [internal/cli/history.readHistory] uses.
//
// The walk includes all reachable refs (--all) so feature-branch
// history is observed; -M enables rename detection (PathTouch.Status
// "R" with SrcPath set rather than separate D + A entries); -m forces
// per-parent diff fan-out so merge commits' name-status output is
// non-empty (a merge with N parents produces N records, each with the
// same Commit / Parents but its parent-specific Paths).
func BulkRevwalk(ctx context.Context, root string, fn func(CommitRecord) error) error {
	if root == "" {
		return nil
	}
	if !IsRepo(ctx, root) {
		return nil
	}
	if !hasAnyCommit(ctx, root) {
		return nil
	}

	pretty := buildBulkPretty()
	cmd := exec.CommandContext(ctx, "git", "log",
		"--all",
		"--name-status",
		"-M",
		"-m",
		"--pretty="+pretty,
	)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("git log: %w\n%s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return fmt.Errorf("git log: %w", err)
	}

	for _, chunk := range splitOnMarker(string(out), bulkRecordMarker) {
		rec, ok := parseBulkChunk(chunk)
		if !ok {
			continue
		}
		if err := fn(rec); err != nil {
			return err
		}
	}
	return nil
}

// buildBulkPretty assembles the --pretty=tformat string used by
// BulkRevwalk. Marker-first so splitOnMarker can detect record
// boundaries even when commit bodies contain newlines.
func buildBulkPretty() string {
	var b strings.Builder
	b.WriteString("tformat:")
	b.WriteString(bulkRecordMarker)
	b.WriteString("%n%H%x1f%P")
	for _, key := range bulkTrailerKeys {
		b.WriteString("%x1f%(trailers:key=")
		b.WriteString(key)
		b.WriteString(",valueonly=true,unfold=true)")
	}
	b.WriteString("%n")
	b.WriteString(bulkPathsMarker)
	b.WriteString("%n")
	return b.String()
}

// splitOnMarker splits raw on lines exactly equal to marker, returning
// non-empty chunks. Each chunk is the content between two markers (or
// between a marker and end-of-output).
//
// Robust against bodies that contain the marker as a substring: only a
// line equal-to-marker counts as a boundary, so embedded matches in a
// quoted code block (e.g. a body that quotes BulkRevwalk's own output)
// don't split the stream. The trade-off is identical to
// internal/cli/history's `\x1e` record-sep approach.
func splitOnMarker(raw, marker string) []string {
	var chunks []string
	var current strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(raw))
	// Bump the line buffer to 1 MiB — git log bodies can carry long
	// prose paragraphs; the default 64 KiB is fine for trailers but
	// thin for the worst body cases.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == marker {
			if current.Len() > 0 {
				chunks = append(chunks, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteString(line)
		current.WriteByte('\n')
	}
	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}
	return chunks
}

// parseBulkChunk parses one per-commit chunk produced by BulkRevwalk's
// `git log` invocation. The chunk shape is:
//
//	<full SHA><US><parents><US><trailer-1><US>...<trailer-N>
//	===AIWF-PATHS===
//	A\tpath\n
//	M\tpath\n
//	R100\told\tnew\n
//	...
//
// Returns ok=false for malformed chunks (missing paths marker, fewer
// fields than expected). Malformed chunks are dropped silently — the
// parser is tolerant of future format extensions that add fields, but
// halts cleanly on shapes it doesn't recognize.
func parseBulkChunk(chunk string) (CommitRecord, bool) {
	chunk = strings.TrimLeft(chunk, "\n")
	idx := strings.Index(chunk, "\n"+bulkPathsMarker+"\n")
	if idx < 0 {
		// Tolerate end-of-output where the trailing newline after the
		// paths marker is absent.
		idx = strings.Index(chunk, "\n"+bulkPathsMarker)
		if idx < 0 {
			return CommitRecord{}, false
		}
	}
	formatBlock := chunk[:idx]
	pathsBlock := ""
	pathsStart := idx + len("\n"+bulkPathsMarker)
	if pathsStart < len(chunk) {
		pathsBlock = strings.TrimLeft(chunk[pathsStart:], "\n")
	}

	fields := strings.Split(formatBlock, "\x1f")
	expectedFields := 2 + len(bulkTrailerKeys)
	if len(fields) < expectedFields {
		return CommitRecord{}, false
	}
	rec := CommitRecord{
		Commit: strings.TrimSpace(fields[0]),
	}
	if parents := strings.TrimSpace(fields[1]); parents != "" {
		rec.Parents = strings.Fields(parents)
	}
	rec.Trailers = parseBulkTrailers(fields[2:])
	rec.Paths = parsePathsBlock(pathsBlock)
	if rec.Commit == "" {
		return CommitRecord{}, false
	}
	return rec, true
}

// parseBulkTrailers reads N trailer fields (in bulkTrailerKeys order)
// into a map. Empty fields don't populate the map — the AC-2/3/4
// predicates use `_, ok := trailers["aiwf-force"]` style presence
// checks, so absent-vs-empty must not collapse.
func parseBulkTrailers(fields []string) map[string]string {
	if len(fields) == 0 {
		return nil
	}
	out := map[string]string{}
	for i, key := range bulkTrailerKeys {
		if i >= len(fields) {
			break
		}
		v := strings.TrimSpace(fields[i])
		if v == "" {
			continue
		}
		out[key] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// parsePathsBlock reads `git log --name-status` output (TAB-separated)
// into PathTouch values. Each line is one path-touch.
//
// Status codes from git's --name-status:
//
//   - "A" / "M" / "D" / "T": one path follows.
//   - "Rxxx" / "Cxxx" (rename / copy with similarity score): two paths
//     follow — source then destination. The score (e.g. R100, R092)
//     collapses to the bare "R" / "C" letter in [PathTouch].Status;
//     the score isn't load-bearing for any current consumer.
//
// Lines that don't match either shape are dropped silently (defensive
// — git's output is well-defined, but a future flag combination could
// emit a shape we don't expect, and dropping is safer than mis-
// classifying).
func parsePathsBlock(block string) []PathTouch {
	if block == "" {
		return nil
	}
	var out []PathTouch
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}
		statusRaw := parts[0]
		if statusRaw == "" {
			continue
		}
		statusCode := string(statusRaw[0])
		switch statusCode {
		case "R", "C":
			if len(parts) < 3 {
				continue
			}
			out = append(out, PathTouch{
				Status:  statusCode,
				SrcPath: parts[1],
				Path:    parts[2],
			})
		default:
			out = append(out, PathTouch{
				Status: statusCode,
				Path:   parts[1],
			})
		}
	}
	return out
}

// hasAnyCommit reports whether root's repo has at least one commit
// reachable from any ref (HEAD or otherwise). BulkRevwalk uses --all,
// so a repo with only a feature-branch tip and no HEAD still walks.
//
// Empty-repo detection is via `git rev-list --all -n 1`: exits 0 with
// one SHA when commits exist, exits 0 with empty output when no
// commits exist on any ref. Differentiated from [hasGitCommits]-style
// HEAD-only checks elsewhere in the kernel.
func hasAnyCommit(ctx context.Context, root string) bool {
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--all", "-n", "1")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}
