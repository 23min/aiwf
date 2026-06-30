package check

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/23min/aiwf/internal/gitops"
)

// head_history.go — E-0053 / M-0216 AC-5: the shared per-check
// HEAD-reachable history walk.
//
// Before this, five gather rules each spawned their own `git log HEAD`
// over the same reachable history: WalkAcknowledgedSHAs,
// WalkAcknowledgedSHAEntities, walkAuditOnlyAcksByEntity,
// WalkCherryPicks, and readProvenanceCommits. WalkHeadCommits walks
// that history ONCE into a typed slice; each of those rules now derives
// its result from the slice in-memory (preserving its exact predicate)
// rather than re-walking. The CLI gather layer computes the slice once
// per check invocation and threads it through — the same single-compute
// / cascading-pass-through pattern the ackedSHAs map already uses (and
// the acks_helper_lift policy still pins WalkAcknowledgedSHAs as the
// single ackedSHAs source).

// headRecMarker delimits per-commit records in WalkHeadCommits' git
// output. A printable marker (rather than a control byte) so the
// stream is debuggable; the collision risk against legitimate body
// content is the same negligible, accepted risk gitops.BulkRevwalk
// takes with its own markers (an aiwf body never contains this line).
const headRecMarker = "===AIWF-HEADREC==="

// HeadCommit is one HEAD-reachable commit captured by WalkHeadCommits:
// the union of fields the five trailer-reading gather rules need.
//
// Trailers is parsed once (from `%(trailers:unfold=true)`) and shared;
// AuthorEmail / CommitterEmail feed the cherry-pick identity-gap check;
// Body feeds the cherry-pick marker match and the provenance
// aiwf-trailer grep.
//
// Trailer-shape assumption (M-0216 AC-5): the provenance gather replaced
// `%(trailers:only=true,unfold=true)` with this `%(trailers:unfold=true)`
// block, which is byte-identical ONLY while no commit's trailer block
// carries a non-trailer, colon-bearing line (e.g. `Note: see http://…`)
// that `gitops.ParseTrailers` would read as a trailer. That holds for
// every aiwf-produced commit and was verified across the whole kernel
// tree; a consumer commit that violated it could synthesize a trailer
// the `only=true` path dropped. A latent constraint, not a structural
// invariant.
type HeadCommit struct {
	SHA            string
	Trailers       []gitops.Trailer
	AuthorEmail    string
	CommitterEmail string
	Body           string
}

// WalkHeadCommits walks HEAD's reachable history once, oldest-first,
// and returns one HeadCommit per commit. Oldest-first matches the
// `--reverse` order readProvenanceCommits depends on; the map-building
// consumers (acks, audit-only, cherry-picks) are order-insensitive.
//
// Returns (nil, nil) for an empty root, a non-git directory, or a repo
// with no commits yet — the consumers treat a nil slice as "no commits
// / no exemptions".
//
// Fail-loud contract (M-0216 Finding 1): a catastrophic `git log HEAD`
// failure — a repo whose HEAD resolves (so hasGitCommits is true) but
// whose reachable history cannot be read, e.g. a corrupt object store
// or a partial/blobless clone missing a reachable commit — returns a
// non-nil error. The sole production caller (internal/cli/check) fails
// the whole check with ExitInternal on that error rather than reading
// the unreadable history as "no commits". This restores the fail-loud
// behavior the pre-refactor readProvenanceCommits had: a shared walk
// that silently returned nil here would disable the provenance /
// isolation-escape / promote-on-wrong-branch integrity checks on a
// degraded repo with no signal at all. The error path is outside the
// byte-identical claim's domain — a healthy tree never triggers it.
//
// One `git log --reverse HEAD` subprocess replaces the five the gather
// rules used to each spawn (E-0053 / M-0216 AC-5).
func WalkHeadCommits(ctx context.Context, root string) ([]HeadCommit, error) {
	if root == "" || !hasGitCommits(ctx, root) {
		return nil, nil
	}
	// Marker-first, then US-separated fixed fields, with %B (the body,
	// which carries newlines) LAST so a SplitN on the unit separator
	// keeps the body intact. Field order: SHA, author-email,
	// committer-email, trailers, body.
	format := "tformat:" + headRecMarker + "%n%H%x1f%ae%x1f%ce%x1f%(trailers:unfold=true)%x1f%B"
	cmd := exec.CommandContext(ctx, "git", "log", "--reverse", "--pretty="+format, "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("walking HEAD history in %s: %w", root, err)
	}
	return parseHeadCommits(string(out)), nil
}

// parseHeadCommits splits WalkHeadCommits' marker-delimited output into
// HeadCommit values. Each record is the text between two marker lines:
//
//	<SHA><US><author-email><US><committer-email><US><trailers…><US><body…>
//
// Trailers and body may both span multiple lines; SplitN with a limit
// of 5 keeps the body (the last field) whole.
func parseHeadCommits(out string) []HeadCommit {
	var commits []HeadCommit
	for _, chunk := range splitOnLineMarker(out, headRecMarker) {
		chunk = strings.TrimPrefix(chunk, "\n")
		fields := strings.SplitN(chunk, "\x1f", 5)
		if len(fields) < 5 {
			continue
		}
		sha := strings.TrimSpace(fields[0])
		if sha == "" {
			continue
		}
		commits = append(commits, HeadCommit{
			SHA:            sha,
			AuthorEmail:    strings.TrimSpace(fields[1]),
			CommitterEmail: strings.TrimSpace(fields[2]),
			Trailers:       gitops.ParseTrailers(fields[3]),
			Body:           fields[4],
		})
	}
	return commits
}

// splitOnLineMarker splits raw into the chunks that follow each line
// exactly equal to marker (content before the first marker is
// discarded — tformat emits the marker first). Only a whole-line match
// counts as a boundary, so a marker substring embedded in body prose
// does not split the stream (same robustness as
// gitops.BulkRevwalk's splitter).
func splitOnLineMarker(raw, marker string) []string {
	var chunks []string
	var cur strings.Builder
	started := false
	for _, line := range strings.Split(raw, "\n") {
		if line == marker {
			if started {
				chunks = append(chunks, cur.String())
				cur.Reset()
			}
			started = true
			continue
		}
		if started {
			cur.WriteString(line)
			cur.WriteByte('\n')
		}
	}
	if started && cur.Len() > 0 {
		chunks = append(chunks, cur.String())
	}
	return chunks
}
