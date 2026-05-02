package tree

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/23min/ai-workflow-v2/tools/internal/entity"
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
	abs := filepath.Join(root, "work/epics/E-01-platform/epic.md")
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

	cases := []struct {
		target string
		want   []string
	}{
		// E-01 is referenced by both milestones (parent) AND by D-001 (relates_to).
		{"E-01", []string{"D-001", "M-001", "M-002"}},
		// M-001 is referenced by M-002 (depends_on), G-001 (discovered_in),
		// AND G-001 again via the composite-id rollup from G-001.addressed_by:M-001/AC-1.
		// Dedup must collapse the two G-001 mentions into one entry.
		{"M-001", []string{"G-001", "M-002"}},
		// Composite key resolves to just G-001 (the addressed_by referrer).
		{"M-001/AC-1", []string{"G-001"}},
		// M-002 is unreferenced.
		{"M-002", nil},
		// ADR-0001 is referenced by ADR-0002.supersedes.
		{"ADR-0001", []string{"ADR-0002"}},
		// ADR-0002 is referenced by ADR-0001.superseded_by AND C-001.linked_adrs.
		{"ADR-0002", []string{"ADR-0001", "C-001"}},
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
