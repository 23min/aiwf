package check_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/tree"
)

// TestFixture_Clean asserts that the synthetic clean tree produces no
// findings of any severity. If this test breaks, either a check has a
// false positive or the fixture has drifted.
func TestFixture_Clean(t *testing.T) {
	tr, loadErrs, err := tree.Load(context.Background(), "testdata/clean")
	if err != nil {
		t.Fatalf("loading: %v", err)
	}
	if len(loadErrs) != 0 {
		t.Fatalf("loadErrs = %+v", loadErrs)
	}
	got := check.Run(tr, loadErrs)
	if len(got) != 0 {
		t.Errorf("expected zero findings, got:\n%+v", got)
	}
}

// TestFixture_Messy asserts that every finding code documented in
// docs/pocv3/archive/poc-plan-pre-migration.md Session 1 is exercised by the messy tree. The test does
// not assert exact counts — multiple checks fire across many entities,
// and counts shift when the fixture is extended — only that each
// expected code appears at least once.
func TestFixture_Messy(t *testing.T) {
	tr, loadErrs, err := tree.Load(context.Background(), "testdata/messy")
	if err != nil {
		t.Fatalf("loading: %v", err)
	}
	got := check.Run(tr, loadErrs)
	if len(got) == 0 {
		t.Fatal("expected findings, got none")
	}

	seen := make(map[string]bool)
	for _, f := range got {
		key := f.Code
		if f.Subcode != "" {
			key = f.Code + "/" + f.Subcode
		}
		seen[key] = true
	}

	expected := []string{
		"ids-unique",
		"frontmatter-shape",
		"id-path-consistent",
		"status-valid",
		"refs-resolve/unresolved",
		"refs-resolve/wrong-kind",
		"no-cycles/depends_on",
		"no-cycles/supersedes",
		"titles-nonempty",
		"adr-supersession-mutual",
		"gap-resolved-has-resolver",
		"epic-active-no-drafted-milestones",
	}
	var missing []string
	for _, code := range expected {
		if !seen[code] {
			missing = append(missing, code)
		}
	}
	if len(missing) > 0 {
		seenList := make([]string, 0, len(seen))
		for k := range seen {
			seenList = append(seenList, k)
		}
		sort.Strings(seenList)
		t.Errorf("missing expected finding codes: %v\nfindings seen: %v", missing, seenList)
	}

	// All errors should sort before all warnings (Run.sortFindings).
	for i := 1; i < len(got); i++ {
		if got[i-1].Severity == check.SeverityWarning && got[i].Severity == check.SeverityError {
			t.Errorf("findings not sorted: warning at %d precedes error at %d", i-1, i)
		}
	}

	// Also confirm HasErrors agrees with our expectation.
	if !check.HasErrors(got) {
		t.Error("HasErrors = false on the messy fixture")
	}

	_ = cmp.Diff // keep import for future granular asserts
}

