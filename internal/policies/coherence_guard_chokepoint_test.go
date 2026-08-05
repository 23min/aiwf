package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// goFixture renders a synthetic source file that imports gitops the way
// a real caller does. The import is not decoration: the scan resolves
// the package's local name from it, so a fixture without one describes
// a file that never calls gitops at all.
func goFixture(pkg, body string) string {
	return "package " + pkg + "\n\nimport \"github.com/23min/aiwf/internal/gitops\"\n\n" + body
}

// writeFixtureTree lays out a synthetic module tree for the chokepoint
// policy: apply.go under internal/verb/ plus any extra files given as
// relative path → contents.
func writeFixtureTree(t *testing.T, applyBody string, extra map[string]string) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		filepath.Join("internal", "verb", "apply.go"): goFixture("verb", "func Apply() {\n"+applyBody+"\n}\n"),
	}
	for p, c := range extra {
		files[p] = c
	}
	for rel, contents := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(contents), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return root
}

// TestPolicyCoherenceGuardChokepoint_RealTreeIsClean is M-0291/AC-3
// against the live repository: no production path reaches a verb commit
// without passing the coherence guard.
func TestPolicyCoherenceGuardChokepoint_RealTreeIsClean(t *testing.T) {
	t.Parallel()

	violations, err := PolicyCoherenceGuardChokepoint(repoRoot(t))
	if err != nil {
		t.Fatalf("scanning: %v", err)
	}
	for _, v := range violations {
		t.Errorf("%s:%d: %s", v.File, v.Line, v.Detail)
	}
}

