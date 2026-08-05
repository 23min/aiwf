package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/render"
)

// sovereign_trailer_ack_test.go is M-0292/AC-1: `aiwf acknowledge
// illegal` clears the finding a sovereign trailer on a non-human
// commit leaves behind.
//
// The commits are hand-rolled `git commit --trailer` rather than verb
// output, which is the case the milestone exists for: M-0291 closed
// the verb route, so the only way to reach these findings now is
// history a verb never produced — an import, a pre-guard commit, a
// hand-composed trailer set.
//
// Each test acknowledges one of two identically-shaped commits and
// asserts exactly one finding survives. The unacknowledged sibling is
// what distinguishes an ack keyed on the commit from the rule having
// been switched off, which is the failure this milestone must not
// produce: the epic forbids clearing the finding by weakening it.
//
// Two claims are asserted at two scopes, and both are needed. Per code:
// the named rule clears for the acknowledged commit and still fires for
// its sibling. Per commit: no error-severity finding names the
// acknowledged commit afterwards. A code-scoped assertion alone is
// satisfied while a second rule still blocks the same push, which is
// the difference between a code leaving a list and an operator being
// able to proceed.

// forcedCommit writes a uniquely-named file and commits it carrying a
// sovereign trailer (`aiwf-force` or `aiwf-audit-only`) alongside a
// non-human actor — the shape both rules fire on. Returns the full SHA.
//
// `aiwf-verb: promote` rides along so the commit does not also trip
// provenance-untrailered-entity-commit, and `aiwf-principal` so it does
// not trip the missing-principal coherence rule. The commit still
// raises provenance-no-active-scope — an `ai/` actor with no
// on-behalf-of — which is left in deliberately: the per-commit
// assertion has to face a commit carrying more than one finding, or it
// asserts nothing.
func forcedCommit(t *testing.T, root, name, sovereignKey string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name+".txt"), []byte(name+"\n"), 0o644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
	if err := osExec(t, root, "git", "add", name+".txt"); err != nil {
		t.Fatalf("git add %s: %v", name, err)
	}
	if err := osExec(t, root, "git", "commit", "-q", "-m", "forced "+name,
		"--trailer", "aiwf-verb: promote",
		"--trailer", "aiwf-actor: ai/claude",
		"--trailer", "aiwf-principal: human/test",
		"--trailer", sovereignKey+": true",
	); err != nil {
		t.Fatalf("git commit %s: %v", name, err)
	}
	return headSHA(t, root)
}

// checkFindingsWithCode runs `aiwf check --format json` in-process and
// returns every finding carrying code. Parsing the envelope rather than
// grepping the text output is what lets the caller assert on the count
// and on which commit survived, not merely that the string appears.
func checkFindingsWithCode(t *testing.T, root, code string) []check.Finding {
	t.Helper()
	captured := testutil.CaptureStdout(t, func() {
		_ = cli.Execute([]string{"check", "--root", root, "--format", "json"})
	})
	var env render.Envelope
	if err := json.Unmarshal(captured, &env); err != nil {
		t.Fatalf("parsing check envelope: %v\nraw output:\n%s", err, captured)
	}
	var out []check.Finding
	for i := range env.Findings {
		if env.Findings[i].Code == code {
			out = append(out, env.Findings[i])
		}
	}
	return out
}

// TestAcknowledgeIllegal_ForceNonHuman_ClearsOnlyTheAcknowledgedCommit is
// M-0292/AC-1 for provenance-force-non-human: two forced non-human
// commits both fire; acknowledging one clears exactly that one.
func TestAcknowledgeIllegal_ForceNonHuman_ClearsOnlyTheAcknowledgedCommit(t *testing.T) {
	// Serial by design, per this package's setup_test.go skip-list:
	// checkFindingsWithCode goes through testutil.CaptureStdout, which
	// swaps the process-global os.Stdout.
	assertAckClearsOnlyItsOwnCommit(t, "aiwf-force", check.CodeProvenanceForceNonHuman)
}

