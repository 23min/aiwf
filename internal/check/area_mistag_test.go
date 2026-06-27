package check

import (
	"context"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestGatherEntityPaths pins M-0181/AC-1: the gather walks HEAD-reachable
// history and unions, per canonical root entity id, every path the entity's
// `aiwf-entity:`-trailered commits touched. The gather is unfiltered (planning
// files and project code alike) and rolls composite AC trailers up to the
// parent milestone; untrailered commits contribute no key.
func TestGatherEntityPaths(t *testing.T) {
	t.Parallel()
	f := newWalkerFixture(t)

	// Two commits for G-0301: app-a code across two files plus a shared
	// README. The README is intentionally outside any project area — gather
	// must still union it (filtering is AreaMistag's job, not gather's).
	f.writeFile("projects/app-a/login.go", "package app\n")
	f.commit("fix login", "aiwf-entity: G-0301", "aiwf-actor: human/test")
	f.writeFile("projects/app-a/auth.go", "package app\n")
	f.writeFile("README.md", "docs\n")
	f.commit("more auth", "aiwf-entity: G-0301")

	// A different entity, G-0302, touching billing.
	f.writeFile("projects/billing/invoice.go", "package billing\n")
	f.commit("billing work", "aiwf-entity: G-0302")

	// A composite AC trailer rolls up to its milestone M-0500.
	f.writeFile("projects/platform/lib.go", "package platform\n")
	f.commit("platform lib", "aiwf-entity: M-0500/AC-2")

	// A commit with NO aiwf-entity trailer contributes no entity key.
	f.writeFile("untracked/whatever.txt", "x\n")
	f.commit("untrailered tidy")

	got := GatherEntityPaths(context.Background(), f.root)

	wantG0301 := map[string]bool{
		"projects/app-a/login.go": true,
		"projects/app-a/auth.go":  true,
		"README.md":               true,
	}
	if diff := cmp.Diff(wantG0301, got["G-0301"]); diff != "" {
		t.Errorf("G-0301 paths mismatch (-want +got):\n%s", diff)
	}
	if !got["G-0302"]["projects/billing/invoice.go"] {
		t.Errorf("G-0302 should include projects/billing/invoice.go; got %v", got["G-0302"])
	}
	if !got["M-0500"]["projects/platform/lib.go"] {
		t.Errorf("M-0500 (rolled up from AC-2 trailer) should include projects/platform/lib.go; got %v", got["M-0500"])
	}
	if keys := sortedKeys(got); len(keys) != 3 {
		t.Errorf("expected exactly 3 entity keys (G-0301, G-0302, M-0500); got %v", keys)
	}
}

// TestGatherEntityPaths_Inert pins the early-return arms (M-0181/AC-1): an
// empty root, a non-git directory, and a git repo whose only history is an
// untrailered seed commit all yield nil — no entity keys to attribute.
func TestGatherEntityPaths_Inert(t *testing.T) {
	t.Parallel()
	if got := GatherEntityPaths(context.Background(), ""); got != nil {
		t.Errorf("empty root: want nil, got %v", got)
	}
	if got := GatherEntityPaths(context.Background(), t.TempDir()); got != nil {
		t.Errorf("non-git dir: want nil, got %v", got)
	}
	f := newWalkerFixture(t) // seeds one untrailered empty commit
	if got := GatherEntityPaths(context.Background(), f.root); got != nil {
		t.Errorf("untrailered-only history: want nil, got %v", got)
	}
}

// TestGatherEntityPaths_EmptyTrailerValueIgnored pins that a malformed
// empty-valued aiwf-entity trailer (git emits `aiwf-entity:` verbatim when the
// value is blank) is skipped — it must not create a bogus "" entity key.
func TestGatherEntityPaths_EmptyTrailerValueIgnored(t *testing.T) {
	t.Parallel()
	f := newWalkerFixture(t)
	f.writeFile("projects/app-a/x.go", "package app\n")
	f.commit("edge", "aiwf-entity:", "aiwf-entity: G-0303") // one blank, one valid
	got := GatherEntityPaths(context.Background(), f.root)
	if _, ok := got[""]; ok {
		t.Errorf("empty-valued trailer created a bogus \"\" key: %v", got)
	}
	if !got["G-0303"]["projects/app-a/x.go"] {
		t.Errorf("valid trailer G-0303 should include projects/app-a/x.go; got %v", got["G-0303"])
	}
}

func sortedKeys(m map[string]map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
