package integration

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// sovereignForceRepo returns a repo in the only state from which a
// non-human actor can reach a verb's --force path at all: E-0001 and
// M-0001 present, HEAD on a ritual branch, and an active scope
// authorizing ai/claude over the epic.
//
// The scope is load-bearing rather than incidental. Without it the
// allow-rule refuses first with provenance-no-active-scope, so the
// trailer set is never assembled and the coherence guard is never
// consulted — a test built without the scope would pass while proving
// nothing about the guard.
func sovereignForceRepo(t *testing.T) (root, binDir string) {
	t.Helper()
	bin := testutil.AiwfBinary(t)
	binDir = filepath.Dir(bin)
	root = t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q", "-b", "main"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := testutil.RunGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	for _, args := range [][]string{
		{"init"},
		{"add", "epic", "--title", "Platform"},
		{"add", "milestone", "--epic", "E-0001", "--title", "First milestone", "--tdd", "required"},
	} {
		if out, err := testutil.RunBin(t, root, binDir, nil, args...); err != nil {
			t.Fatalf("aiwf %v: %v\n%s", args, err, out)
		}
	}
	if out, err := testutil.RunGit(root, "checkout", "-q", "-b", "epic/E-0001-platform"); err != nil {
		t.Fatalf("git checkout -b: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil,
		"authorize", "E-0001", "--to", "ai/claude", "--reason", "implement E-0001"); err != nil {
		t.Fatalf("aiwf authorize: %v\n%s", err, out)
	}
	return root, binDir
}

// TestSovereignForce_NonHumanActor_RefusedBeforeCommit is M-0291/AC-1.
//
// Every site that constructs a sovereign aiwf-force trailer must refuse a
// non-human actor before writing anything. Three sites construct one:
// the shared transitionTrailers helper in internal/verb/promote.go —
// which serves promote, cancel and the AC-granularity transitions — and
// the inline sites in internal/verb/add.go and internal/verb/authorize.go.
// The cases below drive each through the real binary.
//
// The unmoved-HEAD assertion is the load-bearing half. A guard that
// reports a refusal after the commit has already landed leaves exactly
// the record this milestone exists to prevent, and a test asserting only
// the exit code would not tell the two apart.
//
// The refusal is asserted as "a force coherence rule refused", not as one
// named rule. CheckTrailerCoherence returns its first violation only, and
// a non-human actor that got past the allow-rule necessarily carries an
// active scope's aiwf-on-behalf-of — so force-with-on-behalf-of is
// reached before force-non-human. Pinning either name would assert the
// order the rules happen to sit in rather than the behavior.
func TestSovereignForce_NonHumanActor_RefusedBeforeCommit(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)

	// Each case carries its complete argument list rather than having the
	// actor flags appended uniformly: authorize is absent from the
	// ProvenanceContext roster, so it registers no --principal flag. That
	// is the same property that let it call the coherence guard at the
	// verb layer when the other verbs could not.
	//
	// wantRefusal is the substring proving the refusal came from the guard
	// the case is about. Asserting only a non-zero exit would let an
	// unrelated rule — a projection finding, a child-state check — stand
	// in for the guard and report success for the wrong reason.
	cases := []struct {
		name        string
		site        string
		setup       [][]string
		args        []string
		wantRefusal string
	}{
		{
			name: "promote entity",
			site: "internal/verb/promote.go transitionTrailers",
			args: []string{
				"promote", "E-0001", "active", "--force", "--reason", "escalation",
				"--actor", "ai/claude", "--principal", "human/peter",
			},
			wantRefusal: "aiwf-force",
		},
		{
			// The milestone, not the epic: cancelling the epic is refused
			// by epic-cancel-non-terminal-children before a trailer set
			// is ever assembled, which would make the case pass without
			// reaching the site under test.
			name: "cancel entity",
			site: "internal/verb/promote.go transitionTrailers",
			args: []string{
				"cancel", "M-0001", "--force", "--reason", "escalation",
				"--actor", "ai/claude", "--principal", "human/peter",
			},
			wantRefusal: "aiwf-force",
		},
		{
			// A phase transition, not a status one: promoting an AC to met
			// under tdd: required is refused by acs-tdd-audit as a
			// projection finding, again short of the site under test.
			name:  "promote acceptance criterion phase",
			site:  "internal/verb/promote.go transitionTrailers, via internal/verb/ac.go",
			setup: [][]string{{"add", "ac", "M-0001", "--title", "Observable behavior"}},
			args: []string{
				"promote", "M-0001/AC-1", "--phase", "red", "--force", "--reason", "escalation",
				"--actor", "ai/claude", "--principal", "human/peter",
			},
			wantRefusal: "aiwf-force",
		},
		{
			name: "add born-complete entity",
			site: "internal/verb/add.go",
			args: []string{
				"add", "gap", "--title", "Some gap", "--discovered-in", "M-0001",
				"--force", "--reason", "escalation",
				"--actor", "ai/claude", "--principal", "human/peter",
			},
			wantRefusal: "aiwf-force",
		},
		{
			name: "authorize on a scope-entity",
			site: "internal/verb/authorize.go",
			args: []string{
				"authorize", "M-0001", "--to", "ai/other",
				"--branch", "milestone/M-0001-first-milestone",
				"--force", "--reason", "escalation",
				"--actor", "ai/claude",
			},
			// authorize is already guarded, by its own human-actor check
			// rather than by coherence. Pinned here so the site stays
			// refused however the guard below is wired.
			wantRefusal: "only humans authorize",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root, binDir := sovereignForceRepo(t)
			for _, args := range tc.setup {
				if out, err := testutil.RunBin(t, root, binDir, nil, args...); err != nil {
					t.Fatalf("setup aiwf %v: %v\n%s", args, err, out)
				}
			}

			before := headSHA(t, root)
			out, err := testutil.RunBin(t, root, binDir, nil, tc.args...)

			if err == nil {
				t.Errorf("%s: verb succeeded; want refusal (site: %s)\n%s", tc.name, tc.site, out)
			}
			if after := headSHA(t, root); after != before {
				t.Errorf("%s: HEAD moved %s -> %s; the act committed before the guard refused (site: %s)",
					tc.name, before[:8], after[:8], tc.site)
			}
			if !strings.Contains(out, tc.wantRefusal) {
				t.Errorf("%s: refusal does not contain %q, so it came from some other rule (site: %s)\n%s",
					tc.name, tc.wantRefusal, tc.site, out)
			}
		})
	}
}
