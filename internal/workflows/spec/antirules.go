package spec

// AntiRules returns the closed-set catalog of patterns the kernel
// deliberately does NOT police. Anti-rules clarify scope by negation; they
// are not (Kind, FromState, Verb)-keyed cells in Rules().
//
// The list comprises eleven Pass B §10 entries (R-FP-0166..R-FP-0176) plus
// one Q10 addition (ANTI-0012, the zero-milestone-active legality), per
// M-0123 phase 1 enumeration. Listing order follows Pass B with the Q10
// addition appended.
//
// Each entry carries a structural Statement ("the kernel does NOT...") and
// a Reasoning ("...because..."). Together they pin both the anti-rule and
// why a future contributor shouldn't re-litigate it.
//
// Growth is by spec amendment — surface a new mis-assumed pattern via Pass
// C-style reconciliation, then add a row. See spec.go package comment for
// the anti-rule meta-policy.
func AntiRules() []AntiRule {
	return []AntiRule{
		{
			ID:        "ANTI-0001",
			Statement: "A milestone is NOT required to have >=1 AC. ACs are optional.",
			Reasoning: "Codified in design-decisions.md \"What's not a kernel rule.\" The kernel guards the AC outcome, not its existence.",
			Sources:   RuleSource{FP: []string{"R-FP-0166"}},
		},
		{
			ID:        "ANTI-0002",
			Statement: "A milestone is NOT required to enter in_progress with all ACs in tdd_phase: red. The kernel guards only the outcome (met requires done).",
			Reasoning: "Codified in design-decisions.md \"What's not a kernel rule.\" The flow is the rituals plugin's concern.",
			Sources:   RuleSource{FP: []string{"R-FP-0167"}},
		},
		{
			ID:        "ANTI-0003",
			Statement: "There is NO global AC allocator. AC ids are per-milestone.",
			Reasoning: "Codified in design-decisions.md. ACs are sub-elements of milestones (R-FP-0063); their numbering is local to the parent.",
			Sources:   RuleSource{FP: []string{"R-FP-0168"}},
		},
		{
			ID:        "ANTI-0004",
			Statement: "There is NO AC tombstone beyond status-cancel. The position-stable position-in-acs[] retains the cancelled AC.",
			Reasoning: "Codified in design-decisions.md \"What's not a kernel rule.\" Position stability across cancel preserves AC-N references in commits, history, and other entities.",
			Sources:   RuleSource{FP: []string{"R-FP-0169"}},
		},
		{
			ID:        "ANTI-0005",
			Statement: "There is NO aiwf reactivate or aiwf un-archive verb. Archive is forward-only.",
			Reasoning: "Codified in ADR-0004 \"Reversal.\" File a new entity referencing the archived one instead.",
			Sources:   RuleSource{FP: []string{"R-FP-0170"}},
		},
		{
			ID:        "ANTI-0006",
			Statement: "There is NO event log file, no graph projection file, no hash chain, no monotonic ID counter.",
			Reasoning: "Codified in design-decisions.md \"What the framework needs to do\" and \"What is deliberately not in the PoC.\" Git history is the authoritative event log; ids are per-kind and branch-local.",
			Sources:   RuleSource{FP: []string{"R-FP-0171"}},
		},
		{
			ID:        "ANTI-0007",
			Statement: "There is NO kernel rule about which branch a verb is legal on.",
			Reasoning: "Codified in ADR-0010 and ADR-0011 \"Scope.\" Branch choreography is ADR-0010's layer 4, out of E-0033's scope.",
			Sources:   RuleSource{FP: []string{"R-FP-0172"}},
		},
		{
			ID:        "ANTI-0008",
			Statement: "The kernel makes NO assumption about which Claude Code plugins a consumer should have installed. aiwf.yaml.doctor.recommended_plugins is opt-in; default empty.",
			Reasoning: "Codified in design-decisions.md \"aiwf.yaml config.\" Per-consumer plugin preferences are not kernel concerns.",
			Sources:   RuleSource{FP: []string{"R-FP-0173"}},
		},
		{
			ID:        "ANTI-0009",
			Statement: "The kernel does NOT ship validator binaries (cue, ajv). Validators are declared in aiwf.yaml.contracts.validators and installed via the user's toolchain.",
			Reasoning: "Codified in design-decisions.md \"Contracts.\" The engine owns orchestration; the user owns validators.",
			Sources:   RuleSource{FP: []string{"R-FP-0174"}},
		},
		{
			ID:        "ANTI-0010",
			Statement: "--force cannot be wielded by a non-human actor. A future delegated-force flag is deferred.",
			Reasoning: "Codified in provenance-model.md. Sovereign acts always trace to a named human (principal x agent x scope).",
			Sources:   RuleSource{FP: []string{"R-FP-0175"}},
		},
		{
			ID:        "ANTI-0011",
			Statement: "There is NO \"milestone-must-pre-fail-tests-before-in_progress\" rule. The TDD discipline is rituals-plugin-driven, not kernel-driven.",
			Reasoning: "Codified in design-decisions.md \"What's not a kernel rule.\" The kernel polices the AC-met outcome (requires tdd_phase: done when tdd: required), not the entry condition.",
			Sources:   RuleSource{FP: []string{"R-FP-0176"}},
		},
		{
			ID:        "ANTI-0012",
			Statement: "An epic MAY transition proposed -> active with zero child milestones. Distinct from the epic-active-no-drafted-milestones warning, which fires only when at least one milestone exists and ALL are drafts.",
			Reasoning: "Q10 reconciliation: the warning's guard is \"all milestones drafts\" — zero milestones satisfies that vacuously but is deliberately allowed (an epic may be activated to scaffold milestones into it). Catalogued as anti-rule so the zero-case isn't mistaken for a missing guard.",
			Sources:   RuleSource{},
		},
	}
}