// TestPolicyCoherenceGuardChokepoint_FiresOnASecondCommitSite pins the
// case the policy exists for: a second function that builds a verb
// commit itself, bypassing the seam where the guard runs.
func TestPolicyCoherenceGuardChokepoint_FiresOnASecondCommitSite(t *testing.T) {
	t.Parallel()

	root := writeFixtureTree(t,
		"\tCheckForceTrailerCoherence(p.Trailers)\n\tgitops.CommitVerbChange(ctx)",
		map[string]string{
			filepath.Join("internal", "cli", "shortcut", "shortcut.go"): goFixture("shortcut", "func Run() {\n\tgitops.CommitVerbChange(ctx)\n}\n"),
		})

	violations, err := PolicyCoherenceGuardChokepoint(root)
	if err != nil {
		t.Fatalf("scanning: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("got %d violations, want 1: %+v", len(violations), violations)
	}
	if !strings.Contains(violations[0].File, "shortcut") {
		t.Errorf("violation names %q, want the off-seam commit site", violations[0].File)
	}
}

// TestPolicyCoherenceGuardChokepoint_FiresWhenTheGuardIsGone covers the
// other half. Asserting only "one commit site" would keep passing if
// someone deleted the guard from that site, leaving a chokepoint that
// every commit routes through and that checks nothing.
func TestPolicyCoherenceGuardChokepoint_FiresWhenTheGuardIsGone(t *testing.T) {
	t.Parallel()

	root := writeFixtureTree(t, "\tgitops.CommitVerbChange(ctx)", nil)

	violations, err := PolicyCoherenceGuardChokepoint(root)
	if err != nil {
		t.Fatalf("scanning: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("got %d violations, want 1: %+v", len(violations), violations)
	}
	if !strings.Contains(violations[0].Detail, "CheckForceTrailerCoherence") {
		t.Errorf("detail %q does not name the missing guard", violations[0].Detail)
	}
}

// TestPolicyCoherenceGuardChokepoint_FiresWhenNoCommitSiteExists keeps
// the policy fail-closed. A scan finding no commit site at all reports
// no violations, which to a caller reading only the result is
// indistinguishable from a tree that passed — the silent-pass shape the
// sibling lock policy is also held to.
func TestPolicyCoherenceGuardChokepoint_FiresWhenNoCommitSiteExists(t *testing.T) {
	t.Parallel()

	root := writeFixtureTree(t, "\treturn", nil)

	violations, err := PolicyCoherenceGuardChokepoint(root)
	if err != nil {
		t.Fatalf("scanning: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("got %d violations, want 1: %+v", len(violations), violations)
	}
	if !strings.Contains(violations[0].Detail, "no verb-commit site") {
		t.Errorf("detail %q does not report the orphaned scan", violations[0].Detail)
	}
}

// TestPolicyCoherenceGuardChokepoint_FiresOnAnUnparseableCommitSite
// covers the fail-closed skip. A file holding the commit call but no
// valid syntax is the one file a scanner most needs to report: passing
// over it hides precisely the violation being looked for.
func TestPolicyCoherenceGuardChokepoint_FiresOnAnUnparseableCommitSite(t *testing.T) {
	t.Parallel()

	root := writeFixtureTree(t,
		"\tCheckForceTrailerCoherence(p.Trailers)\n\tgitops.CommitVerbChange(ctx)",
		map[string]string{
			filepath.Join("internal", "cli", "broken", "broken.go"): goFixture("broken", "func Run( {\n\tgitops.CommitVerbChange(ctx)\n"),
		})

	violations, err := PolicyCoherenceGuardChokepoint(root)
	if err != nil {
		t.Fatalf("scanning: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("got %d violations, want 1: %+v", len(violations), violations)
	}
	if !strings.Contains(violations[0].Detail, "does not parse") {
		t.Errorf("detail %q does not report the unparseable file", violations[0].Detail)
	}
}

// TestPolicyCoherenceGuardChokepoint_UnreadableRootErrors pins that an
// unwalkable root surfaces as an error rather than as an empty result.
func TestPolicyCoherenceGuardChokepoint_UnreadableRootErrors(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "no-such-tree")
	if _, err := PolicyCoherenceGuardChokepoint(missing); err == nil {
		t.Fatal("scanning a missing root returned no error; an unreadable tree must not read as clean")
	}
}

// TestPolicyCoherenceGuardChokepoint_FiresOnAnAliasedImport pins the
// bypass a name-based scan cannot see.
//
// An import alias renames the selector, so the call reads as
// `g.CommitVerbChange(...)` and no search for "gitops.Commit…" finds
// it. Resolving the local name from the import block is what closes it.
func TestPolicyCoherenceGuardChokepoint_FiresOnAnAliasedImport(t *testing.T) {
	t.Parallel()

	root := writeFixtureTree(t,
		"\tCheckForceTrailerCoherence(p.Trailers)\n\tgitops.CommitVerbChange(ctx)",
		map[string]string{
			filepath.Join("internal", "cli", "aliased", "aliased.go"): "package aliased\n\n" +
				"import g \"github.com/23min/aiwf/internal/gitops\"\n\n" +
				"func Run() {\n\tg.CommitVerbChange(ctx)\n}\n",
		})

	violations, err := PolicyCoherenceGuardChokepoint(root)
	if err != nil {
		t.Fatalf("scanning: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("got %d violations, want 1: %+v", len(violations), violations)
	}
	if !strings.Contains(violations[0].File, "aliased") {
		t.Errorf("violation names %q, want the aliased off-seam site", violations[0].File)
	}
}

// TestPolicyCoherenceGuardChokepoint_FiresOnALowerLevelPrimitive pins
// the other bypass: reaching past CommitVerbChange to a primitive it is
// built from.
//
// Holding only the one call a verb is supposed to make would leave the
// three that skip it unwatched, which is the shape this policy exists
// to catch rather than an edge of it.
func TestPolicyCoherenceGuardChokepoint_FiresOnALowerLevelPrimitive(t *testing.T) {
	t.Parallel()

	for _, primitive := range []string{"Commit", "CommitAllowEmpty", "CommitTree"} {
		t.Run(primitive, func(t *testing.T) {
			t.Parallel()
			root := writeFixtureTree(t,
				"\tCheckForceTrailerCoherence(p.Trailers)\n\tgitops.CommitVerbChange(ctx)",
				map[string]string{
					filepath.Join("internal", "cli", "lower", "lower.go"): goFixture("lower",
						"func Run() {\n\tgitops."+primitive+"(ctx)\n}\n"),
				})

			violations, err := PolicyCoherenceGuardChokepoint(root)
			if err != nil {
				t.Fatalf("scanning: %v", err)
			}
			if len(violations) != 1 {
				t.Fatalf("got %d violations, want 1 for gitops.%s: %+v", len(violations), primitive, violations)
			}
			if !strings.Contains(violations[0].File, "lower") {
				t.Errorf("violation names %q, want the lower-level commit site", violations[0].File)
			}
		})
	}
}

// TestPolicyCoherenceGuardChokepoint_IgnoresAPrimitiveNamedInAString
// pins the false positive the resolved scan removes.
//
// internal/verb/apply.go names CommitTree in comments inside function
// bodies; a textual scan cannot tell a mentioned name from a call, so
// it would report those functions as off-seam commit sites.
//
// The fixture deliberately does NOT live under internal/policies/,
// which WalkGoFiles skips wholesale so that scanners do not trip the
// policies they implement. A fixture placed there is never read, and
// the test would pass no matter what the scan did.
func TestPolicyCoherenceGuardChokepoint_IgnoresAPrimitiveNamedInAString(t *testing.T) {
	t.Parallel()

	root := writeFixtureTree(t,
		"\tCheckForceTrailerCoherence(p.Trailers)\n\tgitops.CommitVerbChange(ctx)",
		map[string]string{
			filepath.Join("internal", "cli", "quoter", "quoter.go"): goFixture("quoter",
				"func Names() []string {\n\t// gitops.CommitTree is named here, not called.\n"+
					"\treturn []string{\"gitops.CommitVerbChange(\", \"gitops.CommitTree(\"}\n}\n"),
		})

	violations, err := PolicyCoherenceGuardChokepoint(root)
	if err != nil {
		t.Fatalf("scanning: %v", err)
	}
	for _, v := range violations {
		t.Errorf("%s:%d: %s — a quoted primitive name is not a call site", v.File, v.Line, v.Detail)
	}
}
