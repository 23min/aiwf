package check

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/entity"
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

// TestAreaMistag_FiresOnForeignAreaWork pins M-0181/AC-2: an entity tagged to
// one area whose linked commits touched a DIFFERENT area's path territory fires
// exactly one area-mistag warning naming the entity, its area, and the foreign
// area. The entity's own planning file (matching no area glob) is ignored.
// (AC-2 is the crude predicate — fire on any foreign-area work; AC-3 refines it
// to tolerate cross-cutting.)
func TestAreaMistag_FiresOnForeignAreaWork(t *testing.T) {
	t.Parallel()
	tr := makeTree(
		&entity.Entity{ID: "G-0001", Kind: entity.KindGap, Path: "work/gaps/G-0001-x.md", Area: "app-a"},
	)
	areas := []AreaPaths{
		{Name: "app-a", Paths: []string{"projects/app-a/**"}},
		{Name: "billing", Paths: []string{"projects/billing/**"}},
	}
	touched := map[string]map[string]bool{
		"G-0001": {
			"projects/billing/invoice.go": true, // foreign-area work
			"work/gaps/G-0001-x.md":       true, // planning file: matches no area glob
		},
	}
	got := AreaMistag(tr, areas, touched)
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 area-mistag finding, got %d: %+v", len(got), got)
	}
	f := got[0]
	if f.Code != CodeAreaMistag {
		t.Errorf("Code = %q, want %q", f.Code, CodeAreaMistag)
	}
	if f.Severity != SeverityWarning {
		t.Errorf("Severity = %q, want warning", f.Severity)
	}
	if f.EntityID != "G-0001" {
		t.Errorf("EntityID = %q, want G-0001", f.EntityID)
	}
	if f.Field != "area" {
		t.Errorf("Field = %q, want area", f.Field)
	}
	for _, want := range []string{"G-0001", "app-a", "billing"} {
		if !strings.Contains(f.Message, want) {
			t.Errorf("Message %q does not contain %q", f.Message, want)
		}
	}
}

// TestAreaMistag_NoFinding pins every no-fire guard of AreaMistag (M-0181/AC-2),
// one case per branch. These are mandatory unit cases: the CLI seam gates the
// gather behind AnyAreaHasPaths, so the guards no longer get incidental coverage
// from unrelated `aiwf check` runs — only these dedicated cases exercise them.
// (AC-4 adds the end-to-end inert guarantees through `aiwf check`; this is the
// unit-level complement.)
func TestAreaMistag_NoFinding(t *testing.T) {
	t.Parallel()
	withPaths := []AreaPaths{
		{Name: "app-a", Paths: []string{"projects/app-a/**"}},
		{Name: "billing", Paths: []string{"projects/billing/**"}},
	}
	cases := []struct {
		name    string
		entity  *entity.Entity
		areas   []AreaPaths
		touched map[string]map[string]bool
	}{
		{
			name:    "no area declares paths",
			entity:  &entity.Entity{ID: "G-0001", Kind: entity.KindGap, Path: "work/gaps/G-0001-x.md", Area: "app-a"},
			areas:   []AreaPaths{{Name: "app-a"}, {Name: "billing"}}, // label-only, no paths
			touched: map[string]map[string]bool{"G-0001": {"projects/billing/x.go": true}},
		},
		{
			name:    "archived entity is out of scope",
			entity:  &entity.Entity{ID: "G-0002", Kind: entity.KindGap, Path: "work/gaps/archive/G-0002-x.md", Area: "app-a"},
			areas:   withPaths,
			touched: map[string]map[string]bool{"G-0002": {"projects/billing/x.go": true}},
		},
		{
			name:    "untagged entity (empty area)",
			entity:  &entity.Entity{ID: "G-0003", Kind: entity.KindGap, Path: "work/gaps/G-0003-x.md", Area: ""},
			areas:   withPaths,
			touched: map[string]map[string]bool{"G-0003": {"projects/billing/x.go": true}},
		},
		{
			name:    "global sentinel is inherently cross-cutting",
			entity:  &entity.Entity{ID: "ADR-0001", Kind: entity.KindADR, Path: "docs/adr/ADR-0001-x.md", Area: entity.AreaGlobal},
			areas:   withPaths,
			touched: map[string]map[string]bool{"ADR-0001": {"projects/billing/x.go": true}},
		},
		{
			name:    "entity's own area declares no paths",
			entity:  &entity.Entity{ID: "G-0004", Kind: entity.KindGap, Path: "work/gaps/G-0004-x.md", Area: "app-a"},
			areas:   []AreaPaths{{Name: "app-a"}, {Name: "billing", Paths: []string{"projects/billing/**"}}},
			touched: map[string]map[string]bool{"G-0004": {"projects/billing/x.go": true}},
		},
		{
			name:    "entity has no linked commits",
			entity:  &entity.Entity{ID: "G-0005", Kind: entity.KindGap, Path: "work/gaps/G-0005-x.md", Area: "app-a"},
			areas:   withPaths,
			touched: map[string]map[string]bool{}, // nothing gathered for G-0005
		},
		{
			name:    "work landed only in the entity's own area",
			entity:  &entity.Entity{ID: "G-0006", Kind: entity.KindGap, Path: "work/gaps/G-0006-x.md", Area: "app-a"},
			areas:   withPaths,
			touched: map[string]map[string]bool{"G-0006": {"projects/app-a/x.go": true, "work/gaps/G-0006-x.md": true}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := AreaMistag(makeTree(tc.entity), tc.areas, tc.touched)
			if len(got) != 0 {
				t.Errorf("expected no findings, got %d: %+v", len(got), got)
			}
		})
	}
}

