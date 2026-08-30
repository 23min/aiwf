package policies

// The exemptions from PolicyShippedProseAssertion, split by the ground that
// earns each one. Keeping them in separate maps is what makes "closed to two
// classes" a property of the code rather than a claim in a comment: adding an
// exemption means choosing a class, and a case fitting neither has nowhere to
// go.
//
// Both lists carry no grandfather entries. That is what makes the retirement
// real rather than incidental — an allowlist admitting "this one already
// existed" would let the corpus survive with every acceptance criterion green.
//
// D-0070's other exception, the cross-*document* relationship check, appears in
// neither list because it needs no exemption: it draws its needle from a second
// document, so the rule does not fire on it at all.

// triggerPhraseExemptions cover the phrasings that decide whether an assistant
// reaches for a skill at all — those in a skill's `## When to use` section and
// its `description:` frontmatter.
//
// The evidence for keeping the class is behavioural: session mining measured
// the deployer agent at approximately zero dispatches before these phrasings
// existed. The limit is worth stating, because it bounds what the exception is
// worth — nothing mechanical consumes a trigger phrase, so the property rests
// on an assistant's judgment. The class is kept on the strength of that
// evidence, not the soundness of the mechanism.
//
// Adding an entry means asserting that the phrases decide dispatch. An
// assertion merely sitting in one of those two sections does not qualify —
// exempting by location was measured and rejected, because it would have
// covered nine further assertions that bear on nothing.
var triggerPhraseExemptions = map[string]string{
	"TestDeployerCard_FrontmatterDescriptionNamesReleaseTriggers": "the deployer agent card's `description:` trigger phrases; an assistant picks a subagent from the agent listing before any skill body is read",
	"TestAiwfxRelease_FrontmatterDescriptionRoutesToDeployer":     "aiwfx-release's `description:` trigger phrases, read from the skill listing before the body",
	"TestAiwfxRelease_DelegatesToDeployerAgent":                   "aiwfx-release's `## When to use` trigger phrases",
	"TestAiwfxHandoff_SkillScaffolded":                            "aiwfx-handoff's `description:` on-request trigger phrases; the skill fires mid-conversation on these",
	"TestAiwfxWhiteboard_AC2_DescriptionPhrasings":                "aiwfx-whiteboard's `description:` trigger phrases",
}

// derivedExpectationExemptions cover relationship checks that reach their
// second artefact through code rather than a second file. The expectation is
// computed by running the thing under test, so the check fails when either side
// moves — which is the reach that earns the cross-document class its place, and
// is not the defect D-0070 retires. A phrase assertion pins a reading that
// drifts on rewording; here the only literal is a stable identifier, and
// rewording the surrounding prose changes nothing.
//
// Restating the expectation as a literal forfeits the exemption: the entry is
// earned by deriving it, not by the subject matter.
var derivedExpectationExemptions = map[string]string{
	"TestPolicy_SkillBodyIDRowMatchesEmittedSeverity": "derives which findings table to expect from the severity the rule actually emits, so the check fails whether the skill's table or the rule's severity moves",
	"TestPolicy_AuthorizeSkillDocumentsItsOwnSurface": "derives the expected flag set from the Cobra tree, so the check fails whether a flag is added to the verb or dropped from the skill; the needle is a flag name the CLI already commits to, not a phrasing",
}

// shippedProseAssertionExempt reports whether a test function is allowlisted.
func shippedProseAssertionExempt(fn string) bool {
	if _, ok := triggerPhraseExemptions[fn]; ok {
		return true
	}
	_, ok := derivedExpectationExemptions[fn]
	return ok
}
