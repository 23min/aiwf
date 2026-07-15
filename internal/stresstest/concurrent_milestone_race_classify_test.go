package stresstest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// concurrent_milestone_race_classify_test.go pins the pure helpers
// behind ConcurrentMilestoneRaceScenario: raceActorArgs (which argv
// one actor's operation builds) and buildRaceOutcome (how one actor's
// decoded envelope reduces to a raceActorOutcome) — both split out of
// Run/launchActor so their branches are deterministically
// unit-testable without depending on real race timing to exercise both
// sides (M-0258/AC-1) — plus parseRaceCommitSHAs and
// classifyMilestoneRaceOutcomes, AC-2's own legitimate-race-vs-
// violation oracle, exercised against fabricated outcome/commit-order
// inputs rather than real race timing (M-0258/AC-2).

func TestRaceActorArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		operation   string
		milestoneID string
		want        []string
	}{
		{
			name:        "promote targets the milestone's AC-1 composite id with the met target",
			operation:   raceOpPromote,
			milestoneID: "M-0007",
			want:        []string{"promote", "M-0007/AC-1", "met", "--format=json"},
		},
		{
			name:        "cancel targets the milestone itself with a reason",
			operation:   raceOpCancel,
			milestoneID: "M-0007",
			want:        []string{"cancel", "M-0007", "--reason", "concurrent-milestone-race probe", "--format=json"},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := raceActorArgs(tc.operation, tc.milestoneID)
			if len(got) != len(tc.want) {
				t.Fatalf("raceActorArgs(%q, %q) = %v, want %v", tc.operation, tc.milestoneID, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("raceActorArgs(%q, %q)[%d] = %q, want %q", tc.operation, tc.milestoneID, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestBuildRaceOutcome(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		operation     string
		env           verbEnvelope
		wantStatus    string
		wantErrorCode string
	}{
		{
			name:          "a successful actor carries no error code",
			operation:     raceOpPromote,
			env:           verbEnvelope{Status: "ok"},
			wantStatus:    "ok",
			wantErrorCode: "",
		},
		{
			name:      "a refused actor carries the refusal's typed code",
			operation: raceOpCancel,
			env: verbEnvelope{
				Status: "error",
				Error:  &verbEnvelopeError{Code: "milestone-cancel-non-terminal-acs", Message: "refused"},
			},
			wantStatus:    "error",
			wantErrorCode: "milestone-cancel-non-terminal-acs",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := buildRaceOutcome(tc.operation, tc.env)
			if got.operation != tc.operation {
				t.Errorf("operation = %q, want %q", got.operation, tc.operation)
			}
			if got.status != tc.wantStatus {
				t.Errorf("status = %q, want %q", got.status, tc.wantStatus)
			}
			if got.errorCode != tc.wantErrorCode {
				t.Errorf("errorCode = %q, want %q", got.errorCode, tc.wantErrorCode)
			}
		})
	}
}

// TestConcurrentMilestoneRaceExpectedWarnings pins the scenario's own
// baseline map (M-0257/AC-1's convention), derived empirically by
// running the scenario repeatedly (see concurrent_milestone_race.go's
// doc comment): a provenance-scope-undefined warning is always
// accepted noise; the archive-sweep advisory pair is accepted because a
// legitimate race outcome can land the milestone at the terminal
// `cancelled` status, which this scenario never sweeps.
func TestConcurrentMilestoneRaceExpectedWarnings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		findings       []verbEnvelopeFinding
		wantViolations int
	}{
		{name: "no findings", findings: nil, wantViolations: 0},
		{
			name:           "the baseline provenance-scope-undefined warning is accepted",
			findings:       []verbEnvelopeFinding{{Code: check.CodeProvenanceUntrailedScopeUndefined, Severity: "warning"}},
			wantViolations: 0,
		},
		{
			name:           "the baseline archive-sweep-pending warning is accepted",
			findings:       []verbEnvelopeFinding{{Code: check.CodeArchiveSweepPending, Severity: "warning"}},
			wantViolations: 0,
		},
		{
			name:           "the baseline terminal-entity-not-archived warning is accepted",
			findings:       []verbEnvelopeFinding{{Code: check.CodeTerminalEntityNotArchived, Severity: "warning"}},
			wantViolations: 0,
		},
		{
			name:           "an unbaselined warning code is a violation",
			findings:       []verbEnvelopeFinding{{Code: "some-unexpected-code", Severity: "warning"}}, //enums:ignore deliberately fabricated non-code for the test, not a real finding
			wantViolations: 1,
		},
		{
			name:           "an error-severity finding is a violation even for a baselined code",
			findings:       []verbEnvelopeFinding{{Code: check.CodeProvenanceUntrailedScopeUndefined, Severity: "error"}},
			wantViolations: 1,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyAgainstBaseline(tc.findings, concurrentMilestoneRaceExpectedWarnings)
			if len(got) != tc.wantViolations {
				t.Fatalf("violations = %+v, want %d", got, tc.wantViolations)
			}
		})
	}
}

