package integration

import (
	"os"
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
// The refusal message is pinned, not just the fact of a refusal. Only
// the first violation is reported, and a non-human actor that got past
// the allow-rule necessarily carries an active scope's
// aiwf-on-behalf-of — so the rule order decides which sentence the
// operator reads. It is ordered to name force itself, rather than the
// trailer pair that merely co-occurs with it, and pinning the sentence
// is what keeps a future reorder from quietly degrading it into two
// trailer keys nobody typed.
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
	// wantExit is asserted because "non-zero" does not distinguish a
	// refusal from a crash. A coherence refusal is a legality refusal, so
	// it takes ExitFindings (1) — the exit `aiwf check` reports for the
	// same violation class once the act has landed. Reporting it as
	// ExitInternal (3) would tell an operator, and any pipeline reading
	// the code, that aiwf itself broke.
	cases := []struct {
		name        string
		site        string
		setup       [][]string
		args        []string
		wantRefusal string
		wantExit    int
	}{
		{
			name: "promote entity",
			site: "internal/verb/promote.go transitionTrailers",
			args: []string{
				"promote", "E-0001", "active", "--force", "--reason", "escalation",
				"--actor", "ai/claude", "--principal", "human/peter",
			},
			wantRefusal: "only humans wield --force",
			wantExit:    1,
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
			wantRefusal: "only humans wield --force",
			wantExit:    1,
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
			wantRefusal: "only humans wield --force",
			wantExit:    1,
		},
		{
			name: "add born-complete entity",
			site: "internal/verb/add.go",
			args: []string{
				"add", "gap", "--title", "Some gap", "--discovered-in", "M-0001",
				"--force", "--reason", "escalation",
				"--actor", "ai/claude", "--principal", "human/peter",
			},
			wantRefusal: "only humans wield --force",
			wantExit:    1,
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
			wantExit:    2,
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
			} else if !testutil.ExitedWithCode(err, tc.wantExit) {
				t.Errorf("%s: exited %v, want %d; a refusal reported as an internal error tells the operator "+
					"aiwf broke, and leaves a pipeline unable to tell a denial from a crash (site: %s)\n%s",
					tc.name, err, tc.wantExit, tc.site, out)
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

// TestForceReplaceVerbs_NonHumanActor_StillWork is the other half of
// M-0291/AC-1, and it guards the boundary the force refusal must not
// cross.
//
// contract bind, contract unbind and both contract recipe verbs declare
// a --force that means force-replace: overwrite the binding already
// there. It is a different word spelled the same, it emits no sovereign
// trailer, and sweeping these verbs into the sovereign refusal would
// break legitimate automation.
//
// The steps run in sequence against one repo because they are stateful
// — a binding cannot be replaced before a validator is installed — and
// because that sequence is how an agent actually drives them. Each step
// asserts a landed commit rather than only a zero exit: a verb that
// returns success without committing has been closed just as
// effectively, only more quietly.
//
// None of these verbs passes through the provenance-decoration layer,
// so none carries an aiwf-principal and none registers a flag that
// could supply one. That is what makes this the load-bearing case: a
// seam enforcing any rule beyond the force-predicated ones refuses all
// four with no invocation that could satisfy it.
func TestForceReplaceVerbs_NonHumanActor_StillWork(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)

	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)
	root := t.TempDir()
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
	bodyPath := filepath.Join(root, "contract-body.md")
	if err := os.WriteFile(bodyPath,
		[]byte("## Purpose\n\nPin the rendered shape.\n\n## Stability\n\nStable.\n"), 0o644); err != nil {
		t.Fatalf("writing contract body: %v", err)
	}
	for _, dir := range []string{"render", "render2"} {
		if err := os.MkdirAll(filepath.Join(root, "fixtures", dir), 0o755); err != nil {
			t.Fatalf("mkdir fixtures/%s: %v", dir, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "schemas"), 0o755); err != nil {
		t.Fatalf("mkdir schemas: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "schemas", "render.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("writing schema: %v", err)
	}
	for _, args := range [][]string{
		{"init"},
		{"add", "contract", "--title", "Render contract", "--body-file", bodyPath},
	} {
		if out, err := testutil.RunBin(t, root, binDir, nil, args...); err != nil {
			t.Fatalf("setup aiwf %v: %v\n%s", args, err, out)
		}
	}

	steps := []struct {
		name string
		args []string
	}{
		{"recipe install", []string{"contract", "recipe", "install", "jsonschema"}},
		{"bind", []string{
			"contract", "bind", "C-0001", "--validator", "jsonschema",
			"--schema", "schemas/render.json", "--fixtures", "fixtures/render",
		}},
		// --force here is the force-replace sense: overwrite the binding
		// just made. It is the flag the constraint is about, so a step
		// has to actually pass it — otherwise the test would stay green
		// if contract bind began emitting a sovereign aiwf-force trailer.
		{"bind --force (replace)", []string{
			"contract", "bind", "C-0001", "--validator", "jsonschema",
			"--schema", "schemas/render.json", "--fixtures", "fixtures/render2",
			"--force",
		}},
		{"unbind", []string{"contract", "unbind", "C-0001"}},
		{"recipe remove", []string{"contract", "recipe", "remove", "jsonschema"}},
	}
	for _, step := range steps {
		before := headSHA(t, root)
		args := append(append([]string{}, step.args...), "--actor", "ai/claude")
		out, err := testutil.RunBin(t, root, binDir, nil, args...)
		if err != nil {
			t.Fatalf("%s: refused a force-replace verb for a non-human actor: %v\n%s", step.name, err, out)
		}
		if after := headSHA(t, root); after == before {
			t.Fatalf("%s: exited zero but committed nothing; the verb is closed, not working", step.name)
		}
	}
}

// TestAuditOnly_NonHumanActor_ExitsAsALegalityRefusal pins the exit
// class of the coherence refusals raised inside a verb, not at the
// commit seam.
//
// Making *verb.CoherenceError carry a finding code moved these from the
// usage exit to the findings exit. That is the documented contract for
// a Coded error — a legality refusal takes the same exit as the
// check-time finding for its violation class — but it is a CLI-contract
// change on a path this milestone did not otherwise touch, and nothing
// pinned the old value or the new one.
//
// The refusal itself is correct policy: audit-only is sovereign, so a
// non-human actor cannot wield it. The message is not, and is tracked
// rather than fixed here — the principal it names was supplied, because
// the verb consults the rule set before the CLI layer decorates the
// plan.
func TestAuditOnly_NonHumanActor_ExitsAsALegalityRefusal(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)

	root, binDir := sovereignForceRepo(t)
	before := headSHA(t, root)
	// The epic's current status: audit-only records what is already true,
	// so any other target is refused by the FSM before the trailer set is
	// consulted, and the case would pass without reaching the rule.
	out, err := testutil.RunBin(t, root, binDir, nil,
		"promote", "E-0001", "proposed", "--audit-only", "--reason", "backfill",
		"--actor", "ai/claude", "--principal", "human/peter")

	if err == nil {
		t.Fatalf("audit-only by a non-human actor succeeded; want refusal\n%s", out)
	}
	if !testutil.ExitedWithCode(err, 1) {
		t.Errorf("exited %v, want 1; a coherence refusal is a legality refusal wherever it is raised, "+
			"and reporting one exit at the verb and another at the seam splits one violation class\n%s", err, out)
	}
	if after := headSHA(t, root); after != before {
		t.Errorf("HEAD moved %s -> %s; the refusal must precede the commit", before[:8], after[:8])
	}
}

// TestSovereignActRefusal_DoesNotAdviseAForceThatIsRefused pins the
// remedy a sovereign-act refusal offers.
//
// The message is reachable only for a non-human actor — a human/ actor
// returns before it — and this milestone made --force by a non-human
// actor refuse. Advising --force there would send the operator at a
// gate that rejects them, which is CLAUDE.md's self-explaining-error
// rule inverted: it says what to do next, and that thing now fails.
func TestSovereignActRefusal_DoesNotAdviseAForceThatIsRefused(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)

	root, binDir := sovereignForceRepo(t)
	// A sovereign-act-shape transition without --force: refused by the
	// sovereign-act gate, which is the message under test.
	out, err := testutil.RunBin(t, root, binDir, nil,
		"promote", "E-0001", "active", "--actor", "ai/claude", "--principal", "human/peter")
	if err == nil {
		t.Fatalf("sovereign act by a non-human actor succeeded; want refusal\n%s", out)
	}
	if !strings.Contains(out, "sovereign act requires a human/ actor") {
		t.Fatalf("refusal came from some other guard:\n%s", out)
	}
	if strings.Contains(out, "--force") {
		t.Errorf("the refusal advises --force, which this actor cannot use — "+
			"following it exits 1 at the coherence guard:\n%s", out)
	}
}
