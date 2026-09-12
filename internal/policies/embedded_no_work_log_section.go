package policies

import "regexp"

// workLogBans are the two rules, in the order runShippedBan applies
// them.
var workLogBans = []lineBan{
	{
		// The section named as a section: a heading, a backticked heading,
		// or the JSON body key `aiwf show` derives from it. No sentence
		// about a consumer's own habits produces either shape, so the rule
		// takes no escape — naming the section as a section is the
		// reintroduction whatever else the line says.
		pattern: regexp.MustCompile(`(?i)##\s+work[ _-]log\b|work_log\b`),
		detail:  "a shipped surface names the retired `## Work log` section of the milestone spec; what it held is `acs[]`, the TDD phase ladder, and `aiwf history M-NNNN/AC-<N>`, which lists the implementation commit by its entity trailer",
	},
	{
		// Any other mention, which must be conditional on the reader's own
		// project. A skill asking whether a project keeps its own work log
		// alongside a diff passes; an instruction to fill one in an aiwf
		// milestone spec does not. The escape is the conditional rather
		// than a bare mention of a project, because shipped prose is
		// consumer-scoped throughout and names one constantly — an
		// instruction reintroducing the section reads naturally as "append
		// a work log entry to the project's spec", which the bare form
		// would clear.
		pattern: regexp.MustCompile(`(?i)work[ _-]log\b`),
		escape:  "if the project",
		detail:  "\"work log\" mention that is not conditional on the reader's own project — the milestone spec's `## Work log` section is retired, so a mention here reads as an instruction to fill one; a sentence about a consumer's own habit opens \"if the project\"",
	},
}

// PolicyEmbeddedNoWorkLogSection asserts that no surface aiwf ships names a
// `## Work log` section of the milestone spec. The section is retired: what it
// held is `acs[]`, the TDD phase ladder, and `aiwf history M-NNNN/AC-<N>`,
// which lists the implementation commit by its entity trailer.
//
// The ban is what makes the retirement hold. Removing the section from the
// template, the rituals and the agent cards leaves nothing asserting it stays
// removed — an exact revert of that removal passes every other check in this
// repo, and three single-line edits each restore the convention on their own: a
// heading placed above the template's ownership map, an instruction bullet in
// either milestone ritual, or any mention in the engineering-skill tree the
// section-ownership policies do not scan.
//
// The mirror of PolicyEmbeddedRitualsNoRetiredTrackingDoc, which bans the
// retired v1 tracking-doc convention on the same reasoning: a retired
// convention an agent can still read somewhere is one it will still follow, and
// which convention it follows depends on which artifact it happened to open.
// The two share their walk, their rule shape and their scope through
// runShippedBan; what is theirs alone is the rule table above.
func PolicyEmbeddedNoWorkLogSection(root string) ([]Violation, error) {
	return runShippedBan(root, workLogBans, func() Violation {
		return Violation{Policy: "embedded-no-work-log-section"}
	})
}