// TestFixture_ProliminalCascadeEndToEnd is the literal end-to-end repro
// of the wrap-epic bug: an epic with an unknown `completed:` field and
// 12 entities referencing it through every reference field type. The
// loaded tree → check.Run pipeline must produce exactly 1 finding (the
// load error), not 13. Pre-fix this returned 13 findings; post-fix it
// returns 1 because the loader registers a path-derived stub for E-01
// and refs-resolve consults it. The fixture is built in a tmpdir
// (rather than testdata/) so it doubles as documentation: anyone
// reading this test sees the exact shape of the bug.
func TestFixture_ProliminalCascadeEndToEnd(t *testing.T) {
	root := t.TempDir()

	// E-01 with the unknown `completed:` field — the bug.
	writeFile(t, root, "work/epics/E-01-platform/epic.md", `---
id: E-01
title: Platform
status: done
completed: 2026-04-30
---
`)
	// 5 milestones under E-01. Status is non-terminal so the
	// M-0086 terminal-entity-not-archived rule doesn't fire on
	// them — this fixture's narrative is the refs-resolve cascade,
	// not archive sweep state.
	for i := 1; i <= 5; i++ {
		writeFile(t, root, fmt.Sprintf("work/epics/E-01-platform/M-%03d.md", i), fmt.Sprintf(`---
id: M-%03d
title: Milestone %d
status: in_progress
parent: E-01
---
`, i, i))
	}
	// 5 gaps discovered_in E-01.
	for i := 1; i <= 5; i++ {
		writeFile(t, root, fmt.Sprintf("work/gaps/G-%03d.md", i), fmt.Sprintf(`---
id: G-%03d
title: Gap %d
status: open
discovered_in: E-01
---
`, i, i))
	}
	// 2 decisions relates_to E-01.
	for i := 1; i <= 2; i++ {
		writeFile(t, root, fmt.Sprintf("work/decisions/D-%03d.md", i), fmt.Sprintf(`---
id: D-%03d
title: Decision %d
status: accepted
relates_to: [E-01]
---
`, i, i))
	}

	tr, loadErrs, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := check.Run(tr, loadErrs)

	// Exactly one finding: the load error on E-01's epic.md.
	if len(got) != 1 {
		t.Errorf("expected exactly 1 finding (the load error), got %d:\n%+v", len(got), got)
	}
	// The one finding must be the load error, not a refs-resolve cascade.
	if len(got) > 0 {
		f := got[0]
		if f.Code != "load-error" {
			t.Errorf("the surviving finding should be the load error; got code=%q", f.Code)
		}
		if filepath.ToSlash(f.Path) != "work/epics/E-01-platform/epic.md" {
			t.Errorf("the load error should be on E-01's epic.md; got path=%q", f.Path)
		}
	}
	// And: refs-resolve must have suppressed the 12 cascade findings.
	for _, f := range got {
		if f.Code == "refs-resolve" {
			t.Errorf("refs-resolve cascade not suppressed; saw: %+v", f)
		}
	}
}

// TestFixture_MultipleStubsAndCrossLinks exercises the gnarliest case:
// multiple parse failures simultaneously, with references crossing
// between stubs and real entities in every direction. The 2 stubs +
// the cross-links must produce exactly 2 findings (the load errors)
// and zero cascade noise.
func TestFixture_MultipleStubsAndCrossLinks(t *testing.T) {
	root := t.TempDir()

	// E-01: parse failure (unknown field).
	writeFile(t, root, "work/epics/E-01-platform/epic.md", `---
id: E-01
title: Platform
status: active
completed: 2026-04-30
---
`)
	// M-001: parse failure (also under E-01).
	writeFile(t, root, "work/epics/E-01-platform/M-001-cache.md", `---
id: M-001
title: Cache
status: in_progress
parent: E-01
notes_field_aiwf_does_not_know: yes
---
`)
	// M-002: real, depends_on the stubbed M-001.
	writeFile(t, root, "work/epics/E-01-platform/M-002-bar.md", `---
id: M-002
title: Bar
status: in_progress
parent: E-01
depends_on: [M-001]
---
`)
	// G-001: real, discovered_in the stubbed M-001.
	writeFile(t, root, "work/gaps/G-001-flake.md", `---
id: G-001
title: Flake
status: open
discovered_in: M-001
---
`)
	// D-001: real, relates_to both stubbed entities.
	writeFile(t, root, "work/decisions/D-001-shape.md", `---
id: D-001
title: Shape
status: accepted
relates_to: [E-01, M-001]
---
`)

	tr, loadErrs, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(tr.Stubs) != 2 {
		t.Errorf("expected 2 stubs (E-01, M-001); got %+v", tr.Stubs)
	}
	got := check.Run(tr, loadErrs)

	// Exactly 2 findings — the two load errors. No refs-resolve
	// cascade despite 4 inbound references to the stubbed entities.
	loadErrors := 0
	cascade := 0
	for _, f := range got {
		switch f.Code {
		case "load-error":
			loadErrors++
		case "refs-resolve":
			cascade++
		}
	}
	if loadErrors != 2 {
		t.Errorf("expected 2 load-error findings; got %d in:\n%+v", loadErrors, got)
	}
	if cascade != 0 {
		t.Errorf("expected 0 refs-resolve cascade findings; got %d in:\n%+v", cascade, got)
	}
}

// writeFile mirrors the helper in tree_test.go but lives here so this
// black-box _test package can stay self-contained.
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", abs, err)
	}
}
