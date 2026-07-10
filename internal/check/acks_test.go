package check

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// acks_test.go pins findDanglingAckHint (G-0395): a best-effort,
// local-clone-only diagnostic that looks for a commit no longer
// reachable from HEAD — but still present in the local object
// database as a dangling object — carrying an
// `aiwf-force-for: <targetSHA>` trailer. It never changes a
// finding's severity or blocking behavior; it only enriches the
// message when local evidence of a since-dropped acknowledgment is
// still recoverable.

// TestFindDanglingAckHint_FindsDanglingAcknowledgment reproduces the
// exact G-0395 sequence: an illegal transition is acknowledged, a
// later commit lands, then a rebase drops just the acknowledgment
// commit (git rebase --onto), leaving it dangling but not yet
// garbage-collected. findDanglingAckHint must find it and name it.
func TestFindDanglingAckHint_FindsDanglingAcknowledgment(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add E-0001")
	illegalSHA := r.commitEntity("E-0001", entity.KindEpic, entity.StatusDone,
		"skip-ahead proposed->done (FSM-illegal)")
	writeAcknowledgmentCommit(t, r.root, illegalSHA, "test ack")
	ackSHA := strings.TrimSpace(r.run("git", "rev-parse", "HEAD"))
	r.gitCommit("unrelated follow-up")

	// Drop just the ack commit, keeping the illegal commit and the
	// follow-up reachable.
	// The 2-arg form (no trailing branch arg) stays on and moves the
	// current branch; a 3rd literal "HEAD" argument instead detaches
	// HEAD and leaves the branch ref pointing at the pre-rebase tip —
	// which would keep the dropped commit reachable via that branch
	// and defeat this exact reproduction (confirmed empirically).
	r.run("git", "rebase", "-q", "--onto", illegalSHA, ackSHA)

	got := findDanglingAckHint(context.Background(), r.root, illegalSHA)
	if got == "" {
		t.Fatal("expected a non-empty hint naming the dangling acknowledgment; got empty string")
	}
	if !strings.Contains(got, ackSHA[:8]) {
		t.Errorf("hint %q does not name the dangling ack commit %s", got, ackSHA[:8])
	}
}

// TestFindDanglingAckHint_EmptyWhenNothingDangling confirms the
// negative case: a clean repo with no dangling objects at all
// returns "".
func TestFindDanglingAckHint_EmptyWhenNothingDangling(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add E-0001")
	illegalSHA := r.commitEntity("E-0001", entity.KindEpic, entity.StatusDone,
		"skip-ahead proposed->done (FSM-illegal, never acknowledged)")

	got := findDanglingAckHint(context.Background(), r.root, illegalSHA)
	if got != "" {
		t.Errorf("expected empty hint, got %q", got)
	}
}

// TestFindDanglingAckHint_EmptyWhenDanglingCommitTargetsADifferentSHA
// confirms per-SHA scoping: a dangling ack commit exists, but it
// targets a different SHA than the one being asked about.
func TestFindDanglingAckHint_EmptyWhenDanglingCommitTargetsADifferentSHA(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add E-0001")
	illegalSHA := r.commitEntity("E-0001", entity.KindEpic, entity.StatusDone,
		"skip-ahead proposed->done (FSM-illegal)")
	r.commitEntity("E-0002", entity.KindEpic, entity.StatusProposed, "add E-0002")
	otherIllegalSHA := r.commitEntity("E-0002", entity.KindEpic, entity.StatusDone,
		"skip-ahead proposed->done (FSM-illegal, different entity)")
	// Acknowledge the OTHER commit, then drop that ack via rebase (2-arg
	// form — see the rationale in the test above).
	writeAcknowledgmentCommit(t, r.root, otherIllegalSHA, "test ack for a different SHA")
	ackSHA := strings.TrimSpace(r.run("git", "rev-parse", "HEAD"))
	r.gitCommit("unrelated follow-up")
	r.run("git", "rebase", "-q", "--onto", otherIllegalSHA, ackSHA)

	// Asking about illegalSHA (E-0001's), not otherIllegalSHA (E-0002's).
	got := findDanglingAckHint(context.Background(), r.root, illegalSHA)
	if got != "" {
		t.Errorf("expected empty hint (dangling ack targets a different SHA), got %q", got)
	}
}