// TestAnyAreaHasPaths pins the CLI-seam gate (M-0181/AC-2 / blocker 2): true iff
// at least one declared area carries a `paths:` glob, so the seam can skip the
// full-history gather when mistag is inert.
func TestAnyAreaHasPaths(t *testing.T) {
	t.Parallel()
	if AnyAreaHasPaths(nil) {
		t.Error("nil areas: want false")
	}
	if AnyAreaHasPaths([]AreaPaths{{Name: "app-a"}, {Name: "billing"}}) {
		t.Error("label-only areas: want false")
	}
	if !AnyAreaHasPaths([]AreaPaths{{Name: "app-a"}, {Name: "billing", Paths: []string{"projects/billing/**"}}}) {
		t.Error("one paths-carrying area: want true")
	}
}

// TestGatherEntityPaths_NarrowIdCanonicalized pins the gather seam's canonical-
// width promise (M-0181/AC-1, flagged in review): a pre-ADR-0008 narrow-width
// trailer (`aiwf-entity: G-123`, the legacy 3-digit gap width) is keyed at
// canonical width (`G-0123`), so a canonical-width lookup from the check — whose
// tree was rewidth'd — still finds it.
func TestGatherEntityPaths_NarrowIdCanonicalized(t *testing.T) {
	t.Parallel()
	f := newWalkerFixture(t)
	f.writeFile("projects/app-a/x.go", "package app\n")
	f.commit("narrow", "aiwf-entity: G-123")
	got := GatherEntityPaths(context.Background(), f.root)
	if !got["G-0123"]["projects/app-a/x.go"] {
		t.Errorf("narrow trailer G-123 should be keyed canonical G-0123; got keys %v", sortedKeys(got))
	}
}

// TestMatchesAnyGlob pins the matching helper (M-0181/AC-2): a path matching
// one of the globs returns true; a malformed glob is treated as no-match (the
// AreaDeadGlob precedent — malformed globs are rejected at config load, so an
// error here is indeterminate and skipped).
func TestMatchesAnyGlob(t *testing.T) {
	t.Parallel()
	if !matchesAnyGlob("projects/app-a/x.go", []string{"other/**", "projects/app-a/**"}) {
		t.Error("expected a match against projects/app-a/**")
	}
	if matchesAnyGlob("projects/app-a/x.go", []string{"projects/billing/**"}) {
		t.Error("expected no match against a non-covering glob")
	}
	// A malformed glob (unterminated character class) errors in areamatch.Match
	// and is treated as no-match, exercising the defensive err branch.
	if matchesAnyGlob("anything", []string{"projects/[app"}) {
		t.Error("malformed glob must be treated as no-match, not a match")
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
