package integration

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// TestAuthorizeEnd_ReEnding_ConvergesToNoOp is half of M-0325/AC-2.
//
// Re-running the verb against a scope already ended reports exit 0 and
// writes nothing. The unmoved HEAD is the load-bearing half: an
// implementation that emitted a second aiwf-scope-ends: for the same SHA
// would also exit 0, and a test reading only the exit code could not
// tell the two apart — the kernel's one-mutation-one-commit invariant is
// what the second commit would break.
func TestAuthorizeEnd_ReEnding_ConvergesToNoOp(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)

	root, binDir := sovereignScopedRepo(t, nil)
	if out, err := testutil.RunBin(t, root, binDir, nil,
		"authorize", "E-0001", "--end", "--reason", "first end"); err != nil {
		t.Fatalf("first end: %v\n%s", err, out)
	}

	before := headSHA(t, root)
	out, err := testutil.RunBin(t, root, binDir, nil,
		"authorize", "E-0001", "--scope", scopeSHAOf(t, root, binDir, "E-0001"),
		"--end", "--reason", "second end")
	if err != nil {
		t.Fatalf("re-ending an ended scope exited non-zero: %v\n%s", err, out)
	}
	if after := headSHA(t, root); after != before {
		t.Errorf("HEAD moved %s -> %s; a converging re-run writes no commit", before[:8], after[:8])
	}
	if !strings.Contains(out, "already ended") {
		t.Errorf("output does not report the state as already holding, so an operator cannot tell "+
			"convergence from a fresh end:\n%s", out)
	}
}

// TestAuthorizeEnd_UnresolvableTargets_Refuse is the other half of
// M-0325/AC-2, and it pins the R1-before-R2 ordering rather than the two
// refusals separately.
//
// Both cases run against the same repo state — one scope, already
// ended — so the only thing distinguishing them is whether the argument
// names something real. Naming that scope converges (the case above);
// naming a SHA that matches nothing refuses. An implementation that
// resolved only over non-ended scopes would collapse both into the same
// refusal, and one that converged whenever anything was ended would
// collapse both into the same success. Neither collapse is visible
// without the pair.
func TestAuthorizeEnd_UnresolvableTargets_Refuse(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)

	cases := []struct {
		name        string
		setup       [][]string
		args        []string
		wantRefusal string
	}{
		{
			name:  "--scope names no scope on the entity",
			setup: [][]string{{"authorize", "E-0001", "--end", "--reason", "ending the only scope"}},
			// Hex, and not a prefix of the real auth SHA: a malformed
			// value would be refused by the trailer shape rule instead,
			// which is a different guard than the one under test.
			args:        []string{"authorize", "E-0001", "--end", "--scope", "0123456", "--reason", "typo"},
			wantRefusal: "no scope on E-0001 matches",
		},
		{
			name: "bare --end on an entity that never had a scope",
			// M-0001 carries no scope; the fixture opens one on E-0001.
			args:        []string{"authorize", "M-0001", "--end", "--reason", "nothing to end"},
			wantRefusal: "no non-ended scope on M-0001 to end",
		},
		{
			name:        "bare --end when every scope is already ended",
			setup:       [][]string{{"authorize", "E-0001", "--end", "--reason", "ending the only scope"}},
			args:        []string{"authorize", "E-0001", "--end", "--reason", "again"},
			wantRefusal: "no non-ended scope on E-0001 to end",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root, binDir := sovereignScopedRepo(t, nil)
			for _, args := range tc.setup {
				if out, err := testutil.RunBin(t, root, binDir, nil, args...); err != nil {
					t.Fatalf("setup aiwf %v: %v\n%s", args, err, out)
				}
			}

			before := headSHA(t, root)
			out, err := testutil.RunBin(t, root, binDir, nil, tc.args...)
			if err == nil {
				t.Fatalf("verb succeeded; an unresolvable target must refuse rather than converge, "+
					"which would assert success for state that cannot exist\n%s", out)
			}
			if after := headSHA(t, root); after != before {
				t.Errorf("HEAD moved %s -> %s; the refusal must precede any commit", before[:8], after[:8])
			}
			if !strings.Contains(out, tc.wantRefusal) {
				t.Errorf("refusal does not contain %q, so it came from some other rule:\n%s", tc.wantRefusal, out)
			}
		})
	}
}

// TestAuthorizeEnd_AmbiguousTarget_RefusesAndListsCandidates covers the
// case ADR-0047 introduced --scope for: more than one non-ended scope on
// one entity, which two authorize --to calls naming different agents
// produce and nothing refuses.
//
// Ending is terminal, so guessing is not available the way pause's
// most-recently-opened convention is — a pause picked wrongly is undone
// by a resume, and an end picked wrongly is not. The listing is asserted
// because a refusal that named no candidates would leave the operator
// running `aiwf show` to recover what the verb already had in hand.
func TestAuthorizeEnd_AmbiguousTarget_RefusesAndListsCandidates(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)

	root, binDir := sovereignScopedRepo(t, nil)
	if out, err := testutil.RunBin(t, root, binDir, nil,
		"authorize", "E-0001", "--to", "ai/other", "--reason", "a second delegation"); err != nil {
		t.Fatalf("opening a second scope: %v\n%s", err, out)
	}

	before := headSHA(t, root)
	out, err := testutil.RunBin(t, root, binDir, nil,
		"authorize", "E-0001", "--end", "--reason", "which one?")
	if err == nil {
		t.Fatalf("bare --end with two candidates succeeded; ending is terminal, so the verb must not "+
			"pick one\n%s", out)
	}
	if after := headSHA(t, root); after != before {
		t.Errorf("HEAD moved %s -> %s; the refusal must precede any commit", before[:8], after[:8])
	}
	if !strings.Contains(out, "--scope") {
		t.Errorf("refusal does not name the flag that resolves it:\n%s", out)
	}
	for _, agent := range []string{"ai/claude", "ai/other"} {
		if !strings.Contains(out, agent) {
			t.Errorf("refusal does not list candidate %q; the operator cannot choose from a list "+
				"that omits an option:\n%s", agent, out)
		}
	}
}

// scopeSHAOf returns the auth SHA of the entity's first scope, as
// `aiwf show` reports it.
func scopeSHAOf(t *testing.T, root, binDir, id string) string {
	t.Helper()
	_, scopes := showScopes(t, root, binDir, id)
	if len(scopes) == 0 {
		t.Fatalf("%s carries no scopes", id)
	}
	return scopes[0].AuthSHA
}
