package check

import (
	"testing"

	"github.com/23min/aiwf/internal/gitops"
)

// TestParseHeadCommits covers WalkHeadCommits' pure parser: a valid
// two-record stream, a record with too few unit-separated fields
// (dropped), and a record with an empty SHA (dropped).
func TestParseHeadCommits(t *testing.T) {
	t.Parallel()
	const m = headRecMarker
	out := m + "\n" +
		"sha1\x1fauthor@a\x1fcommitter@a\x1faiwf-verb: add\x1fsubject one\nbody one\n" +
		m + "\n" +
		"\x1fauthor@b\x1fcommitter@b\x1faiwf-verb: edit\x1fempty-sha record\n" + // empty SHA → dropped
		m + "\n" +
		"sha3-too-few-fields\n" + // no \x1f separators → len(fields)<5 → dropped
		m + "\n" +
		"sha4\x1fa@x\x1fb@y\x1faiwf-verb: promote\x1fsubject four\n"

	got := parseHeadCommits(out)
	if len(got) != 2 {
		t.Fatalf("parseHeadCommits returned %d records, want 2 (sha1, sha4); got %+v", len(got), got)
	}
	if got[0].SHA != "sha1" || got[0].AuthorEmail != "author@a" || got[0].CommitterEmail != "committer@a" {
		t.Errorf("record 0 = %+v, want sha1/author@a/committer@a", got[0])
	}
	if got[1].SHA != "sha4" {
		t.Errorf("record 1 SHA = %q, want sha4", got[1].SHA)
	}
}

// TestWalkCherryPicks_FromHead exercises every WalkCherryPicks branch:
// the empty-head short-circuit, the both-signals predicate (identity gap
// AND marker), and the skips when either signal is absent.
func TestWalkCherryPicks_FromHead(t *testing.T) {
	t.Parallel()
	if got := WalkCherryPicks(nil); got != nil {
		t.Errorf("WalkCherryPicks(nil) = %v, want nil", got)
	}
	head := []HeadCommit{
		// qualifies: identity gap + marker.
		{SHA: "cp", AuthorEmail: "a@x", CommitterEmail: "b@y", Body: "pick\n\n(cherry picked from commit deadbeef)"},
		// gap but no marker → skip.
		{SHA: "gap-no-marker", AuthorEmail: "a@x", CommitterEmail: "b@y", Body: "no marker here"},
		// marker but no gap (same email) → skip.
		{SHA: "marker-no-gap", AuthorEmail: "a@x", CommitterEmail: "a@x", Body: "(cherry picked from commit abcd1234)"},
		// missing committer email → skip.
		{SHA: "no-committer", AuthorEmail: "a@x", CommitterEmail: "", Body: "(cherry picked from commit abcd1234)"},
	}
	got := WalkCherryPicks(head)
	if len(got) != 1 || !got["cp"] {
		t.Errorf("WalkCherryPicks = %v, want only {cp:true}", got)
	}
}

// TestWalkAuditOnlyAcksByEntity_FromHead covers the audit-only gather:
// empty head, the empty-SHA skip, the both-trailers requirement, and the
// composite-root canonicalization of the entity key.
func TestWalkAuditOnlyAcksByEntity_FromHead(t *testing.T) {
	t.Parallel()
	if got := walkAuditOnlyAcksByEntity(nil); got != nil {
		t.Errorf("walkAuditOnlyAcksByEntity(nil) = %v, want nil", got)
	}
	head := []HeadCommit{
		{SHA: "", Trailers: []gitops.Trailer{{Key: gitops.TrailerAuditOnly, Value: ""}, {Key: gitops.TrailerEntity, Value: "M-0001"}}}, // empty SHA → skip
		{SHA: "ack1", Trailers: []gitops.Trailer{{Key: gitops.TrailerAuditOnly, Value: ""}, {Key: gitops.TrailerEntity, Value: "M-001/AC-1"}}},
		{SHA: "not-audit", Trailers: []gitops.Trailer{{Key: gitops.TrailerEntity, Value: "M-0002"}}}, // no audit-only → skip
	}
	got := walkAuditOnlyAcksByEntity(head)
	// M-001/AC-1 rolls up to composite root M-0001 (canonical width).
	if shas := got["M-0001"]; len(shas) != 1 || shas[0] != "ack1" {
		t.Errorf("walkAuditOnlyAcksByEntity[M-0001] = %v, want [ack1]", got["M-0001"])
	}
	if len(got) != 1 {
		t.Errorf("walkAuditOnlyAcksByEntity returned %d entities, want 1; got %v", len(got), got)
	}
}
