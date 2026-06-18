package branchparse

import "testing"

// TestParseEntityFromBranch covers the ritual-shape branch grammar
// defined by ADR-0010: `epic/E-NNNN-<slug>`, `milestone/M-NNNN-<slug>`,
// `patch/g-NNNN-<slug>` (case-insensitive id segment). The prefix and
// the id kind must agree (G-0198) — `epic/M-...`, `milestone/E-...`,
// `patch/E-...` and other incoherent or non-ritual shapes yield "".
// This is the source of truth M-0102 lifts out of
// internal/cli/status/worktrees.go so M-0103's preflight and the
// existing aiwf status --worktrees correlation share one regex set.
func TestParseEntityFromBranch(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, branch, want string
	}{
		{"epic branch with slug", "epic/E-0010-cobra-and-completion", "E-0010"},
		{"epic branch id-only", "epic/E-0010", "E-0010"},
		{"milestone branch with slug", "milestone/M-0007-cache", "M-0007"},
		{"milestone branch id-only", "milestone/M-0007", "M-0007"},
		{"patch branch lowercase id", "patch/g-0099-isolation", "G-0099"},
		{"patch branch uppercase id", "patch/G-0099-isolation", "G-0099"},
		{"narrow-legacy id width preserved on output", "epic/E-01-old", "E-01"},
		{"main branch returns empty", "main", ""},
		{"empty branch returns empty", "", ""},
		{"fix prefix returns empty", "fix/something", ""},
		{"chore prefix returns empty", "chore/something", ""},
		{"patch without id segment returns empty", "patch/some-topic", ""},
		{"epic without id segment returns empty", "epic/no-id-here", ""},
		// G-0198: prefix and id kind must agree. Incoherent
		// combinations parse to "" so the status --worktrees correlator
		// surfaces the typo instead of silently miscorrelating a
		// hand-created branch to the wrong entity.
		{"milestone/ with E- id rejected (prefix-id mismatch)", "milestone/E-0010-mismatch", ""},
		{"epic/ with M- id rejected (prefix-id mismatch)", "epic/M-0001-foo", ""},
		{"patch/ with M- id rejected (prefix-id mismatch)", "patch/M-0042-foo", ""},
		{"patch/ with E- id rejected (prefix-id mismatch)", "patch/E-0042-foo", ""},
		{"epic/ with G- id rejected (prefix-id mismatch)", "epic/G-0001-foo", ""},
		{"milestone/ with G- id rejected (prefix-id mismatch)", "milestone/G-0001-foo", ""},
		// Per-prefix case-insensitivity of the id segment is preserved
		// (G-0198 constrains the kind letter, not its case).
		{"epic/ lowercase e- id still accepted (case-insensitive)", "epic/e-0001-x", "E-0001"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := ParseEntityFromBranch(tc.branch); got != tc.want {
				t.Errorf("ParseEntityFromBranch(%q) = %q, want %q", tc.branch, got, tc.want)
			}
		})
	}
}