// TestParseRaceCommitSHAs pins parseRaceCommitSHAs's own reduction of
// readRaceCommitOrder's raw `git log --reverse --format=%H` output
// into an ordered []string, split out of readRaceCommitOrder so its
// blank-line-skipping branch is testable without a real git
// subprocess.
func TestParseRaceCommitSHAs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		out  []byte
		want []string
	}{
		{
			name: "empty output yields no SHAs",
			out:  []byte(""),
			want: nil,
		},
		{
			name: "three SHAs parse in oldest-first order",
			out:  []byte("aaa\nbbb\nccc\n"),
			want: []string{"aaa", "bbb", "ccc"},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parseRaceCommitSHAs(tc.out)
			if len(got) != len(tc.want) {
				t.Fatalf("parseRaceCommitSHAs(%q) = %+v, want %+v", tc.out, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("parseRaceCommitSHAs(%q)[%d] = %q, want %q", tc.out, i, got[i], tc.want[i])
				}
			}
		})
	}
}

// TestReadRaceCommitOrder_RealGit pins readRaceCommitOrder's own
// happy-path integration (SHA listing + per-SHA trailer reads) against
// a real repo carrying real aiwf-verb/aiwf-entity-trailered commits,
// via writeRaceTestCommit below.
func TestReadRaceCommitOrder_RealGit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := gitInitAndConfig(dir); err != nil {
		t.Fatalf("gitInitAndConfig: %v", err)
	}
	writeRaceTestCommit(t, dir, "first.txt", "add", "M-0100")
	writeRaceTestCommit(t, dir, "second.txt", "promote", "M-0100/AC-1")

	got, err := readRaceCommitOrder(dir)
	if err != nil {
		t.Fatalf("readRaceCommitOrder: %v", err)
	}
	want := []raceCommit{
		{verb: "add", entity: "M-0100"},
		{verb: "promote", entity: "M-0100/AC-1"},
	}
	if len(got) != len(want) {
		t.Fatalf("readRaceCommitOrder = %+v, want %+v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("readRaceCommitOrder[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// writeRaceTestCommit writes filename (trivial content) and commits it
// in dir carrying aiwf-verb/aiwf-entity trailers — a hand-rolled
// stand-in for a real `aiwf` mutating-verb commit, so
// TestReadRaceCommitOrder_RealGit can pin the trailer-reading pipeline
// without a real aiwf subprocess.
func writeRaceTestCommit(t *testing.T, dir, filename, verbName, entityID string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, filename), []byte("content\n"), 0o644); err != nil {
		t.Fatalf("writing %s: %v", filename, err)
	}
	if err := runGit(dir, "add", filename); err != nil {
		t.Fatalf("git add %s: %v", filename, err)
	}
	msg := fmt.Sprintf("test commit\n\naiwf-verb: %s\naiwf-entity: %s\n", verbName, entityID)
	if err := runGit(dir, "commit", "-q", "-m", msg); err != nil {
		t.Fatalf("git commit: %v", err)
	}
}

// TestReadRaceCommitOrder_ErrorsOnNonGitDir pins readRaceCommitOrder's
// own error branch via a directory with no git repo at all — the same
// direct-error-path style commitTrailerValue's own
// TestCommitTrailerValue_ErrorsOnUnreadableRef test uses.
func TestReadRaceCommitOrder_ErrorsOnNonGitDir(t *testing.T) {
	t.Parallel()
	if _, err := readRaceCommitOrder(t.TempDir()); err == nil {
		t.Fatal("expected readRaceCommitOrder to error against a non-git directory")
	}
}

// TestClassifyMilestoneRaceOutcomes pins AC-2's oracle: the two-signal
// judgment (outcome-shape/refusal-reason, plus commit-order causality)
// classifyMilestoneRaceOutcomes applies to a concurrent-milestone-race
// run's outcomes. Every "legitimate race" case below must produce
// ZERO violations — the AC's own explicit warning that over-eager
// classification would make every green run meaningless.
func TestClassifyMilestoneRaceOutcomes(t *testing.T) {
	t.Parallel()
	const milestoneID = "M-0100"
	const acEntity = milestoneID + "/AC-1"

	promoteOK := raceActorOutcome{operation: raceOpPromote, status: "ok"}
	promoteRefused := raceActorOutcome{operation: raceOpPromote, status: "error", errorCode: entity.CodeFSMTransitionIllegal.ID}
	cancelOK := raceActorOutcome{operation: raceOpCancel, status: "ok"}
	cancelRefusedOpenAC := raceActorOutcome{operation: raceOpCancel, status: "error", errorCode: verb.CodeMilestoneCancelNonTerminalACs.ID}
	cancelRefusedAlreadyCancelled := raceActorOutcome{operation: raceOpCancel, status: "error", errorCode: entity.CodeFSMTransitionIllegal.ID}

	promoteCommit := raceCommit{verb: raceOpPromote, entity: acEntity}
	cancelCommit := raceCommit{verb: raceOpCancel, entity: milestoneID}

	tests := []struct {
		name           string
		outcomes       []raceActorOutcome
		order          []raceCommit
		wantSubstrings []string
	}{
		{
			name: "legitimate race, no cancel wins — zero violations",
			outcomes: []raceActorOutcome{
				promoteOK, promoteRefused, promoteRefused, promoteRefused,
				cancelRefusedOpenAC, cancelRefusedOpenAC, cancelRefusedOpenAC, cancelRefusedOpenAC,
			},
			order:          []raceCommit{promoteCommit},
			wantSubstrings: nil,
		},
		{
			name: "legitimate race, a cancel wins after the promote — zero violations",
			outcomes: []raceActorOutcome{
				promoteOK, promoteRefused, promoteRefused, promoteRefused,
				cancelOK, cancelRefusedOpenAC, cancelRefusedAlreadyCancelled, cancelRefusedAlreadyCancelled,
			},
			order:          []raceCommit{promoteCommit, cancelCommit},
			wantSubstrings: nil,
		},
		{
			name: "zero promote actors succeed — a mutually-exclusive-transition violation (the AC's own open -> met never landed at all)",
			outcomes: []raceActorOutcome{
				promoteRefused, promoteRefused, promoteRefused, promoteRefused,
				cancelRefusedOpenAC, cancelRefusedOpenAC, cancelRefusedOpenAC, cancelRefusedOpenAC,
			},
			order:          nil,
			wantSubstrings: []string{"want exactly 1"},
		},
		{
			name: "two promote actors both ok — a mutually-exclusive-transition violation",
			outcomes: []raceActorOutcome{
				promoteOK, promoteOK, promoteRefused, promoteRefused,
				cancelRefusedOpenAC, cancelRefusedOpenAC, cancelRefusedOpenAC, cancelRefusedOpenAC,
			},
			order:          []raceCommit{promoteCommit},
			wantSubstrings: []string{"want exactly 1"},
		},
		{
			name: "two cancel actors both ok — a mutually-exclusive-transition violation",
			outcomes: []raceActorOutcome{
				promoteOK, promoteRefused, promoteRefused, promoteRefused,
				cancelOK, cancelOK, cancelRefusedAlreadyCancelled, cancelRefusedAlreadyCancelled,
			},
			order:          []raceCommit{promoteCommit, cancelCommit},
			wantSubstrings: []string{"want at most 1"},
		},
		{
			name: "a promote refusal carries an unexpected error code — contradicts the FSM's own verdict",
			outcomes: []raceActorOutcome{
				promoteOK,
				{operation: raceOpPromote, status: "error", errorCode: "some-unexpected-code"},
				promoteRefused, promoteRefused,
				cancelRefusedOpenAC, cancelRefusedOpenAC, cancelRefusedOpenAC, cancelRefusedOpenAC,
			},
			order:          []raceCommit{promoteCommit},
			wantSubstrings: []string{"contradicts the FSM's own verdict"},
		},
		{
			name: "a cancel refusal carries an unexpected error code — contradicts the guard or the FSM's own verdict",
			outcomes: []raceActorOutcome{
				promoteOK, promoteRefused, promoteRefused, promoteRefused,
				{operation: raceOpCancel, status: "error", errorCode: "some-unexpected-code"},
				cancelRefusedOpenAC, cancelRefusedOpenAC, cancelRefusedOpenAC,
			},
			order:          []raceCommit{promoteCommit},
			wantSubstrings: []string{"contradicts"},
		},
		{
			name: "a cancel actor reports ok but its commit landed before the promote commit — the G-0335 regression shape",
			outcomes: []raceActorOutcome{
				promoteOK, promoteRefused, promoteRefused, promoteRefused,
				cancelOK, cancelRefusedOpenAC, cancelRefusedAlreadyCancelled, cancelRefusedAlreadyCancelled,
			},
			order:          []raceCommit{cancelCommit, promoteCommit}, // cancel BEFORE promote
			wantSubstrings: []string{"the open-AC guard did not hold"},
		},
		{
			name: "a cancel actor reports ok but no promote commit is found in the order at all — malformed input",
			outcomes: []raceActorOutcome{
				promoteOK, promoteRefused, promoteRefused, promoteRefused,
				cancelOK, cancelRefusedOpenAC, cancelRefusedAlreadyCancelled, cancelRefusedAlreadyCancelled,
			},
			order:          []raceCommit{cancelCommit},
			wantSubstrings: []string{"no " + raceOpPromote + " commit"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyMilestoneRaceOutcomes(tc.outcomes, tc.order, milestoneID)
			if len(got) != len(tc.wantSubstrings) {
				t.Fatalf("violations = %+v, want %d matching %v", got, len(tc.wantSubstrings), tc.wantSubstrings)
			}
			for _, want := range tc.wantSubstrings {
				found := false
				for _, v := range got {
					if strings.Contains(v.Message, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("no violation contained %q; got %+v", want, got)
				}
			}
		})
	}
}
