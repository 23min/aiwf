package check

import (
	"bytes"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
)

// commit_msg_ac_scope_test.go pins the rule tying a commit's own claim to the
// record that makes it findable: a subject naming an `(M-NNNN/AC-N)` scope
// asserts this commit implemented that criterion, and only the matching
// `aiwf-entity` trailer makes the assertion reachable from the criterion.
//
// The rule is what an AC's history has instead of a milestone's `## Work log`.
// It binds by convention rather than by repo: a subject that names no AC never
// meets it, so a project not following the scoped-subject convention is
// unaffected, and an AC met by an observation rather than by code owes nothing.
func TestRunCommitMsg_SubjectNamingAnACRequiresTheMatchingEntityTrailer(t *testing.T) {
	t.Parallel()

	verbs := map[string]struct{}{"promote": {}, "add": {}}

	cases := []struct {
		name string
		msg  string
		want int
	}{
		{
			"subject names an AC and the trailer matches",
			"feat(history): admit by the parsed trailer (M-0327/AC-1)\n\naiwf-entity: M-0327/AC-1\n",
			cliutil.ExitOK,
		},
		{
			// The shape every AC commit in this repo has had: the subject
			// claims the AC, the trailer names only the milestone, and the
			// AC's own history stays empty of the work.
			"trailer names the bare milestone instead",
			"feat(history): admit by the parsed trailer (M-0327/AC-1)\n\naiwf-entity: M-0327\n",
			cliutil.ExitFindings,
		},
		{
			"no trailer at all",
			"feat(history): admit by the parsed trailer (M-0327/AC-1)\n",
			cliutil.ExitFindings,
		},
		{
			// A trailer block that exists but carries no aiwf-entity: the
			// lookup loop runs and finds nothing, which is a different path
			// from the empty-block case above.
			"a trailer block with no entity key",
			"feat(history): admit by the parsed trailer (M-0327/AC-1)\n\naiwf-verb: promote\naiwf-actor: human/peter\n",
			cliutil.ExitFindings,
		},
		{
			"trailer names a different AC",
			"feat(history): admit by the parsed trailer (M-0327/AC-1)\n\naiwf-entity: M-0327/AC-2\n",
			cliutil.ExitFindings,
		},
		{
			// Narrower legacy widths name the same AC, so the comparison
			// canonicalizes both sides rather than matching spelling. Three
			// digits is the milestone grammar's floor (`^M-\\d{3,}$`).
			"narrow subject id against a canonical trailer",
			"feat(x): a thing (M-007/AC-1)\n\naiwf-entity: M-0007/AC-1\n",
			cliutil.ExitOK,
		},
		{
			"canonical subject id against a narrow trailer",
			"feat(x): a thing (M-0007/AC-1)\n\naiwf-entity: M-007/AC-1\n",
			cliutil.ExitOK,
		},
		{
			// No claim, no obligation — this is what keeps the rule silent
			// in a project that does not scope subjects by AC.
			"subject names no AC",
			"chore: tidy the thing\n\naiwf-entity: M-0327\n",
			cliutil.ExitOK,
		},
		{
			// Anchored at end of line: a scope mentioned mid-sentence is
			// discussing a criterion, not claiming to implement it.
			"scope appears mid-subject rather than as the trailing scope",
			"docs: explain (M-0327/AC-1) in the guide\n",
			cliutil.ExitOK,
		},
		{
			// A milestone-scoped subject is not an AC claim.
			"subject names a milestone, not an AC",
			"feat(x): a thing (M-0327)\n\naiwf-entity: M-0327\n",
			cliutil.ExitOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			code := runCommitMsg(writeMsg(t, tc.msg), verbs, &buf)
			if code != tc.want {
				t.Errorf("code = %d, want %d; stderr = %q", code, tc.want, buf.String())
			}
			if tc.want != cliutil.ExitFindings {
				return
			}
			// The refusal names the claim and what is missing, or an
			// operator cannot act on it.
			if !strings.Contains(buf.String(), "AC-1") {
				t.Errorf("refusal does not name the AC the subject claims; stderr = %q", buf.String())
			}
		})
	}
}