// TestRungOf pins the M-0161/AC-2 (G-0201) rung classifier. The helper
// maps a branch name to its ritual rung — "trunk", "epic", "milestone",
// "patch", or "" (no match). The trunk-rung detection is config-driven,
// not regex-only: the caller passes the configured trunk short-name
// (sourced from Config.TrunkBranchShortName() per AC-1) so a repo using
// `master` (or any other operator-chosen trunk) gets the right
// classification.
//
// The verb-layer authorize carve-out uses (RungOf(current, trunk),
// RungOf(target, trunk)) as the input to LegalRungPair so the rung-pair
// predicate refuses cross-rung typos and up-the-tree shapes (12 illegal
// cells) while accepting the 4 legitimate ritual flows (trunk→epic,
// epic→milestone, milestone→patch, epic→patch).
//
// Per AC-2 §"Auxiliary unit tests": diagnostic, not load-bearing — the
// 17-cell E2E table at
// internal/cli/integration/authorize_scenarios_test.go is the
// behavioral-correctness pin.
func TestRungOf(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		branch     string
		trunkShort string
		wantRung   string
	}{
		// Trunk shapes — config-driven match.
		{"trunk-main-on-main-repo", "main", "main", "trunk"},
		{"trunk-master-on-master-repo", "master", "master", "trunk"},
		{"trunk-dev-on-dev-repo", "dev", "dev", "trunk"},
		{"main-on-master-repo-not-trunk", "main", "master", ""},
		{"master-on-main-repo-not-trunk", "master", "main", ""},

		// Ritual shapes — rung detected regardless of trunkShort.
		{"epic-slug", "epic/E-0001-engine", "main", "epic"},
		{"epic-id-only", "epic/E-0001", "main", "epic"},
		{"milestone-slug", "milestone/M-0007-cache", "main", "milestone"},
		{"milestone-id-only", "milestone/M-0007", "main", "milestone"},
		{"patch-lowercase-id", "patch/g-0099-fix", "main", "patch"},
		{"patch-uppercase-id", "patch/G-0099-fix", "main", "patch"},

		// Ritual rung is INDEPENDENT of trunk-short-name (i.e. a
		// `master`-repo still sees `epic/E-X` as "epic").
		{"epic-on-master-repo", "epic/E-0001-engine", "master", "epic"},

		// Non-ritual and degenerate inputs → "".
		{"empty-branch", "", "main", ""},
		{"empty-branch-empty-trunk", "", "", ""},
		{"feature-prefix", "feature/foo", "main", ""},
		{"fix-prefix", "fix/typo", "main", ""},
		{"chore-prefix", "chore/lint", "main", ""},
		{"patch-without-id", "patch/some-topic", "main", ""},
		{"epic-without-id", "epic/no-id", "main", ""},

		// Empty trunkShort: no branch can be classified as trunk
		// (the empty-guard prevents silent coincidence with an
		// empty-CurrentBranch detached-HEAD state).
		{"main-with-empty-trunk", "main", "", ""},
		{"epic-with-empty-trunk-still-epic", "epic/E-0001-engine", "", "epic"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := RungOf(tc.branch, tc.trunkShort)
			if got != tc.wantRung {
				t.Errorf("RungOf(%q, trunk=%q) = %q, want %q",
					tc.branch, tc.trunkShort, got, tc.wantRung)
			}
		})
	}
}

// TestLegalRungPair pins the M-0161/AC-2 (G-0201) closed legal set:
//
//	{(trunk, epic), (epic, milestone), (milestone, patch), (epic, patch)}
//
// Every other (rung, rung) pair refuses. Exhaustive 5×5 = 25-cell
// enumeration (4 ritual rungs + "") catches drift if anyone widens
// the legal set without thinking it through.
//
// The legal pairs encode the ritual flows ADR-0010 names:
//   - trunk → epic        — aiwfx-start-epic (sovereign promote +
//     authorize on trunk; epic branch cut next)
//   - epic → milestone    — aiwfx-start-milestone from parent epic
//   - milestone → patch   — wf-patch under a milestone
//   - epic → patch        — wf-patch directly under an epic, skipping
//     an intermediate milestone (deliberate
//     operator-intent; not a typo)
//
// All other combinations are typos (same-rung, cross-rung) or
// up-the-tree shapes (milestone→epic, patch→milestone, etc.) and
// refuse. The empty-rung pair-set (anything involving "") also
// refuses — the rung predicate is only meaningful when both sides
// classify.
func TestLegalRungPair(t *testing.T) {
	t.Parallel()
	rungs := []string{"trunk", "epic", "milestone", "patch", ""}
	legalSet := map[[2]string]bool{
		{"trunk", "epic"}:      true,
		{"epic", "milestone"}:  true,
		{"milestone", "patch"}: true,
		{"epic", "patch"}:      true,
	}
	for _, current := range rungs {
		for _, target := range rungs {
			pair := [2]string{current, target}
			want := legalSet[pair]
			name := "current=" + current + "/target=" + target
			if current == "" {
				name = "current=EMPTY/target=" + target
			}
			if target == "" {
				name += "_targetEMPTY"
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				got := LegalRungPair(current, target)
				if got != want {
					t.Errorf("LegalRungPair(%q, %q) = %v, want %v",
						current, target, got, want)
				}
			})
		}
	}
}
