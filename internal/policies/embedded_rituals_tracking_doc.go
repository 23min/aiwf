package policies

import "regexp"

// trackingDocBans are the two rules, in the order runShippedBan applies
// them.
var trackingDocBans = []lineBan{
	{
		// The retired directory, named as a path. An agent following an
		// instruction that names it recreates it in the consumer repo and
		// nothing kernel-side objects — verified in G-0245, where a stray
		// file under `work/tracking/` produced zero `aiwf check` findings.
		//
		// Case-sensitive, alone among these rules: this one matches a path
		// an agent would copy verbatim, and the directory it names is
		// lower-case. The phrase rules below match prose, where a heading
		// or a sentence start legitimately varies the case.
		pattern: regexp.MustCompile(`work/tracking/`),
		detail:  "a shipped surface references the retired `work/tracking/` directory (G-0245); point at the milestone spec's frontmatter `acs[]`, `aiwf history`, or the spec's `## Decisions made during implementation` section instead",
	},
	{
		// Any mention of the convention, which must carry "v1" on the same
		// line — the mechanical definition of "explicit v1-historical
		// context". A retirement statement ("The v1 separate tracking doc
		// is gone") passes; instruction-shaped phrasing ("finalize the
		// tracking doc") does not.
		pattern: regexp.MustCompile(`(?i)tracking[ -]docs?\b`),
		escape:  "v1",
		detail:  "\"tracking doc\" mention without \"v1\" on the same line — the retired convention may only appear in explicit v1-historical context (G-0245)",
	},
}

// PolicyEmbeddedRitualsNoRetiredTrackingDoc asserts that no surface aiwf ships
// instructs the retired v1 separate tracking-doc convention.
//
// Why this exists: the vendored snapshot shipped self-contradictory at M-0148 —
// five artifacts instructed the retired convention while three others declared
// it gone. Which convention an agent followed depended on which artifact it
// happened to read: the "guarantee depends on the LLM's behavior" failure
// class. G-0224 was the same defect class at nit level; G-0245 is the
// recurrence that crossed the stated threshold for a mechanical chokepoint.
//
// That reasoning is about which artifact an agent opens, not about which tree
// the artifact sits in, so the scan covers every embedded tree rather than the
// ritual snapshot alone — a verb skill instructing the convention would be
// followed exactly as a ritual one would.
//
// The mirror of PolicyEmbeddedNoWorkLogSection, which bans the retired
// `## Work log` section of the milestone spec on the same reasoning. The two
// share their walk, their rule shape and their scope through
// runShippedBan; what is theirs alone is the rule table above.
//
// Pins G-0245 fix-shape item 2.
func PolicyEmbeddedRitualsNoRetiredTrackingDoc(root string) ([]Violation, error) {
	return runShippedBan(root, trackingDocBans, func() Violation {
		return Violation{Policy: "embedded-rituals-no-retired-tracking-doc"}
	})
}
