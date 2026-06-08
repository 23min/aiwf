package branch

import "github.com/23min/aiwf/internal/workflows/spec"

// ac3ExpandedCells returns the M-0162/AC-3 cell-expansion entries:
// 112 cells, one per discriminating E2E subtest across the
// M-0106 / M-0159 / M-0160 / M-0161 surfaces. Generated from the
// CellID/Name pairs stamped into internal/cli/integration/*_test.go
// at AC-3 RED time. Each cell is a catalog-vocabulary entry:
// Outcome=Legal means "the test body's Expect assertion is the
// behavioral pin; this cell exists for the AC-4 bijection mapping."
//
// The bijection invariant at AC-4 will assert:
//  1. Every cell here has at least one Pin (every scenario has a
//     CellID referencing it).
//  2. Every Pin references an ID present in this list (no orphan
//     CellIDs in Scenario literals).
//
// Maintenance: when a scenario is added, removed, or renamed,
// re-stamp via scripts/m0162-stamp-cellid.sh and regenerate
// this file via scripts/m0162-build-ac3-cells.py. The AC-3
// cell-presence test at internal/policies/m0162_ac3_expanded_set_test.go
// pins the CellID → branch.Rules() consistency.
func ac3ExpandedCells() []spec.Rule {
	return []spec.Rule{
		// branch-cell-m0106-baseline-c1 — branch_scenarios: AI commit on bound branch is silent (M-0106 AC-4)
		{
			ID:      "branch-cell-m0106-baseline-c1",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0106-baseline-c2 — branch_scenarios: AI commit on main while bound to epic ritual fires isolation-escape (M-0106 AC-1…
		{
			ID:      "branch-cell-m0106-baseline-c2",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0106-baseline-c3 — branch_scenarios: Force-amend with human actor silences the escape (M-0106 AC-8 legacy override)
		{
			ID:      "branch-cell-m0106-baseline-c3",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac2-c1 — branch_scenarios_ac2: AI commit on different ritual branch fires (M-0106 AC-2)
		{
			ID:      "branch-cell-m0159-ac2-c1",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac2-c10 — branch_scenarios_ac2: AI commit predating opener is silent (M-0106 F-2 ordering)
		{
			ID:      "branch-cell-m0159-ac2-c10",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac2-c11 — branch_scenarios_ac2: Bound branch absent from oracle (operator typo) fires (M-0106 T-6)
		{
			ID:      "branch-cell-m0159-ac2-c11",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac2-c12 — branch_scenarios_ac2: Human-actor commit on wrong branch is silent (M-0106 filter specificity #58)
		{
			ID:      "branch-cell-m0159-ac2-c12",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac2-c13 — branch_scenarios_ac2: AI commit on non-ritual branch is silent (oracle unknown) (M-0106 UnknownBranch)
		{
			ID:      "branch-cell-m0159-ac2-c13",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac2-c2 — branch_scenarios_ac2: AI commit on bound branch with paused scope interposed is silent (M-0106 AC-5 re…
		{
			ID:      "branch-cell-m0159-ac2-c2",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac2-c3 — branch_scenarios_ac2: Human merge bringing AI commits via second-parent is silent (M-0106 AC-7)
		{
			ID:      "branch-cell-m0159-ac2-c3",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac2-c4 — branch_scenarios_ac2: AI commit with no scope opened is silent (M-0106 AC-9)
		{
			ID:      "branch-cell-m0159-ac2-c4",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac2-c5 — branch_scenarios_ac2: Three violating AI commits produce three isolation-escape findings (M-0106 AC-10…
		{
			ID:      "branch-cell-m0159-ac2-c5",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac2-c6 — branch_scenarios_ac2: isolation-escape finding has severity warning (M-0106 AC-11)
		{
			ID:      "branch-cell-m0159-ac2-c6",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac2-c7 — branch_scenarios_ac2: isolation-escape hint names cherry-pick AND force override paths (M-0106 AC-12)
		{
			ID:      "branch-cell-m0159-ac2-c7",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac2-c8 — branch_scenarios_ac2: AI commit after scope-ended is silent (M-0106 F-3 negative arm)
		{
			ID:      "branch-cell-m0159-ac2-c8",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac2-c9 — branch_scenarios_ac2: AI commit before scope-ended still fires (M-0106 F-3 positive arm)
		{
			ID:      "branch-cell-m0159-ac2-c9",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac4-c1 — branch_scenarios_ac4: isolation-escape acknowledged is silent (M-0159/AC-4: E2E for AC-3 lift)
		{
			ID:      "branch-cell-m0159-ac4-c1",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac4-c2 — branch_scenarios_ac4: isolation-escape NOT acknowledged on its own SHA still fires (M-0159/AC-4: per-S…
		{
			ID:      "branch-cell-m0159-ac4-c2",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac4-c3 — branch_scenarios_ac4: forced-untrailered acknowledged is silent (M-0159/AC-4: G-0214 asymmetry closed)
		{
			ID:      "branch-cell-m0159-ac4-c3",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac4-c4 — branch_scenarios_ac4: forced-untrailered NOT acknowledged on its own SHA still fires (M-0159/AC-4: per…
		{
			ID:      "branch-cell-m0159-ac4-c4",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac4-c5 — branch_scenarios_ac4: isolation-escape acknowledged preserves AI authorship on the original escape com…
		{
			ID:      "branch-cell-m0159-ac4-c5",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac5-c1 — branch_scenarios_ac5: trailer-verb-unknown without acknowledgment fires as warning (M-0159/AC-5: basel…
		{
			ID:      "branch-cell-m0159-ac5-c1",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac5-c2 — branch_scenarios_ac5: trailer-verb-unknown acknowledged is silent (M-0159/AC-5: docstring promise mech…
		{
			ID:      "branch-cell-m0159-ac5-c2",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac5-c3 — branch_scenarios_ac5: trailer-verb-unknown NOT acknowledged on its own SHA still fires (M-0159/AC-5: p…
		{
			ID:      "branch-cell-m0159-ac5-c3",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac5-c4 — branch_scenarios_ac5: trailer-verb-unknown acknowledged preserves fabricated aiwf-verb trailer on orig…
		{
			ID:      "branch-cell-m0159-ac5-c4",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac6-c1 — branch_scenarios_ac6: AI commit cherry-picked -x by human (distinct committer) to non-bound branch is …
		{
			ID:      "branch-cell-m0159-ac6-c1",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac6-c2 — branch_scenarios_ac6: Cherry-pick -x by same identity (committer == author, no gap) still fires (M-015…
		{
			ID:      "branch-cell-m0159-ac6-c2",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac6-c3 — branch_scenarios_ac6: Cherry-pick WITHOUT -x by distinct committer (gap, no marker) still fires (M-015…
		{
			ID:      "branch-cell-m0159-ac6-c3",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0159-ac6-c4 — branch_scenarios_ac6: AI commit on non-bound branch without cherry-pick fires (M-0159/AC-6: baseline p…
		{
			ID:      "branch-cell-m0159-ac6-c4",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0160-ac4-c1 — id_rename_untrailered_scenarios: inline git mv of an id-bearing entity file with no aiwf-verb trailer fires id-re…
		{
			ID:      "branch-cell-m0160-ac4-c1",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0160-ac4-c2 — id_rename_untrailered_scenarios: aiwf rename (with aiwf-verb: rename trailer) does NOT fire id-rename-untrailered…
		{
			ID:      "branch-cell-m0160-ac4-c2",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0160-ac4-c3 — id_rename_untrailered_scenarios: aiwf acknowledge-illegal silences id-rename-untrailered for the specific SHA (M-…
		{
			ID:      "branch-cell-m0160-ac4-c3",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0160-ac4-c4 — id_rename_untrailered_scenarios: inline git mv of a non-entity file does NOT fire id-rename-untrailered (M-0160/A…
		{
			ID:      "branch-cell-m0160-ac4-c4",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac1-github-classic-master — authorize_scenarios: trunk shape github-classic-master
		{
			ID:      "branch-cell-m0161-ac1-github-classic-master",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac1-main — authorize_scenarios: trunk shape main
		{
			ID:      "branch-cell-m0161-ac1-main",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac1-operator-chosen-dev — authorize_scenarios: trunk shape operator-chosen-dev
		{
			ID:      "branch-cell-m0161-ac1-operator-chosen-dev",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac1-operator-chosen-trunk — authorize_scenarios: trunk shape operator-chosen-trunk
		{
			ID:      "branch-cell-m0161-ac1-operator-chosen-trunk",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac2-epic_to_epic — authorize_scenarios: rung-pair epic_to_epic
		{
			ID:      "branch-cell-m0161-ac2-epic_to_epic",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac2-epic_to_milestone — authorize_scenarios: rung-pair epic_to_milestone
		{
			ID:      "branch-cell-m0161-ac2-epic_to_milestone",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac2-epic_to_patch — authorize_scenarios: rung-pair epic_to_patch
		{
			ID:      "branch-cell-m0161-ac2-epic_to_patch",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac2-epic_to_trunk — authorize_scenarios: rung-pair epic_to_trunk
		{
			ID:      "branch-cell-m0161-ac2-epic_to_trunk",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac2-milestone_to_epic — authorize_scenarios: rung-pair milestone_to_epic
		{
			ID:      "branch-cell-m0161-ac2-milestone_to_epic",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac2-milestone_to_milestone — authorize_scenarios: rung-pair milestone_to_milestone
		{
			ID:      "branch-cell-m0161-ac2-milestone_to_milestone",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac2-milestone_to_patch — authorize_scenarios: rung-pair milestone_to_patch
		{
			ID:      "branch-cell-m0161-ac2-milestone_to_patch",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac2-milestone_to_trunk — authorize_scenarios: rung-pair milestone_to_trunk
		{
			ID:      "branch-cell-m0161-ac2-milestone_to_trunk",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac2-patch_to_epic — authorize_scenarios: rung-pair patch_to_epic
		{
			ID:      "branch-cell-m0161-ac2-patch_to_epic",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac2-patch_to_milestone — authorize_scenarios: rung-pair patch_to_milestone
		{
			ID:      "branch-cell-m0161-ac2-patch_to_milestone",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac2-patch_to_patch — authorize_scenarios: rung-pair patch_to_patch
		{
			ID:      "branch-cell-m0161-ac2-patch_to_patch",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac2-patch_to_trunk — authorize_scenarios: rung-pair patch_to_trunk
		{
			ID:      "branch-cell-m0161-ac2-patch_to_trunk",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac2-sovereign-override — authorize_scenarios: inline pin in authorize_scenarios_test.go
		{
			ID:      "branch-cell-m0161-ac2-sovereign-override",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac2-trunk_to_epic — authorize_scenarios: rung-pair trunk_to_epic
		{
			ID:      "branch-cell-m0161-ac2-trunk_to_epic",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac2-trunk_to_milestone — authorize_scenarios: rung-pair trunk_to_milestone
		{
			ID:      "branch-cell-m0161-ac2-trunk_to_milestone",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac2-trunk_to_patch — authorize_scenarios: rung-pair trunk_to_patch
		{
			ID:      "branch-cell-m0161-ac2-trunk_to_patch",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac2-trunk_to_trunk — authorize_scenarios: rung-pair trunk_to_trunk
		{
			ID:      "branch-cell-m0161-ac2-trunk_to_trunk",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac3-c1 — isolation_escape_oracle_scenarios: AC-3 cell 1: all refs healthy, no AI commits → isolation-escape silent
		{
			ID:      "branch-cell-m0161-ac3-c1",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac3-c10 — isolation_escape_oracle_scenarios: AC-3 cell 5 (paired): all ritual refs corrupted → oracle-failure fires advisory
		{
			ID:      "branch-cell-m0161-ac3-c10",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac3-c11 — isolation_escape_oracle_scenarios: AC-3 cell 7: repo with only non-ritual refs → isolation-escape silent
		{
			ID:      "branch-cell-m0161-ac3-c11",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac3-c12 — isolation_escape_oracle_scenarios: AC-3 cell 7 (paired): repo with only non-ritual refs → oracle-failure silent
		{
			ID:      "branch-cell-m0161-ac3-c12",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac3-c13 — isolation_escape_oracle_scenarios: AC-3 sovereign: acknowledged escape silences isolation-escape + unrelated corrup…
		{
			ID:      "branch-cell-m0161-ac3-c13",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac3-c14 — isolation_escape_oracle_scenarios: AC-3 sovereign (paired): acknowledged escape + unrelated corrupt ref → oracle-fa…
		{
			ID:      "branch-cell-m0161-ac3-c14",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac3-c2 — isolation_escape_oracle_scenarios: AC-3 cell 1 (paired): all refs healthy, no AI commits → oracle-failure silent
		{
			ID:      "branch-cell-m0161-ac3-c2",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac3-c3 — isolation_escape_oracle_scenarios: AC-3 cell 2: all refs healthy + AI escape → isolation-escape fires
		{
			ID:      "branch-cell-m0161-ac3-c3",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac3-c4 — isolation_escape_oracle_scenarios: AC-3 cell 2 (paired): all refs healthy + AI escape → oracle-failure silent
		{
			ID:      "branch-cell-m0161-ac3-c4",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac3-c5 — isolation_escape_oracle_scenarios: AC-3 cell 3: one ritual ref corrupted, no AI escape → isolation-escape silent
		{
			ID:      "branch-cell-m0161-ac3-c5",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac3-c6 — isolation_escape_oracle_scenarios: AC-3 cell 3 (paired): one ritual ref corrupted, no AI escape → oracle-failure fi…
		{
			ID:      "branch-cell-m0161-ac3-c6",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac3-c7 — isolation_escape_oracle_scenarios: AC-3 cell 4: one ritual ref corrupted + escape on healthy ref → isolation-escape…
		{
			ID:      "branch-cell-m0161-ac3-c7",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac3-c8 — isolation_escape_oracle_scenarios: AC-3 cell 4 (paired): one ritual ref corrupted + escape on healthy ref → oracle-…
		{
			ID:      "branch-cell-m0161-ac3-c8",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac3-c9 — isolation_escape_oracle_scenarios: AC-3 cell 5: all ritual refs corrupted → isolation-escape silent
		{
			ID:      "branch-cell-m0161-ac3-c9",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac4-c1 — isolation_escape_shallow_scenarios: AC-4 cell 1: full clone, no AI commits → isolation-escape silent
		{
			ID:      "branch-cell-m0161-ac4-c1",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac4-c10 — isolation_escape_shallow_scenarios: AC-4 cell 6: unshallow → isolation-escape works again (same as full-clone path)
		{
			ID:      "branch-cell-m0161-ac4-c10",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac4-c11 — isolation_escape_shallow_scenarios: AC-4 sovereign: shallow + force-amended escape → isolation-escape silent
		{
			ID:      "branch-cell-m0161-ac4-c11",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac4-c12 — isolation_escape_shallow_scenarios: AC-4 sovereign (paired): shallow + force-amend → shallow-clone STILL fires
		{
			ID:      "branch-cell-m0161-ac4-c12",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac4-c2 — isolation_escape_shallow_scenarios: AC-4 cell 1 (paired): full clone, no AI commits → shallow-clone silent
		{
			ID:      "branch-cell-m0161-ac4-c2",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac4-c3 — isolation_escape_shallow_scenarios: AC-4 cell 2: full clone + AI escape → isolation-escape fires
		{
			ID:      "branch-cell-m0161-ac4-c3",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac4-c4 — isolation_escape_shallow_scenarios: AC-4 cell 2 (paired): full clone + AI escape → shallow-clone silent
		{
			ID:      "branch-cell-m0161-ac4-c4",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac4-c5 — isolation_escape_shallow_scenarios: AC-4 cell 3: shallow clone + AI escape → isolation-escape silent (fail-shut)
		{
			ID:      "branch-cell-m0161-ac4-c5",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac4-c6 — isolation_escape_shallow_scenarios: AC-4 cell 3 (paired): shallow clone + AI escape → shallow-clone fires warning
		{
			ID:      "branch-cell-m0161-ac4-c6",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac4-c7 — isolation_escape_shallow_scenarios: AC-4 cell 4: shallow clone, no AI escape → isolation-escape silent
		{
			ID:      "branch-cell-m0161-ac4-c7",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac4-c8 — isolation_escape_shallow_scenarios: AC-4 cell 4 (paired): shallow clone, no AI escape → shallow-clone fires
		{
			ID:      "branch-cell-m0161-ac4-c8",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac4-c9 — isolation_escape_shallow_scenarios: AC-4 cell 5: shallow depth=N + escape within window → isolation-escape STILL sil…
		{
			ID:      "branch-cell-m0161-ac4-c9",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac5-c1 — isolation_escape_force_push_scenarios: AC-5 cell 1: full clone + AI escape (no force-push) → isolation-escape fires
		{
			ID:      "branch-cell-m0161-ac5-c1",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac5-c2 — isolation_escape_force_push_scenarios: AC-5 cell 1 (paired): no force-push, AI escape → orphan finding silent
		{
			ID:      "branch-cell-m0161-ac5-c2",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac5-c3 — isolation_escape_force_push_scenarios: AC-5 cell 2: no force-push, no escape → orphan finding silent
		{
			ID:      "branch-cell-m0161-ac5-c3",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac5-c4 — isolation_escape_force_push_scenarios: AC-5 cell 3: force-push orphans AI commit → orphan finding fires
		{
			ID:      "branch-cell-m0161-ac5-c4",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac5-c5 — isolation_escape_force_push_scenarios: AC-5 cell 3 (paired): force-push orphans AI commit → isolation-escape silent
		{
			ID:      "branch-cell-m0161-ac5-c5",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac5-c6 — isolation_escape_force_push_scenarios: AC-5 cell 4: force-push orphans non-AI commit → orphan finding silent
		{
			ID:      "branch-cell-m0161-ac5-c6",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac5-c7 — isolation_escape_force_push_scenarios: AC-5 cell 7: reflog disabled + force-push → oracle-failure advisory fires (AC-3 …
		{
			ID:      "branch-cell-m0161-ac5-c7",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac5-c8 — isolation_escape_force_push_scenarios: AC-5 cell 7 (paired): reflog disabled + force-push → orphan finding silent (no r…
		{
			ID:      "branch-cell-m0161-ac5-c8",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac5-c9 — isolation_escape_force_push_scenarios: AC-5 cell 5: force-push orphans AI commit + ack → orphan finding silent (G-0226 + G-0236)
		{
			ID:      "branch-cell-m0161-ac5-c9",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac6-c1 — isolation_escape_rename_scenarios: AC-6 cell 1: no rename, AI on bound branch → silent
		{
			ID:      "branch-cell-m0161-ac6-c1",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac6-c2 — isolation_escape_rename_scenarios: AC-6 cell 2: no rename + AI escape → fires
		{
			ID:      "branch-cell-m0161-ac6-c2",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac6-c3 — isolation_escape_rename_scenarios: AC-6 cell 3: rename foo→bar + AI on bar (renamed-to) → silent
		{
			ID:      "branch-cell-m0161-ac6-c3",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac6-c4 — isolation_escape_rename_scenarios: AC-6 cell 4: rename foo→bar + AI on baz (cut from renamed) → fires
		{
			ID:      "branch-cell-m0161-ac6-c4",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac6-c5 — isolation_escape_rename_scenarios: AC-6 cell 5: rename foo→bar→foo + AI on foo → silent
		{
			ID:      "branch-cell-m0161-ac6-c5",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac6-c6 — isolation_escape_rename_scenarios: AC-6 cell 7: squat collision (orphan squat) → silent (SHA resolves to renamed)
		{
			ID:      "branch-cell-m0161-ac6-c6",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac6-c7 — isolation_escape_rename_scenarios: AC-6 cell 8: legacy authorize (no SHA), no rename, AI on bound branch → silent
		{
			ID:      "branch-cell-m0161-ac6-c7",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac6-c8 — isolation_escape_rename_scenarios: AC-6 cell 9 (DOCUMENTED LEGACY CARVE-OUT): legacy authorize + rename → fires (G-…
		{
			ID:      "branch-cell-m0161-ac6-c8",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac6-c9 — isolation_escape_rename_scenarios: AC-6 cell 6: bound branch deleted entirely → isolation-escape silent (fail-shut)
		{
			ID:      "branch-cell-m0161-ac6-c9",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac7-c1 — detached_head_scenarios: inline pin in detached_head_scenarios_test.go
		{
			ID:      "branch-cell-m0161-ac7-c1",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac7-c2 — detached_head_scenarios: inline pin in detached_head_scenarios_test.go
		{
			ID:      "branch-cell-m0161-ac7-c2",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac7-c3 — detached_head_scenarios: inline pin in detached_head_scenarios_test.go
		{
			ID:      "branch-cell-m0161-ac7-c3",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac7-c4 — detached_head_scenarios: inline pin in detached_head_scenarios_test.go
		{
			ID:      "branch-cell-m0161-ac7-c4",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac7-c5 — detached_head_scenarios: inline pin in detached_head_scenarios_test.go
		{
			ID:      "branch-cell-m0161-ac7-c5",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac7-c6 — detached_head_scenarios: inline pin in detached_head_scenarios_test.go
		{
			ID:      "branch-cell-m0161-ac7-c6",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac7-c7 — detached_head_scenarios: inline pin in detached_head_scenarios_test.go
		{
			ID:      "branch-cell-m0161-ac7-c7",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac8-c1 — promote_wrong_branch_scenarios: AC-8 cell 1: epic active on trunk → silent (baseline)
		{
			ID:      "branch-cell-m0161-ac8-c1",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac8-c2 — promote_wrong_branch_scenarios: AC-8 cell 2: milestone in_progress on parent epic branch → silent (baseline)
		{
			ID:      "branch-cell-m0161-ac8-c2",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac8-c3 — promote_wrong_branch_scenarios: AC-8 cell 3: epic active on epic/E-X branch → fires
		{
			ID:      "branch-cell-m0161-ac8-c3",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac8-c4 — promote_wrong_branch_scenarios: AC-8 cell 4: milestone in_progress on milestone/M-Y branch → fires
		{
			ID:      "branch-cell-m0161-ac8-c4",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac8-c5 — promote_wrong_branch_scenarios: AC-8 cell 5: milestone in_progress on trunk (skipped parent epic) → fires
		{
			ID:      "branch-cell-m0161-ac8-c5",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac8-c6 — promote_wrong_branch_scenarios: AC-8 cell 7: non-activating promote (epic active → done) on wrong branch → silen…
		{
			ID:      "branch-cell-m0161-ac8-c6",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac8-c7 — promote_wrong_branch_scenarios: AC-8 cell 8: wrong-branch promote + aiwf acknowledge-illegal → silent
		{
			ID:      "branch-cell-m0161-ac8-c7",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-m0161-ac8-c8 — promote_wrong_branch_scenarios: AC-8 cell 9: wrong-branch promote --force --reason → silent (per-commit override…
		{
			ID:      "branch-cell-m0161-ac8-c8",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
	}
}