// TestAcknowledgeIllegal_AuditOnlyNonHuman_ClearsOnlyTheAcknowledgedCommit
// is the same claim for provenance-audit-only-non-human. The two rules
// are one shape — a sovereign trailer requires a human actor — and an
// operator who meets the second after the first is cleared has the same
// dead end this milestone removes.
func TestAcknowledgeIllegal_AuditOnlyNonHuman_ClearsOnlyTheAcknowledgedCommit(t *testing.T) {
	// Serial by design — see the sibling above.
	assertAckClearsOnlyItsOwnCommit(t, "aiwf-audit-only", check.CodeProvenanceAuditOnlyNonHuman)
}

// assertAckClearsOnlyItsOwnCommit is the shared body of the two tests
// above: stage two commits carrying sovereignKey, confirm both fire
// code, acknowledge the first, and confirm the second is the one still
// reported.
func assertAckClearsOnlyItsOwnCommit(t *testing.T, sovereignKey, code string) {
	t.Helper()
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	acked := forcedCommit(t, root, "acked", sovereignKey)
	untouched := forcedCommit(t, root, "untouched", sovereignKey)

	// Positive control: without an acknowledgment both commits report.
	// Without it a green run cannot tell "the ack worked" from "the
	// fixture never produced the finding in the first place".
	if before := checkFindingsWithCode(t, root, code); len(before) != 2 {
		t.Fatalf("%s before the ack: got %d findings, want 2 (one per forced commit): %+v", code, len(before), before)
	}

	const reason = "imported history predating the verb-time guard"
	mustRun(t, "acknowledge", "illegal", acked, "--reason", reason, "--actor", "human/test", "--root", root)

	after := checkFindingsWithCode(t, root, code)
	if len(after) != 1 {
		t.Fatalf("%s after acknowledging %s: got %d findings, want exactly 1 — "+
			"0 means the acknowledgment silenced the rule rather than the commit, "+
			"2 means it silenced nothing: %+v", code, acked[:7], len(after), after)
	}
	if !strings.Contains(after[0].Message, untouched[:7]) {
		t.Errorf("the surviving finding names %q, want the unacknowledged commit %s", after[0].Message, untouched[:7])
	}
	if strings.Contains(after[0].Message, acked[:7]) {
		t.Errorf("the surviving finding names the acknowledged commit %s: %q", acked[:7], after[0].Message)
	}
	// The rule stays at error severity — the milestone adds a way to
	// clear the finding, never a way to ignore it.
	if after[0].Severity != check.SeverityError {
		t.Errorf("surviving %s severity = %q, want error", code, after[0].Severity)
	}
	// Nothing else still blocks on the acknowledged commit. Asserting
	// only that one code cleared cannot see a second rule firing on the
	// same commit and leaving the push blocked anyway, which is what
	// makes the difference between a code disappearing from a list and
	// an operator being able to proceed.
	assertNoErrorNames(t, root, acked)
}

// assertNoErrorNames fails when any error-severity finding still names
// sha. `aiwf check`'s exit code cannot stand in for this: the
// unacknowledged sibling these tests deliberately leave behind keeps the
// run at exit 1, so the claim has to be scoped to the acknowledged
// commit rather than to the run.
func assertNoErrorNames(t *testing.T, root, sha string) {
	t.Helper()
	captured := testutil.CaptureStdout(t, func() {
		_ = cli.Execute([]string{"check", "--root", root, "--format", "json"})
	})
	var env render.Envelope
	if err := json.Unmarshal(captured, &env); err != nil {
		t.Fatalf("parsing check envelope: %v\nraw output:\n%s", err, captured)
	}
	for i := range env.Findings {
		f := env.Findings[i]
		if f.Severity == check.SeverityError && strings.Contains(f.Message, sha[:7]) {
			t.Errorf("commit %s is acknowledged but still reports a blocking finding %s: %s\n"+
				"an acknowledgment that clears one code while another still blocks the push "+
				"leaves the operator exactly where they started", sha[:7], f.Code, f.Message)
		}
	}
}

