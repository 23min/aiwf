package tree

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/entity"
)

// writeFile is a small helper for building synthetic trees in tmpdirs.
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func TestLoad_EmptyRepo(t *testing.T) {
	root := t.TempDir()
	tr, loadErrs, err := Load(context.Background(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loadErrs) != 0 {
		t.Errorf("loadErrs = %v, want empty", loadErrs)
	}
	if len(tr.Entities) != 0 {
		t.Errorf("Entities = %v, want empty", tr.Entities)
	}
}

func TestLoad_AllSixKinds(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "work/epics/E-01-platform/epic.md", `---
id: E-01
title: Platform
status: active
---
`)
	writeFile(t, root, "work/epics/E-01-platform/M-001-cache.md", `---
id: M-001
title: Cache warmup
status: in_progress
parent: E-01
---
`)
	writeFile(t, root, "work/gaps/G-001-noise.md", `---
id: G-001
title: Noise floor
status: open
---
`)
	writeFile(t, root, "work/decisions/D-001-format.md", `---
id: D-001
title: Use OpenAPI
status: accepted
---
`)
	writeFile(t, root, "work/contracts/C-001-orders/contract.md", `---
id: C-001
title: Orders API
status: accepted
linked_adrs:
  - ADR-0001
---
`)
	writeFile(t, root, "docs/adr/ADR-0001-format.md", `---
id: ADR-0001
title: Adopt OpenAPI 3.1
status: accepted
---
`)

	tr, loadErrs, err := Load(context.Background(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loadErrs) != 0 {
		t.Fatalf("loadErrs = %v, want empty", loadErrs)
	}
	if len(tr.Entities) != 6 {
		t.Fatalf("Entities count = %d, want 6: %+v", len(tr.Entities), tr.Entities)
	}

	gotKinds := make(map[entity.Kind]string)
	for _, e := range tr.Entities {
		gotKinds[e.Kind] = e.ID
	}
	want := map[entity.Kind]string{
		entity.KindEpic:      "E-01",
		entity.KindMilestone: "M-001",
		entity.KindGap:       "G-001",
		entity.KindDecision:  "D-001",
		entity.KindContract:  "C-001",
		entity.KindADR:       "ADR-0001",
	}
	for k, wantID := range want {
		if got := gotKinds[k]; got != wantID {
			t.Errorf("kind %s: got %q, want %q", k, got, wantID)
		}
	}
}

func TestLoad_ParseErrorBecomesLoadError(t *testing.T) {
	root := t.TempDir()
	// Valid sibling so we know the loader keeps going.
	writeFile(t, root, "work/epics/E-01-good/epic.md", `---
id: E-01
title: Good epic
status: active
---
`)
	// Malformed YAML.
	writeFile(t, root, "work/epics/E-02-bad/epic.md", `---
id: E-02
title: "Unclosed quote
status: active
---
`)

	tr, loadErrs, err := Load(context.Background(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loadErrs) != 1 {
		t.Fatalf("loadErrs count = %d, want 1: %+v", len(loadErrs), loadErrs)
	}
	if len(tr.Entities) != 1 {
		t.Errorf("Entities count = %d, want 1 (the good one)", len(tr.Entities))
	}
	if loadErrs[0].Path != filepath.FromSlash("work/epics/E-02-bad/epic.md") {
		t.Errorf("loadErrs[0].Path = %q, want work/epics/E-02-bad/epic.md", loadErrs[0].Path)
	}
}

func TestLoad_ParseErrorRegistersStub(t *testing.T) {
	root := t.TempDir()
	// Unknown frontmatter field — KnownFields(true) rejects it,
	// matching the real-world wrap-epic skill bug that motivated
	// the stub mechanism.
	writeFile(t, root, "work/epics/E-01-platform/epic.md", `---
id: E-01
title: Platform
status: done
completed: 2026-04-30
---
`)

	tr, loadErrs, err := Load(context.Background(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loadErrs) != 1 {
		t.Fatalf("loadErrs count = %d, want 1: %+v", len(loadErrs), loadErrs)
	}
	if len(tr.Entities) != 0 {
		t.Errorf("Entities count = %d, want 0 (parse failed)", len(tr.Entities))
	}
	if len(tr.Stubs) != 1 {
		t.Fatalf("Stubs count = %d, want 1 (stub from path-derived id)", len(tr.Stubs))
	}
	stub := tr.Stubs[0]
	if stub.ID != "E-01" || stub.Kind != entity.KindEpic {
		t.Errorf("stub = {ID=%q, Kind=%v}, want {E-01, epic}", stub.ID, stub.Kind)
	}
	if stub.Path != filepath.FromSlash("work/epics/E-01-platform/epic.md") {
		t.Errorf("stub.Path = %q", stub.Path)
	}
}

func TestLoad_ReadFailureRegistersStub(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; cannot create unreadable file")
	}
	root := t.TempDir()
	writeFile(t, root, "work/epics/E-01-platform/epic.md", `---
id: E-01
title: Platform
status: active
---
`)
	// Make it unreadable so os.ReadFile fails.
	abs := filepath.Join(root, "work", "epics", "E-01-platform", "epic.md")
	if err := os.Chmod(abs, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(abs, 0o644) })

	tr, loadErrs, err := Load(context.Background(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loadErrs) != 1 {
		t.Fatalf("loadErrs count = %d, want 1: %+v", len(loadErrs), loadErrs)
	}
	if len(tr.Stubs) != 1 || tr.Stubs[0].ID != "E-01" {
		t.Errorf("expected one stub for E-01 (read failure should still register stub); got %+v", tr.Stubs)
	}
}

func TestLoad_ParseErrorWithUnreadablePathSkipsStub(t *testing.T) {
	// If the path itself doesn't yield a recognizable id (shouldn't
	// happen in practice — PathKind already filtered — but defensive),
	// no stub is registered. The load-error finding still fires.
	root := t.TempDir()
	// Construct a path that PathKind accepts (E-NN dir + epic.md) but
	// whose dir name doesn't carry a valid id prefix. Make the file
	// fail to parse.
	writeFile(t, root, "work/epics/no-id-here/epic.md", `---
not: yaml
  bad: indent
---
`)

	tr, loadErrs, err := Load(context.Background(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loadErrs) != 1 {
		t.Errorf("loadErrs count = %d, want 1", len(loadErrs))
	}
	if len(tr.Stubs) != 0 {
		t.Errorf("Stubs = %+v, want empty (path lacks id)", tr.Stubs)
	}
}

func TestLoad_StraysSkipped(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "work/epics/E-01-platform/epic.md", `---
id: E-01
title: Platform
status: active
---
`)
	// Stray files that don't match entity patterns.
	writeFile(t, root, "work/epics/E-01-platform/notes.md", "stray prose, not an entity")
	writeFile(t, root, "work/contracts/C-001-orders/schema/openapi.yaml", "openapi: 3.1.0\n")
	writeFile(t, root, "README.md", "top-level readme")

	tr, loadErrs, err := Load(context.Background(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loadErrs) != 0 {
		t.Errorf("loadErrs = %+v, want empty (strays should be silently skipped)", loadErrs)
	}
	if len(tr.Entities) != 1 {
		t.Errorf("Entities count = %d, want 1 (the epic)", len(tr.Entities))
	}
}

func TestLoad_PartialLayout(t *testing.T) {
	// Repo has only some of the entity-bearing dirs (a fresh repo with
	// only an ADR, for example). Missing dirs should not error.
	root := t.TempDir()
	writeFile(t, root, "docs/adr/ADR-0001-foundation.md", `---
id: ADR-0001
title: Foundation
status: accepted
---
`)
	tr, loadErrs, err := Load(context.Background(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loadErrs) != 0 {
		t.Errorf("loadErrs = %v", loadErrs)
	}
	if len(tr.Entities) != 1 || tr.Entities[0].Kind != entity.KindADR {
		t.Errorf("Entities = %+v", tr.Entities)
	}
}

func TestTree_ByID(t *testing.T) {
	tr := &Tree{Entities: []*entity.Entity{
		{ID: "E-01", Kind: entity.KindEpic},
		{ID: "M-001", Kind: entity.KindMilestone},
	}}
	if got := tr.ByID("E-01"); got == nil || got.Kind != entity.KindEpic {
		t.Errorf("ByID(E-01) = %v", got)
	}
	if got := tr.ByID("X-99"); got != nil {
		t.Errorf("ByID(X-99) = %v, want nil", got)
	}
}

// TestTree_ByID_AcceptsBothWidths is the AC-2 seam test: a tree
// storing an entity at narrow width is found by a canonical-width
// query, and a tree storing at canonical width is found by a
// narrow-width query. The lookup canonicalizes both sides per
// internal/entity/canonicalize.go::Canonicalize.
//
// Inputs intentionally use narrow forms (E-22, M-007, G-093, etc.):
// these are the parser-tolerance cases by design — the tree-load
// layer accepts the on-disk shape verbatim and the lookup canonicalizes.
func TestTree_ByID_AcceptsBothWidths(t *testing.T) {
	tests := []struct {
		name   string
		stored string
		query  string
	}{
		{"narrow-stored-canonical-query-epic", "E-22", "E-0022"},
		{"canonical-stored-narrow-query-epic", "E-0022", "E-22"},
		{"narrow-stored-canonical-query-milestone", "M-007", "M-0007"},
		{"canonical-stored-narrow-query-milestone", "M-0007", "M-007"},
		{"narrow-stored-canonical-query-gap", "G-093", "G-0093"},
		{"narrow-stored-canonical-query-decision", "D-005", "D-0005"},
		{"narrow-stored-canonical-query-contract", "C-009", "C-0009"},
		{"adr-already-canonical-both-sides", "ADR-0001", "ADR-0001"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, ok := entity.KindFromID(tt.stored)
			if !ok {
				t.Fatalf("KindFromID(%q) ok=false", tt.stored)
			}
			tr := &Tree{Entities: []*entity.Entity{{ID: tt.stored, Kind: kind}}}
			got := tr.ByID(tt.query)
			if got == nil || got.ID != tt.stored {
				t.Errorf("ByID(%q) on tree storing %q = %v, want match", tt.query, tt.stored, got)
			}
			all := tr.ByIDAll(tt.query)
			if len(all) != 1 || all[0].ID != tt.stored {
				t.Errorf("ByIDAll(%q) = %v, want one match for %q", tt.query, all, tt.stored)
			}
		})
	}
}

// TestTree_ByPriorID_AcceptsBothWidths exercises the prior-id lookup's
// width tolerance — a query for the canonical form of a narrow legacy
// id finds the entity that carries the narrow id in PriorIDs (and
// vice versa).
func TestTree_ByPriorID_AcceptsBothWidths(t *testing.T) {
	g := &entity.Entity{
		ID:       "G-0094",
		Kind:     entity.KindGap,
		PriorIDs: []string{"G-093"}, // narrow legacy lineage
	}
	tr := &Tree{Entities: []*entity.Entity{g}}
	for _, q := range []string{"G-093", "G-0093"} {
		if got := tr.ByPriorID(q); got != g {
			t.Errorf("ByPriorID(%q) = %v, want G-0094", q, got)
		}
		if got := tr.ResolveByCurrentOrPriorID(q); got != g {
			t.Errorf("ResolveByCurrentOrPriorID(%q) = %v, want G-0094", q, got)
		}
	}
}

func TestTree_ByPriorIDAndResolve(t *testing.T) {
	// G-003 carries lineage [G-001, G-002] — two prior reallocations.
	// Queries for any of the three should resolve to G-003 via
	// ResolveByCurrentOrPriorID.
	g003 := &entity.Entity{
		ID:       "G-003",
		Kind:     entity.KindGap,
		PriorIDs: []string{"G-001", "G-002"},
	}
	other := &entity.Entity{ID: "G-004", Kind: entity.KindGap}
	tr := &Tree{Entities: []*entity.Entity{g003, other}}

	if got := tr.ByPriorID("G-001"); got != g003 {
		t.Errorf("ByPriorID(G-001) = %v, want G-003", got)
	}
	if got := tr.ByPriorID("G-002"); got != g003 {
		t.Errorf("ByPriorID(G-002) = %v, want G-003", got)
	}
	if got := tr.ByPriorID("G-099"); got != nil {
		t.Errorf("ByPriorID(G-099) = %v, want nil", got)
	}

	for _, q := range []string{"G-001", "G-002", "G-003"} {
		if got := tr.ResolveByCurrentOrPriorID(q); got != g003 {
			t.Errorf("ResolveByCurrentOrPriorID(%s) = %v, want G-003", q, got)
		}
	}
	if got := tr.ResolveByCurrentOrPriorID("G-004"); got != other {
		t.Errorf("ResolveByCurrentOrPriorID(G-004) = %v, want G-004 itself", got)
	}
	if got := tr.ResolveByCurrentOrPriorID("G-099"); got != nil {
		t.Errorf("ResolveByCurrentOrPriorID(G-099) = %v, want nil", got)
	}
}

func TestLoad_ReverseRefs(t *testing.T) {
	root := t.TempDir()
	// Epic with two milestones; the second depends on the first.
	writeFile(t, root, "work/epics/E-01-platform/epic.md", `---
id: E-01
title: Platform
status: active
---
`)
	writeFile(t, root, "work/epics/E-01-platform/M-001-cache.md", `---
id: M-001
title: Cache warmup
status: in_progress
parent: E-01
acs:
  - id: AC-1
    title: warm before requests
    status: open
---
`)
	writeFile(t, root, "work/epics/E-01-platform/M-002-evict.md", `---
id: M-002
title: Eviction policy
status: draft
parent: E-01
depends_on:
  - M-001
---
`)
	// Gap addresses M-001/AC-1 (composite) — should appear in both
	// the AC's referrers AND the milestone's referrers.
	writeFile(t, root, "work/gaps/G-001-thrash.md", `---
id: G-001
title: Cache thrash
status: open
discovered_in: M-001
addressed_by:
  - M-001/AC-1
---
`)
	// Decision relates to E-01.
	writeFile(t, root, "work/decisions/D-001-strategy.md", `---
id: D-001
title: Cache strategy
status: accepted
relates_to:
  - E-01
---
`)
	// ADR superseded by another ADR.
	writeFile(t, root, "docs/adr/ADR-0001-old.md", `---
id: ADR-0001
title: Old policy
status: superseded
superseded_by: ADR-0002
---
`)
	writeFile(t, root, "docs/adr/ADR-0002-new.md", `---
id: ADR-0002
title: New policy
status: accepted
supersedes:
  - ADR-0001
---
`)
	// Contract linked to ADR-0002.
	writeFile(t, root, "work/contracts/C-001-cache/contract.md", `---
id: C-001
title: Cache contract
status: accepted
linked_adrs:
  - ADR-0002
---
`)

	tr, loadErrs, err := Load(context.Background(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loadErrs) != 0 {
		t.Fatalf("loadErrs = %v, want empty", loadErrs)
	}
	if tr.ReverseRefs == nil {
		t.Fatal("ReverseRefs is nil; want non-nil map")
	}

	// On-disk fixtures use narrow widths (E-01, M-001, …) — that's
	// the legacy shape the parser still tolerates. Per AC-2's
	// lookup-seam rule, the reverse-ref map is keyed by canonical id
	// and its value-slices carry canonical referrer ids regardless of
	// on-disk width; ReferencedBy canonicalizes the query.
	cases := []struct {
		target string
		want   []string
	}{
		// E-01 is referenced by both milestones (parent) AND by D-001 (relates_to).
		{"E-01", []string{"D-0001", "M-0001", "M-0002"}},
		{"E-0001", []string{"D-0001", "M-0001", "M-0002"}}, // canonical query: same answer
		// M-001 is referenced by M-002 (depends_on), G-001 (discovered_in),
		// AND G-001 again via the composite-id rollup from G-001.addressed_by:M-001/AC-1.
		// Dedup must collapse the two G-001 mentions into one entry.
		{"M-001", []string{"G-0001", "M-0002"}},
		// Composite key resolves to just G-001 (the addressed_by referrer).
		{"M-001/AC-1", []string{"G-0001"}},
		{"M-0001/AC-1", []string{"G-0001"}}, // canonical query: same answer
		// M-002 is unreferenced.
		{"M-002", nil},
		// ADR-0001 is referenced by ADR-0002.supersedes.
		{"ADR-0001", []string{"ADR-0002"}},
		// ADR-0002 is referenced by ADR-0001.superseded_by AND C-001.linked_adrs.
		{"ADR-0002", []string{"ADR-0001", "C-0001"}},
	}
	for _, tc := range cases {
		t.Run(tc.target, func(t *testing.T) {
			got := tr.ReferencedBy(tc.target)
			if !equalStrings(got, tc.want) {
				t.Errorf("ReferencedBy(%q) = %v, want %v", tc.target, got, tc.want)
			}
		})
	}
}

// TestReaches: the forward-reachability primitive used by the I2.5
// allow-rule. Walks parent / depends_on / addressed_by / etc. edges
// from `from` toward `to`, with composite ids rolling up to their
// parent for traversal. Same fixture shape as TestLoad_ReverseRefs.
func TestReaches(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "work/epics/E-01-platform/epic.md", `---
id: E-01
title: Platform
status: active
---
`)
	writeFile(t, root, "work/epics/E-01-platform/M-001-cache.md", `---
id: M-001
title: Cache warmup
status: in_progress
parent: E-01
acs:
  - id: AC-1
    title: warm before requests
    status: open
---
`)
	writeFile(t, root, "work/epics/E-01-platform/M-002-evict.md", `---
id: M-002
title: Eviction policy
status: draft
parent: E-01
depends_on:
  - M-001
---
`)
	writeFile(t, root, "work/gaps/G-001-thrash.md", `---
id: G-001
title: Cache thrash
status: open
discovered_in: M-001
addressed_by:
  - M-001/AC-1
---
`)

	tr, _, err := Load(context.Background(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	cases := []struct {
		name string
		from string
		to   string
		want bool
	}{
		{"self-loop bare", "E-01", "E-01", true},
		{"self-loop composite", "M-001/AC-1", "M-001/AC-1", true},
		{"composite to its parent", "M-001/AC-1", "M-001", true},
		{"milestone to epic via parent", "M-001", "E-01", true},
		{"milestone via depends_on then parent", "M-002", "M-001", true},
		{"milestone via depends_on chain to epic", "M-002", "E-01", true},
		{"AC under M-001 reaches E-01 by parent rollup", "M-001/AC-1", "E-01", true},
		{"gap reaches AC's parent via addressed_by composite rollup", "G-001", "M-001", true},
		{"gap reaches AC composite directly", "G-001", "M-001/AC-1", true},
		{"gap reaches epic via discovered_in chain", "G-001", "E-01", true},
		{"unreferenced milestone has no path to gap", "M-002", "G-001", false},
		{"backwards: epic does not reach milestone", "E-01", "M-001", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tr.Reaches(tc.from, tc.to); got != tc.want {
				t.Errorf("Reaches(%q, %q) = %v, want %v", tc.from, tc.to, got, tc.want)
			}
		})
	}
}

// TestReachesAny: the multi-source variant for creation acts. New
// entities don't yet exist in the tree; the caller passes the new
// entity's outbound references (read from the proposed frontmatter)
// and asks "does any of them reach the scope-entity."
func TestReachesAny(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "work/epics/E-01-platform/epic.md", `---
id: E-01
title: Platform
status: active
---
`)
	writeFile(t, root, "work/epics/E-01-platform/M-001-cache.md", `---
id: M-001
title: Cache warmup
status: in_progress
parent: E-01
---
`)
	tr, _, err := Load(context.Background(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !tr.ReachesAny([]string{"M-001", "X-99"}, "E-01") {
		t.Error("ReachesAny([M-001 X-99], E-01) = false; want true (M-001 reaches E-01)")
	}
	if tr.ReachesAny([]string{"X-99"}, "E-01") {
		t.Error("ReachesAny([X-99], E-01) = true; want false (X-99 not in tree)")
	}
	if tr.ReachesAny(nil, "E-01") {
		t.Error("ReachesAny(nil, E-01) = true; want false (no froms)")
	}
}

// TestLoad_ReverseRefsEmptyTree verifies that an empty tree yields a
// non-nil empty map — callers can range or index without a nil check.
func TestLoad_ReverseRefsEmptyTree(t *testing.T) {
	tr, _, err := Load(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if tr.ReverseRefs == nil {
		t.Error("ReverseRefs is nil; want non-nil empty map")
	}
	if got := tr.ReferencedBy("E-99"); got != nil {
		t.Errorf("ReferencedBy on empty tree = %v, want nil", got)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestTree_ByKind(t *testing.T) {
	tr := &Tree{Entities: []*entity.Entity{
		{ID: "E-01", Kind: entity.KindEpic},
		{ID: "M-001", Kind: entity.KindMilestone},
		{ID: "M-002", Kind: entity.KindMilestone},
		{ID: "E-02", Kind: entity.KindEpic},
	}}
	got := tr.ByKind(entity.KindMilestone)
	if len(got) != 2 {
		t.Fatalf("ByKind count = %d, want 2", len(got))
	}
	ids := []string{got[0].ID, got[1].ID}
	sort.Strings(ids)
	if ids[0] != "M-001" || ids[1] != "M-002" {
		t.Errorf("ByKind ids = %v", ids)
	}
}

// TestTree_FilterByKindStatuses pins the helper's filter axes and
// id-ascending sort. The chokepoint test for M-072 AC-6: both `aiwf
// list --kind X --status Y` and `aiwf status`'s per-section slices
// route through this single function, so a regression here would
// drift both verbs in lockstep — much easier to spot than two
// independent regressions diverging silently.
func TestTree_FilterByKindStatuses(t *testing.T) {
	tr := &Tree{Entities: []*entity.Entity{
		{ID: "E-02", Kind: entity.KindEpic, Status: "active"},
		{ID: "E-01", Kind: entity.KindEpic, Status: "active"},
		{ID: "E-03", Kind: entity.KindEpic, Status: "proposed"},
		{ID: "G-001", Kind: entity.KindGap, Status: "open"},
		{ID: "G-002", Kind: entity.KindGap, Status: "addressed"},
		{ID: "G-003", Kind: entity.KindGap, Status: "open"},
	}}

	t.Run("kind+single-status filter, sorted by id", func(t *testing.T) {
		got := tr.FilterByKindStatuses(entity.KindEpic, "active")
		ids := make([]string, len(got))
		for i, e := range got {
			ids[i] = e.ID
		}
		want := []string{"E-01", "E-02"}
		if !equalStrings(ids, want) {
			t.Errorf("ids = %v, want %v", ids, want)
		}
	})

	t.Run("kind+multiple-statuses filter (active OR proposed)", func(t *testing.T) {
		got := tr.FilterByKindStatuses(entity.KindEpic, "active", "proposed")
		if len(got) != 3 {
			t.Errorf("count = %d, want 3", len(got))
		}
	})

	t.Run("kind only (no status filter)", func(t *testing.T) {
		got := tr.FilterByKindStatuses(entity.KindGap)
		if len(got) != 3 {
			t.Errorf("count = %d, want 3", len(got))
		}
	})

	t.Run("empty kind keeps every kind", func(t *testing.T) {
		got := tr.FilterByKindStatuses("", "open")
		if len(got) != 2 {
			t.Errorf("count = %d, want 2 (G-001 and G-003)", len(got))
		}
		if got[0].ID != "G-001" || got[1].ID != "G-003" {
			t.Errorf("ids = [%s %s], want [G-001 G-003]", got[0].ID, got[1].ID)
		}
	})

	t.Run("empty result is non-nil empty slice", func(t *testing.T) {
		got := tr.FilterByKindStatuses(entity.KindEpic, "cancelled")
		if got == nil {
			t.Error("expected non-nil empty slice for sentinel-friendly callers")
		}
		if len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})
}
