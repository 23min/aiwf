package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// (extractMarkdownSection is shared with adr_0007_test.go in the same
// package — section-scoped extraction at the named heading level, used
// by M-0154's structural assertions per .guidance/project/testing.md's "substring
// assertions are not structural assertions" rule.)

// loadADR0015 reads ADR-0015 via the kernel loader. Returns the body
// content as a string. The test fails fast if the ADR has not been
// allocated yet (the AC-1 RED state).
func loadADR0015(t *testing.T) string {
	t.Helper()
	root, tr := sharedRepoTree(t)
	e := tr.ByID("ADR-0015")
	if e == nil {
		t.Fatal("AC-1: ADR-0015 not found in tree — allocate via `aiwf add adr` before this test passes")
	}
	data, err := os.ReadFile(filepath.Join(root, e.Path))
	if err != nil {
		t.Fatalf("reading ADR-0015 at %s: %v", e.Path, err)
	}
	return string(data)
}

// TestM0154_AC1_ADR0015RecordsConsentGatedStance asserts M-0154/AC-1:
// ADR-0015 exists under `docs/adr/` and records the consent-gated
// settings.json stance with all three load-bearing details named in its
// `## Decision` section — the interactive consent prompt, the non-TTY
// flag, and the project-scope settings.local.json default. Per the
// milestone spec, M-0154 is the sole owner of this prose; M-0156's
// wiring milestone implements what the ADR records but does not
// re-author it.
//
// Two layers of assertion per .guidance/project/testing.md's "substring assertions are not
// structural assertions" rule:
//
//   - Structural: the body carries `## Context` / `## Decision` /
//     `## Consequences` headings (the named-section shape the milestone
//     spec requires).
//   - Section-scoped: the three load-bearing mentions live *inside*
//     `## Decision` — not floating in Context or Consequences. The
//     `extractMarkdownSection` helper bounds each assertion to the
//     Decision body.
func TestM0154_AC1_ADR0015RecordsConsentGatedStance(t *testing.T) {
	t.Parallel()
	body := loadADR0015(t)

	for _, section := range []string{"Context", "Decision", "Consequences"} {
		if extractMarkdownSection(body, 2, section) == "" {
			t.Errorf("AC-1: ADR-0015 must carry a `## %s` section (the named-section shape the milestone spec requires)", section)
		}
	}

	decision := extractMarkdownSection(body, 2, "Decision")
	if decision == "" {
		// The structural check above already reported this; further
		// section-scoped assertions would be vacuous, so return.
		return
	}

	// The interactive consent prompt. The literal `[y/N]` is the
	// canonical CLI prompt shape and is what the spec names.
	if !strings.Contains(decision, "[y/N]") {
		t.Errorf("AC-1: ADR-0015's `## Decision` section must name the interactive consent prompt `[y/N]`")
	}

	// The non-TTY consent flag.
	if !strings.Contains(decision, "--wire-settings") {
		t.Errorf("AC-1: ADR-0015's `## Decision` section must name the `--wire-settings` flag (the non-TTY consent mechanism)")
	}

	// The project-scope settings target. `settings.local.json` (gitignored,
	// personal) is the deliberate non-shared target — not the tracked
	// `settings.json` which would force a broken statusline on teammates.
	if !strings.Contains(decision, "settings.local.json") {
		t.Errorf("AC-1: ADR-0015's `## Decision` section must name `settings.local.json` as the project-scope target (not the shared `settings.json`)")
	}
}

// TestM0154_AC3 retired: the function it asserted on
// (appendMarketplaceOverlapReport) was removed as part of G-0194's
// marketplace-retirement completion. The consent-gated stance that
// AC-3 pinned is now expressed solely in .guidance/project/environment.md §"Operator setup"
// (the statusline consent gate).