// TestAcknowledgeIllegal_UnblocksAFullyDecoratedForcedCommit is the
// claim the per-code tests above cannot make. A forced promote by an
// authorized agent does not carry the bare trailer set those tests
// build: the provenance-decoration layer adds aiwf-principal,
// aiwf-on-behalf-of and aiwf-authorized-by after the verb returns, and
// the resulting commit raises provenance-trailer-incoherent — "aiwf-force:
// and aiwf-on-behalf-of: are mutually exclusive (force is human-only)" —
// alongside provenance-force-non-human.
//
// No verb produces this commit any more; the shape survives in imported
// history and in anything written before the verb-time guard landed,
// which is why the fixture hand-rolls it.
//
// The two say the same thing about the same trailer. An acknowledgment
// that cleared only the first would leave the push blocked by a
// restatement of the rule it had just ratified, which is a ratification
// path in name only. This test is scoped to the operator's actual
// question — can I proceed? — rather than to a code.
func TestAcknowledgeIllegal_UnblocksAFullyDecoratedForcedCommit(t *testing.T) {
	// Serial by design — see the skip-list note above.
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	// A scope opener, so the forced commit's aiwf-authorized-by resolves
	// and the authorization rules judge it as a real delegated act.
	if err := os.WriteFile(filepath.Join(root, "opener.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatalf("writing opener: %v", err)
	}
	if err := osExec(t, root, "git", "add", "opener.txt"); err != nil {
		t.Fatalf("git add opener: %v", err)
	}
	if err := osExec(t, root, "git", "commit", "-q", "-m", "open scope",
		"--trailer", "aiwf-verb: authorize", "--trailer", "aiwf-entity: E-0001",
		"--trailer", "aiwf-actor: human/test", "--trailer", "aiwf-scope: opened",
	); err != nil {
		t.Fatalf("git commit opener: %v", err)
	}
	opener := headSHA(t, root)

	if err := os.WriteFile(filepath.Join(root, "forced.txt"), []byte("y\n"), 0o644); err != nil {
		t.Fatalf("writing forced: %v", err)
	}
	if err := osExec(t, root, "git", "add", "forced.txt"); err != nil {
		t.Fatalf("git add forced: %v", err)
	}
	if err := osExec(t, root, "git", "commit", "-q", "-m", "forced promote by an authorized agent",
		"--trailer", "aiwf-verb: promote", "--trailer", "aiwf-entity: E-0001",
		"--trailer", "aiwf-actor: ai/claude", "--trailer", "aiwf-principal: human/test",
		"--trailer", "aiwf-on-behalf-of: human/test", "--trailer", "aiwf-authorized-by: "+opener,
		"--trailer", "aiwf-force: true",
	); err != nil {
		t.Fatalf("git commit forced: %v", err)
	}
	forced := headSHA(t, root)

	// Positive control: more than one code fires, which is the whole
	// point — a single-code fixture cannot exhibit the defect.
	codesBefore := errorCodesNaming(t, root, forced)
	if len(codesBefore) < 2 {
		t.Fatalf("fixture fired %v; it must raise at least two distinct provenance codes "+
			"or it cannot demonstrate that clearing one is insufficient", codesBefore)
	}

	mustRun(t, "acknowledge", "illegal", forced, "--reason",
		"forced by a delegated agent during the migration; reviewed and accepted",
		"--actor", "human/test", "--root", root)

	if got := errorCodesNaming(t, root, forced); len(got) != 0 {
		t.Errorf("after acknowledging %s, these still block: %v (before: %v)\n"+
			"the acknowledgment must clear the commit, not one of the codes it raises",
			forced[:7], got, codesBefore)
	}
}

// errorCodesNaming returns the sorted, de-duplicated set of
// error-severity finding codes whose message names sha.
func errorCodesNaming(t *testing.T, root, sha string) []string {
	t.Helper()
	captured := testutil.CaptureStdout(t, func() {
		_ = cli.Execute([]string{"check", "--root", root, "--format", "json"})
	})
	var env render.Envelope
	if err := json.Unmarshal(captured, &env); err != nil {
		t.Fatalf("parsing check envelope: %v\nraw output:\n%s", err, captured)
	}
	seen := map[string]bool{}
	var out []string
	for i := range env.Findings {
		f := env.Findings[i]
		if f.Severity == check.SeverityError && strings.Contains(f.Message, sha[:7]) && !seen[f.Code] {
			seen[f.Code] = true
			out = append(out, f.Code)
		}
	}
	sort.Strings(out)
	return out
}

// TestAcknowledgeIllegal_ForceNonHuman_LeavesTheAcknowledgedCommitIntact
// is M-0292/AC-2: ratification records, it does not rewrite. History
// afterwards is exactly what it was plus the acknowledging commit, and
// the reason is readable from that commit.
//
// The assertion is on HEAD's commit list, not on the target commit's
// own bytes. A commit object is immutable and content-addressed, so
// re-reading it after an amend or a rebase returns the same bytes it
// always did — the rewritten copy is a *different* object and the
// original merely becomes unreachable. Comparing the reachable list
// before and after is therefore the claim that can fail: an amend
// replaces the target's SHA in it, a rebase replaces every SHA from
// the target forward, and either shows up as a list that is not the
// old list with one entry appended.
func TestAcknowledgeIllegal_ForceNonHuman_LeavesTheAcknowledgedCommitIntact(t *testing.T) {
	// Serial by design — see the skip-list note above.
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	target := forcedCommit(t, root, "target", "aiwf-force")
	before := revListHead(t, root)

	const reason = "ratified after review; the actor was a bot with a human principal"
	mustRun(t, "acknowledge", "illegal", target, "--reason", reason, "--actor", "human/test", "--root", root)

	after := revListHead(t, root)
	if len(after) != len(before)+1 {
		t.Fatalf("history went from %d commits to %d; ratification must append exactly one and rewrite none\nbefore: %v\nafter:  %v",
			len(before), len(after), before, after)
	}
	for i := range before {
		if after[i] != before[i] {
			t.Fatalf("commit %d of history changed from %s to %s — ratification rewrote history instead of appending to it",
				i, before[i], after[i])
		}
	}
	// The reason is readable from the acknowledging commit — the record
	// is what makes the ratification auditable rather than a silent
	// suppression.
	//
	// Read through the trailer set rather than as a substring of the
	// commit object. The verb writes the reason twice, into the message
	// body and into aiwf-reason, and a substring match is satisfied by
	// either — so it would still pass with the durable carrier gone.
	// The trailer is that carrier: it is what `aiwf history` renders and
	// what a machine consumer parses.
	ackTrailers := commitTrailers(t, root, after[len(after)-1])
	for key, want := range map[string]string{
		gitops.TrailerReason:   reason,
		gitops.TrailerForceFor: target,
		gitops.TrailerActor:    "human/test",
		gitops.TrailerVerb:     "acknowledge-illegal",
	} {
		if got := ackTrailers[key]; got != want {
			t.Errorf("acknowledging commit's %s = %q, want %q", key, got, want)
		}
	}
}

// commitTrailers returns sha's trailer set as a key→value map, read the
// way the kernel's own rules read it.
func commitTrailers(t *testing.T, root, sha string) map[string]string {
	t.Helper()
	raw := gitOutput(t, root, "show", "-s", "--format=%(trailers:only=true,unfold=true)", sha)
	out := map[string]string{}
	for _, tr := range gitops.ParseTrailers(raw) {
		out[tr.Key] = tr.Value
	}
	return out
}

// TestAcknowledgeIllegal_NonHumanActor_IsRefused pins the human-only
// half of AC-2's constraint at the CLI seam: the ratification path
// cannot itself be walked by the actor whose act it would ratify.
func TestAcknowledgeIllegal_NonHumanActor_IsRefused(t *testing.T) {
	// Serial by design — see the skip-list note above.
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	target := forcedCommit(t, root, "target", "aiwf-force")

	rc := cli.Execute([]string{
		"acknowledge", "illegal", target,
		"--reason", "self-ratification attempt", "--actor", "ai/claude", "--root", root,
	})
	if rc == cliutil.ExitOK {
		t.Fatal("acknowledge illegal accepted a non-human actor; the ratification path is human-only")
	}
	if got := checkFindingsWithCode(t, root, check.CodeProvenanceForceNonHuman); len(got) != 1 {
		t.Errorf("after the refused ack: got %d findings, want 1 — a refused ack must clear nothing: %+v", len(got), got)
	}
}

// revListHead returns every commit reachable from HEAD, oldest first.
func revListHead(t *testing.T, root string) []string {
	t.Helper()
	out := strings.TrimSpace(gitOutput(t, root, "rev-list", "--reverse", "HEAD"))
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}