// TestFindDanglingAckHint_IgnoresDanglingNonCommitObjects is an
// end-to-end coexistence check: a dangling unreferenced blob object
// present alongside a genuine dangling acknowledgment commit must not
// disrupt finding the real one. This does NOT prove
// parseDanglingCommitSHAs' own type-filtering discriminates —
// downstream (git show on a blob) tends to return nothing useful
// regardless of whether the filter ran, so this test still passes
// even with the filter removed; TestParseDanglingCommitSHAs pins the
// filter itself, directly, against fabricated multi-type input.
func TestFindDanglingAckHint_IgnoresDanglingNonCommitObjects(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add E-0001")
	illegalSHA := r.commitEntity("E-0001", entity.KindEpic, entity.StatusDone,
		"skip-ahead proposed->done (FSM-illegal)")
	writeAcknowledgmentCommit(t, r.root, illegalSHA, "test ack")
	ackSHA := strings.TrimSpace(r.run("git", "rev-parse", "HEAD"))
	r.gitCommit("unrelated follow-up")
	r.run("git", "rebase", "-q", "--onto", illegalSHA, ackSHA)

	// An unreferenced blob, dangling alongside the ack commit.
	cmd := exec.Command("git", "hash-object", "-w", "--stdin")
	cmd.Dir = r.root
	cmd.Stdin = strings.NewReader("unrelated dangling blob content\n")
	if err := cmd.Run(); err != nil {
		t.Fatalf("git hash-object: %v", err)
	}

	got := findDanglingAckHint(context.Background(), r.root, illegalSHA)
	if got == "" {
		t.Fatal("expected the dangling ack commit's hint despite an unrelated dangling blob also present; got empty string")
	}
	if !strings.Contains(got, ackSHA[:8]) {
		t.Errorf("hint %q does not name the dangling ack commit %s", got, ackSHA[:8])
	}
}

// TestParseDanglingCommitSHAs pins the type-filtering logic directly
// against fabricated `git fsck --unreachable` output — a mix of
// commit, blob, and tree lines, plus malformed ones — since a real
// fsck run can't be coerced to emit a specific type mix on demand.
func TestParseDanglingCommitSHAs(t *testing.T) {
	t.Parallel()
	const commitA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const commitB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	const blobSHA = "cccccccccccccccccccccccccccccccccccccccc"
	const treeSHA = "dddddddddddddddddddddddddddddddddddddddd"
	fsckOutput := "unreachable commit " + commitA + "\n" +
		"unreachable blob " + blobSHA + "\n" +
		"unreachable tree " + treeSHA + "\n" +
		"unreachable commit " + commitB + "\n" +
		"\n" +
		"notable-garbage-line\n"

	got := parseDanglingCommitSHAs(fsckOutput)
	want := []string{commitA, commitB}
	if len(got) != len(want) {
		t.Fatalf("parseDanglingCommitSHAs() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("parseDanglingCommitSHAs()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestParseDanglingCommitSHAs_EmptyInput pins the no-dangling-objects
// case: empty fsck output yields a nil slice.
func TestParseDanglingCommitSHAs_EmptyInput(t *testing.T) {
	t.Parallel()
	if got := parseDanglingCommitSHAs(""); got != nil {
		t.Errorf("parseDanglingCommitSHAs(\"\") = %v, want nil", got)
	}
}

// TestFindDanglingAckHint_EmptyForNonGitDir pins the defensive path:
// git fsck fails outright (not a repo) — the function returns ""
// rather than propagating an error, since this is advisory-only.
func TestFindDanglingAckHint_EmptyForNonGitDir(t *testing.T) {
	t.Parallel()
	got := findDanglingAckHint(context.Background(), t.TempDir(), "0123456789abcdef0123456789abcdef01234567")
	if got != "" {
		t.Errorf("expected empty hint for a non-git directory, got %q", got)
	}
}
