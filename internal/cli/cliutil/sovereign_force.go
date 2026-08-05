package cliutil

import (
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/verb"
)

// RefuseNonHumanSovereignForce refuses a `--force` invocation by a
// non-human actor at the dispatcher layer, for the verbs whose flag
// unconditionally produces a sovereign force trailer.
//
// The rule is not reimplemented here. The check builds the minimal
// trailer set the flag implies and hands it to
// verb.CheckForceTrailerCoherence — the same function verb.Apply
// consults at the commit seam (ADR-0040). One implementation, read at
// two moments: the refusal an operator sees is the seam's own message
// and the seam stays authoritative, so this cannot drift into a second
// opinion about who may wield the flag.
//
// What it buys is the moment. Apply refuses before writing, but a verb
// reaches Apply only after the repo lock is taken and the tree loaded;
// this runs immediately after the prelude resolves the actor, so a
// refusal costs neither.
//
// Only force-non-human can fire on this set, and that is the whole
// intent. The other two force-predicated rules are about trailers the
// dispatcher has not assembled — on-behalf-of is appended later by the
// provenance-decoration layer, audit-only by the verb — so judging them
// here would be judging an incomplete set.
//
// Callers whose `--force` is conditional must not use this. `aiwf add`
// stamps the trailer only when the flag actually bypassed the
// born-complete body gate, so a check keyed on the flag would refuse
// invocations that emit no force trailer at all and that the kernel
// permits today; its guard is the seam, where the trailer's presence is
// the fact being judged.
func RefuseNonHumanSovereignForce(label, actor string, force bool) (int, bool) {
	if !force {
		return ExitOK, true
	}
	// The reason's text is irrelevant to every rule in the set; a
	// non-empty placeholder keeps the trailer well-formed without
	// implying the dispatcher's own --reason has been validated yet.
	trailers := []gitops.Trailer{
		{Key: gitops.TrailerActor, Value: actor},
		{Key: gitops.TrailerForce, Value: "precheck"},
	}
	if err := verb.CheckForceTrailerCoherence(trailers); err != nil {
		Errorf("%s: %v\n", label, err)
		// A coherence violation is a legality refusal, not an internal
		// failure — the same exit class the seam uses for the same act
		// (ADR-0040), so one consumer routes on a denial without
		// needing to know which moment produced it.
		return ExitFindings, false
	}
	return ExitOK, true
}
