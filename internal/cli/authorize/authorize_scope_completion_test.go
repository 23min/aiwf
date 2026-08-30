package authorize_test

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/authorize"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/entityview"
)

// TestCompleteScopeFlag_SuggestsNonEndedScopesOfTheNamedEntity is
// M-0325/AC-3's completion half.
//
// The registration is already policed — the drift test fails CI on a
// value-taking flag with no completion function — but registration says
// nothing about what the function returns. What makes --scope usable is
// that it offers the entity's own candidates, at the same seven
// characters the operator would otherwise read out of `aiwf show`.
//
// Ended scopes are excluded deliberately: naming one converges rather
// than doing anything, so suggesting it would offer a value whose only
// outcome is "nothing to change".
func TestCompleteScopeFlag_SuggestsNonEndedScopesOfTheNamedEntity(t *testing.T) {
	t.Parallel()
	root := mustNewGitRepo(t)
	mustGit(t, root, "commit", "--allow-empty", "-m", "init")

	// Two scopes on E-0001 and one on E-0002, so the assertion below
	// distinguishes "the entity's scopes" from "every scope in the repo".
	live := commitScope(t, root, "E-0001", "ai/claude")
	ended := commitScope(t, root, "E-0001", "ai/other")
	elsewhere := commitScope(t, root, "E-0002", "ai/third")
	mustGit(t, root, "commit", "--allow-empty", "-m", "aiwf promote E-0001 done",
		"--trailer", "aiwf-verb: promote", "--trailer", "aiwf-entity: E-0001",
		"--trailer", "aiwf-actor: human/test", "--trailer", "aiwf-scope-ends: "+ended)

	complete := authorize.CompleteScopeFlagForTest(&root)
	got, _ := complete(nil, []string{"E-0001"}, "")

	if len(got) != 1 {
		t.Fatalf("completion offered %d candidates, want 1 (the entity's one non-ended scope): %v", len(got), got)
	}
	sha, desc, _ := strings.Cut(got[0], "\t")
	// Derived from ShortHash rather than spelled as a second literal:
	// aiwf show, the refusal listing and the commit subject all render
	// through that call, so an assertion against a hand-written prefix
	// would keep passing if the shared abbreviation moved and completion
	// alone stayed behind — the two-literals-that-agree failure this
	// milestone set out to avoid.
	if want := entityview.ShortHash(live); sha != want {
		t.Errorf("suggested %q, want %q — completion must offer the same abbreviation every other "+
			"surface renders", sha, want)
	}
	if !strings.Contains(desc, "ai/claude") {
		t.Errorf("description %q does not name the agent, which is what tells two candidates apart", desc)
	}
	for _, absent := range []string{entityview.ShortHash(ended), entityview.ShortHash(elsewhere)} {
		if strings.Contains(got[0], absent) {
			t.Errorf("completion offered %q; an ended scope converges and a scope on another entity "+
				"is not a candidate at all", absent)
		}
	}
}

// TestCompleteScopeFlag_NonGitDir_OffersNothing pins the best-effort
// contract completion functions hold in this package: a directory with
// nothing to enumerate yields an empty list so the shell falls through
// to its own default, rather than printing something into the
// operator's prompt mid-keystroke.
//
// It reaches that through LoadEntityScopes' no-commits short-circuit,
// which is the path a real repo-less directory takes. The loader's
// error return is a different branch and is annotated at its call site
// as unreachable from a deterministic fixture.
func TestCompleteScopeFlag_NonGitDir_OffersNothing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	complete := authorize.CompleteScopeFlagForTest(&dir)
	if got, _ := complete(nil, []string{"E-0001"}, ""); got != nil {
		t.Errorf("completion in a non-git directory returned %v, want nil", got)
	}
}

// TestCompleteScopeFlag_NoEntityArgument_OffersNothing pins the
// before-the-id case: completion fires while the operator is still
// typing, so it must tolerate an empty argument list rather than
// indexing into it.
func TestCompleteScopeFlag_NoEntityArgument_OffersNothing(t *testing.T) {
	t.Parallel()
	root := mustNewGitRepo(t)
	complete := authorize.CompleteScopeFlagForTest(&root)
	if got, _ := complete(nil, nil, ""); got != nil {
		t.Errorf("completion with no entity argument returned %v, want nil", got)
	}
}

// commitScope writes an authorize-opener commit for entity/agent and
// returns its SHA — the value that becomes the scope's AuthSHA on
// replay.
func commitScope(t *testing.T, root, entityID, agent string) string {
	t.Helper()
	mustGit(t, root, "commit", "--allow-empty", "-m", "aiwf authorize "+entityID+" --to "+agent,
		"--trailer", "aiwf-verb: authorize",
		"--trailer", "aiwf-entity: "+entityID,
		"--trailer", "aiwf-actor: human/test",
		"--trailer", "aiwf-to: "+agent,
		"--trailer", "aiwf-scope: opened")
	out, err := testutil.RunGit(root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v\n%s", err, out)
	}
	return strings.TrimSpace(out)
}
