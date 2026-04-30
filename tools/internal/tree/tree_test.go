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
