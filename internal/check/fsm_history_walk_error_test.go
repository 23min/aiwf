package check

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
)

// M-0137/AC-4 + AC-5 RED tests. The contracts these pin:
//
//   - AC-4: when a walker encounters an error (subprocess crash,
//     protocol violation, per-blob read failure), the rule emits a
//     `fsm-history-consistent/history-walk-error` finding (severity
//     error) naming the affected entity. Today the rule silently
//     swallows walker errors at FSMHistoryConsistent:71-77 and
//     returns nil — the silent-swallow this milestone exists to fix.
//
//   - AC-5: when the walker partially fails (some entities walked
//     successfully, others errored), the rule emits findings for the
//     successful portion alongside `history-walk-error` for the
//     failed portion. Today the per-entity walker fail-fasts at
//     fsm_history_consistent.go:154 and the entry-point swallows
//     even partial work — no findings emerge at all.
//
// Both tests fail at their assertion lines today. GREEN phase
// (deferred to a fresh session) rewires the walker to use
// gitops.BulkRevwalk + gitops.BlobReader, surfaces walker errors as
// findings, and continues past per-blob failures.

// TestFSMHistoryConsistent_AC4_CancelledContext_EmitsWalkError —
// AC-4 contract: a cancelled context causes the walker's underlying
// subprocess to fail; the rule must emit a history-walk-error
// finding rather than silently returning nil.
//
// RED state: today FSMHistoryConsistent returns nil for any walker
// error (the documented silent-swallow at line 71-77). The
// assertion `expected a fsm-history-consistent/history-walk-error
// finding` fails.
//
// GREEN state: the new walker emits a finding with Code=
// "fsm-history-consistent", Subcode="history-walk-error", Severity=
// SeverityError; the assertion passes.
func TestFSMHistoryConsistent_AC4_CancelledContext_EmitsWalkError(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add")
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusDone, "skip-ahead illegal")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel: every subprocess call sees ctx.Err() immediately

	got := FSMHistoryConsistent(ctx, r.root, r.tree())

	var hasWalkError bool
	for _, f := range got {
		if f.Code == CodeFSMHistoryConsistent && f.Subcode == "history-walk-error" {
			if f.Severity != SeverityError {
				t.Errorf("history-walk-error severity = %q, want error", f.Severity)
			}
			hasWalkError = true
		}
	}
	if !hasWalkError {
		t.Errorf("expected a fsm-history-consistent/history-walk-error finding from cancelled walker; got %d finding(s): %+v",
			len(got), got)
	}
}

// TestFSMHistoryConsistent_AC5_PartialFailure_PreservesGoodFindings —
// AC-5 contract: when one entity's blob read fails but another
// entity's walk succeeds, the rule emits findings for the successful
// portion (illegal-transition / etc.) alongside `history-walk-error`
// for the failed portion. Today the per-entity walker fail-fasts on
// the first error and the entry-point's swallow drops everything;
// no partial findings emerge.
//
// Uses the blobReader dep seam (fsmHistoryConsistentWithDeps) so the
// failure is injected deterministically rather than relying on
// subprocess timing.
//
// RED state: today fsmHistoryConsistentWithDeps delegates to
// FSMHistoryConsistent and ignores the fake reader. No
// history-walk-error finding emerges → the assertion `expected a
// history-walk-error for E-0002` fails.
//
// GREEN state: the new walker routes blob reads through the dep,
// surfaces per-read errors as history-walk-error findings, and
// continues processing other commits/entities. The illegal-transition
// for E-0001 emerges AND the history-walk-error for E-0002 emerges.
func TestFSMHistoryConsistent_AC5_PartialFailure_PreservesGoodFindings(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add E-0001")
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusDone, "skip-ahead illegal")
	r.commitEntity("E-0002", entity.KindEpic, entity.StatusProposed, "add E-0002")
	r.commitEntity("E-0002", entity.KindEpic, entity.StatusActive, "promote E-0002")

	tr := r.tree()
	// Fake blobReader: errors on any read targeting E-0002's path,
	// delegates to a real BlobReader for everything else. Delegation
	// matters: E-0001's reads need to return the actual on-disk
	// frontmatter so its illegal-transition observation emerges
	// normally (a fake returning constant content would mask the
	// transition).
	delegate, err := gitops.NewBlobReader(context.Background(), r.root)
	if err != nil {
		t.Fatalf("NewBlobReader: %v", err)
	}
	defer delegate.Close()
	fake := &fakeBlobReader{
		delegate:        delegate,
		errOnPathPrefix: "work/epics/E-0002",
		readErr:         errors.New("synthetic blob read failure"),
	}

	got := fsmHistoryConsistentWithDeps(context.Background(), r.root, tr, fake)

	var hasIllegalE0001, hasWalkErrorE0002 bool
	for _, f := range got {
		switch {
		case f.Code == CodeFSMHistoryConsistent && f.Subcode == "illegal-transition" && f.EntityID == "E-0001":
			hasIllegalE0001 = true
		case f.Code == CodeFSMHistoryConsistent && f.Subcode == "history-walk-error" && f.EntityID == "E-0002":
			hasWalkErrorE0002 = true
		}
	}
	if !hasIllegalE0001 {
		t.Errorf("expected illegal-transition finding for E-0001 (good portion preserved); got %d finding(s): %+v",
			len(got), got)
	}
	if !hasWalkErrorE0002 {
		t.Errorf("expected history-walk-error finding for E-0002 (failed portion surfaced); got %d finding(s): %+v",
			len(got), got)
	}
}

// fakeBlobReader implements the blobReader interface for AC-5's
// partial-failure test. Reads with path matching errOnPathPrefix
// return readErr; all other reads delegate to the underlying real
// BlobReader. Delegation preserves the on-disk semantics for the
// "success" portion of the partial-failure scenario so the legitimate
// findings (illegal-transition / forced-untrailered / etc.) emerge as
// they would in production.
//
// Defined here (not in fsm_history_consistent.go) because it's a
// test-only fixture. Tests live in package check (internal), so the
// fake can satisfy the unexported blobReader interface directly.
type fakeBlobReader struct {
	delegate        *gitops.BlobReader
	errOnPathPrefix string
	readErr         error
}

func (f *fakeBlobReader) Read(commit, path string) ([]byte, error) {
	if f.errOnPathPrefix != "" && strings.HasPrefix(path, f.errOnPathPrefix) {
		return nil, f.readErr
	}
	return f.delegate.Read(commit, path)
}

func (f *fakeBlobReader) Close() error {
	// Close is owned by the test (which holds the delegate).
	return nil
}
