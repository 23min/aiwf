package check

import (
	"fmt"
	"strings"

	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
	"github.com/23min/ai-workflow-v2/tools/internal/scope"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
)

// Provenance finding codes — the I2.5 standing rules from
// docs/pocv3/design/provenance-model.md §"`aiwf check` rules". Each
// fires on commit-history audit, not on tree state.
const (
	CodeProvenanceTrailerIncoherent       = "provenance-trailer-incoherent"
	CodeProvenanceForceNonHuman           = "provenance-force-non-human"
	CodeProvenanceActorMalformed          = "provenance-actor-malformed"
	CodeProvenancePrincipalNonHuman       = "provenance-principal-non-human"
	CodeProvenanceOnBehalfOfNonHuman      = "provenance-on-behalf-of-non-human"
	CodeProvenanceAuthorizedByMalformed   = "provenance-authorized-by-malformed"
	CodeProvenanceAuthorizationMissing    = "provenance-authorization-missing"
	CodeProvenanceAuthorizationOutOfScope = "provenance-authorization-out-of-scope"
	CodeProvenanceAuthorizationEnded      = "provenance-authorization-ended"
	CodeProvenanceNoActiveScope           = "provenance-no-active-scope"
	CodeProvenanceAuditOnlyNonHuman       = "provenance-audit-only-non-human"
)

// RunProvenance returns provenance findings for the given commit
// history. commits must be ordered oldest-first (`git log --reverse`)
// and should already be filtered to those carrying any `aiwf-*`
// trailer; pre-aiwf commits are silently skipped by the per-rule
// checks anyway, but pre-filtering keeps the work proportional.
//
// t is the current entity tree, consulted by the
// authorization-out-of-scope rule for reference reachability.
//
// The function is pure (no I/O, no git subprocess); the caller
// (cmd/aiwf) gathers commits via gitops and hands them in.
func RunProvenance(commits []scope.Commit, t *tree.Tree) []Finding {
	authIndex := buildAuthOpenerIndex(commits)
	chronoIdx := make(map[string]int, len(commits))
	for i, c := range commits {
		chronoIdx[c.SHA] = i
	}
	endedAt := buildEndedAtIndex(commits, chronoIdx)
	renameChain := buildRenameChain(commits)

	var findings []Finding
	for i := range commits {
		c := &commits[i]
		idx := indexCommitTrailersForProvenance(c.Trailers)
		findings = append(findings, provenanceShapeFindings(c, idx)...)
		findings = append(findings, provenanceCoherenceFindings(c, idx)...)
		findings = append(findings, provenanceAuthorizationFindings(c, idx, authIndex, endedAt, chronoIdx, renameChain, t)...)
	}
	return findings
}

// provenanceShapeFindings runs the per-trailer shape rules: actor,
// principal, on-behalf-of, authorized-by, plus the human-only
// constraints on aiwf-force / aiwf-audit-only.
func provenanceShapeFindings(c *scope.Commit, idx map[string]string) []Finding {
	var findings []Finding
	actor := idx[gitops.TrailerActor]
	if actor != "" && !roleIDOK(actor) {
		findings = append(findings, Finding{
			Code:     CodeProvenanceActorMalformed,
			Severity: SeverityError,
			Message:  fmt.Sprintf("commit %s: aiwf-actor: %q does not match <role>/<id>", short(c.SHA), actor),
			EntityID: idx[gitops.TrailerEntity],
		})
	}
	if v := idx[gitops.TrailerPrincipal]; v != "" && !isHumanRoleID(v) {
		findings = append(findings, Finding{
			Code:     CodeProvenancePrincipalNonHuman,
			Severity: SeverityError,
			Message:  fmt.Sprintf("commit %s: aiwf-principal: %q must be human/<id>", short(c.SHA), v),
			EntityID: idx[gitops.TrailerEntity],
		})
	}
	if v := idx[gitops.TrailerOnBehalfOf]; v != "" && !isHumanRoleID(v) {
		findings = append(findings, Finding{
			Code:     CodeProvenanceOnBehalfOfNonHuman,
			Severity: SeverityError,
			Message:  fmt.Sprintf("commit %s: aiwf-on-behalf-of: %q must be human/<id>", short(c.SHA), v),
			EntityID: idx[gitops.TrailerEntity],
		})
	}
	if v := idx[gitops.TrailerAuthorizedBy]; v != "" && !shaOK(v) {
		findings = append(findings, Finding{
			Code:     CodeProvenanceAuthorizedByMalformed,
			Severity: SeverityError,
			Message:  fmt.Sprintf("commit %s: aiwf-authorized-by: %q must be 7–40 hex", short(c.SHA), v),
			EntityID: idx[gitops.TrailerEntity],
		})
	}
	if _, hasForce := idx[gitops.TrailerForce]; hasForce && actor != "" && !strings.HasPrefix(actor, "human/") {
		findings = append(findings, Finding{
			Code:     CodeProvenanceForceNonHuman,
			Severity: SeverityError,
			Message:  fmt.Sprintf("commit %s: aiwf-force: requires aiwf-actor: human/... (got %q)", short(c.SHA), actor),
			EntityID: idx[gitops.TrailerEntity],
		})
	}
	if _, hasAudit := idx[gitops.TrailerAuditOnly]; hasAudit && actor != "" && !strings.HasPrefix(actor, "human/") {
		findings = append(findings, Finding{
			Code:     CodeProvenanceAuditOnlyNonHuman,
			Severity: SeverityError,
			Message:  fmt.Sprintf("commit %s: aiwf-audit-only: requires aiwf-actor: human/... (got %q)", short(c.SHA), actor),
			EntityID: idx[gitops.TrailerEntity],
		})
	}
	return findings
}

// provenanceCoherenceFindings runs the required-together / mutually-
// exclusive rules from provenance-model.md §"Trailer set". Each
// violation surfaces as a single `provenance-trailer-incoherent`
// finding whose message names the specific pair.
//
// Pre-aiwf commits (no aiwf-actor: trailer) are skipped: the
// required-together rule on (principal, non-human actor) would
// otherwise fire on every untrailered commit.
func provenanceCoherenceFindings(c *scope.Commit, idx map[string]string) []Finding {
	actor := idx[gitops.TrailerActor]
	if actor == "" {
		return nil
	}
	_, hasPrincipal := idx[gitops.TrailerPrincipal]
	_, hasOnBehalfOf := idx[gitops.TrailerOnBehalfOf]
	_, hasAuthorizedBy := idx[gitops.TrailerAuthorizedBy]
	_, hasForce := idx[gitops.TrailerForce]

	actorIsHuman := strings.HasPrefix(actor, "human/")
	var findings []Finding

	emit := func(reason string) {
		findings = append(findings, Finding{
			Code:     CodeProvenanceTrailerIncoherent,
			Severity: SeverityError,
			Message:  fmt.Sprintf("commit %s: %s", short(c.SHA), reason),
			EntityID: idx[gitops.TrailerEntity],
		})
	}

	// Required-together: (on-behalf-of, authorized-by).
	switch {
	case hasOnBehalfOf && !hasAuthorizedBy:
		emit("aiwf-on-behalf-of: present without aiwf-authorized-by:")
	case hasAuthorizedBy && !hasOnBehalfOf:
		emit("aiwf-authorized-by: present without aiwf-on-behalf-of:")
	}

	// Required-together: (principal, non-human actor). Symmetric: a
	// non-human actor without principal is unaccountable.
	if !actorIsHuman && !hasPrincipal {
		emit(fmt.Sprintf("aiwf-actor: %q is non-human but aiwf-principal: is missing", actor))
	}

	// Mutually exclusive: (force, on-behalf-of); (principal, human
	// actor); (on-behalf-of, human actor). Force-non-human is reported
	// separately by provenanceShapeFindings; we still report the
	// force+on-behalf-of conflict as incoherent because it's a
	// distinct rule.
	if hasForce && hasOnBehalfOf {
		emit("aiwf-force: and aiwf-on-behalf-of: are mutually exclusive (force is human-only)")
	}
	if actorIsHuman && hasPrincipal {
		emit("aiwf-principal: is forbidden when aiwf-actor: is human/... (humans act directly)")
	}
	if actorIsHuman && hasOnBehalfOf {
		emit("aiwf-on-behalf-of: is forbidden when aiwf-actor: is human/... (humans act directly)")
	}
	return findings
}

// provenanceAuthorizationFindings runs the cross-commit authorization
// rules: -missing / -out-of-scope / -ended for commits carrying
// aiwf-authorized-by, plus -no-active-scope for ai/... actors with no
// authorization.
//
// authIndex maps known authorize-opener SHAs to their opener commit;
// endedAt maps each opener SHA to the chrono index of its first
// scope-end (or absence); chronoIdx is the position-by-SHA lookup.
func provenanceAuthorizationFindings(
	c *scope.Commit,
	idx map[string]string,
	authIndex map[string]*scope.Commit,
	endedAt map[string]int,
	chronoIdx map[string]int,
	renameChain map[string]string,
	t *tree.Tree,
) []Finding {
	actor := idx[gitops.TrailerActor]
	authSHA := idx[gitops.TrailerAuthorizedBy]
	_, hasOnBehalfOf := idx[gitops.TrailerOnBehalfOf]

	var findings []Finding

	// -no-active-scope: ai/... actor with no on-behalf-of trailer.
	// Skipped on aiwf-verb: authorize commits — those don't operate
	// inside a scope (the scope is what they open). The verb gate
	// already refuses non-human authorizers; this surfaces the same
	// constraint on hand-edited history.
	if strings.HasPrefix(actor, "ai/") && !hasOnBehalfOf && idx[gitops.TrailerVerb] != "authorize" {
		findings = append(findings, Finding{
			Code:     CodeProvenanceNoActiveScope,
			Severity: SeverityError,
			Message:  fmt.Sprintf("commit %s: aiwf-actor: %q has no aiwf-on-behalf-of: (no active scope authorized this act)", short(c.SHA), actor),
			EntityID: idx[gitops.TrailerEntity],
		})
	}

	if authSHA == "" || !shaOK(authSHA) {
		// Shape rule already reported malformed SHAs; cross-commit
		// rules only run on a well-shaped reference.
		return findings
	}

	opener, ok := resolveAuthSHA(authSHA, authIndex)
	if !ok {
		findings = append(findings, Finding{
			Code:     CodeProvenanceAuthorizationMissing,
			Severity: SeverityError,
			Message:  fmt.Sprintf("commit %s: aiwf-authorized-by: %s does not resolve to an authorize/opened commit", short(c.SHA), short(authSHA)),
			EntityID: idx[gitops.TrailerEntity],
		})
		return findings
	}

	// -authorization-ended: ended at or before this commit's chrono
	// position. Equality is the auto-end edge case (a terminal-promote
	// commit ends its own scope and references it via aiwf-authorized-
	// by); the act is allowed because the scope was active when this
	// commit landed. Strict less-than fires only when an earlier
	// commit already ended the scope.
	if endIdx, ended := endedAt[opener.SHA]; ended {
		thisIdx, ok := chronoIdx[c.SHA]
		if !ok {
			thisIdx = -1
		}
		if endIdx < thisIdx {
			findings = append(findings, Finding{
				Code:     CodeProvenanceAuthorizationEnded,
				Severity: SeverityError,
				Message:  fmt.Sprintf("commit %s: aiwf-authorized-by: %s references a scope that was already ended", short(c.SHA), short(authSHA)),
				EntityID: idx[gitops.TrailerEntity],
			})
		}
	}

	// -authorization-out-of-scope: the verb's target entity must reach
	// the scope-entity through forward refs (parent / depends_on /
	// addressed_by / etc.). Reachability is forward from target to
	// scope, not the inverse — see verb.scopeAllowsAct for the same
	// rule. The scope-entity id is resolved through the rename chain
	// so a reallocated entity keeps its scopes valid.
	openerIdx := indexCommitTrailersForProvenance(opener.Trailers)
	scopeEntity := openerIdx[gitops.TrailerEntity]
	scopeEntity = walkRenameChain(scopeEntity, renameChain)
	target := idx[gitops.TrailerEntity]
	if scopeEntity != "" && target != "" && t != nil {
		from := compositeRoot(target)
		to := compositeRoot(scopeEntity)
		if from != to && !t.Reaches(from, to) {
			findings = append(findings, Finding{
				Code:     CodeProvenanceAuthorizationOutOfScope,
				Severity: SeverityError,
				Message: fmt.Sprintf("commit %s: aiwf-authorized-by: %s — target %s does not reach scope-entity %s",
					short(c.SHA), short(authSHA), target, scopeEntity),
				EntityID: target,
			})
		}
	}
	return findings
}

// buildAuthOpenerIndex maps every authorize-opener commit's SHA to a
// pointer into the supplied slice. Used by the cross-commit rules to
// resolve aiwf-authorized-by: SHAs.
//
// Pause/resume authorize commits are NOT in the index — only openers
// (aiwf-scope: opened) qualify. A reference to a pause/resume SHA
// surfaces as -authorization-missing.
func buildAuthOpenerIndex(commits []scope.Commit) map[string]*scope.Commit {
	out := make(map[string]*scope.Commit)
	for i := range commits {
		c := &commits[i]
		idx := indexCommitTrailersForProvenance(c.Trailers)
		if idx[gitops.TrailerVerb] == "authorize" && idx[gitops.TrailerScope] == "opened" {
			out[c.SHA] = c
		}
	}
	return out
}

// buildEndedAtIndex maps each authorize-opener SHA to the chrono
// position of the first commit carrying `aiwf-scope-ends: <auth-sha>`.
// Openers without an end commit are absent.
func buildEndedAtIndex(commits []scope.Commit, chronoIdx map[string]int) map[string]int {
	out := map[string]int{}
	for i := range commits {
		c := &commits[i]
		for _, tr := range c.Trailers {
			if tr.Key != gitops.TrailerScopeEnds {
				continue
			}
			if _, already := out[tr.Value]; already {
				continue
			}
			if pos, ok := chronoIdx[c.SHA]; ok {
				out[tr.Value] = pos
			}
		}
	}
	return out
}

// buildRenameChain maps prior-entity ids to the new id assigned by
// the most recent reallocate. Cycles (defensive — corrupted history)
// are broken by visit-once.
func buildRenameChain(commits []scope.Commit) map[string]string {
	out := map[string]string{}
	for i := range commits {
		c := &commits[i]
		idx := indexCommitTrailersForProvenance(c.Trailers)
		prior := idx[gitops.TrailerPriorEntity]
		now := idx[gitops.TrailerEntity]
		if prior == "" || now == "" || prior == now {
			continue
		}
		out[prior] = now
	}
	return out
}

// walkRenameChain follows id forward through the rename map until it
// stops moving, with a visit-once cycle guard.
func walkRenameChain(id string, chain map[string]string) string {
	if id == "" {
		return id
	}
	visited := map[string]bool{id: true}
	current := id
	for {
		next, ok := chain[current]
		if !ok || visited[next] {
			return current
		}
		visited[next] = true
		current = next
	}
}

// resolveAuthSHA returns the opener commit for a (possibly short)
// authorized-by SHA. authIndex's keys are full SHAs as recorded in
// commit history; an exact match wins, otherwise we accept any unique
// prefix match (mirroring git's short-SHA resolution).
func resolveAuthSHA(authSHA string, authIndex map[string]*scope.Commit) (*scope.Commit, bool) {
	if c, ok := authIndex[authSHA]; ok {
		return c, true
	}
	var match *scope.Commit
	for full, c := range authIndex {
		if strings.HasPrefix(full, authSHA) {
			if match != nil {
				// Ambiguous prefix: treat as missing rather than
				// silently picking. Hand-edited histories with such
				// collisions are vanishingly rare; the hint will tell
				// the user to use the full SHA.
				return nil, false
			}
			match = c
		}
	}
	if match == nil {
		return nil, false
	}
	return match, true
}

// indexCommitTrailersForProvenance is a local key→value lookup for a
// commit's trailers. Repeating keys (notably aiwf-scope-ends) collapse
// to last-wins; provenance rules iterate the slice directly when they
// need every value.
func indexCommitTrailersForProvenance(trailers []gitops.Trailer) map[string]string {
	out := make(map[string]string, len(trailers))
	for _, tr := range trailers {
		out[tr.Key] = tr.Value
	}
	return out
}

// compositeRoot rolls a composite id (M-NNN/AC-N) up to its parent so
// reachability runs against the parent milestone. Plain ids pass
// through unchanged.
func compositeRoot(id string) string {
	if entity.IsCompositeID(id) {
		parent, _, _ := entity.ParseCompositeID(id)
		return parent
	}
	return id
}

// roleIDOK is the same regex gitops.ValidateTrailer uses, exposed for
// in-package shape checks (gitops keeps the regex unexported).
func roleIDOK(s string) bool {
	if s == "" {
		return false
	}
	if strings.ContainsAny(s, " \t\n") {
		return false
	}
	slash := strings.IndexByte(s, '/')
	if slash <= 0 || slash == len(s)-1 {
		return false
	}
	if strings.IndexByte(s[slash+1:], '/') >= 0 {
		return false
	}
	return true
}

// isHumanRoleID checks both the role/id shape and the human/ prefix.
func isHumanRoleID(s string) bool {
	return roleIDOK(s) && strings.HasPrefix(s, "human/")
}

// shaOK reports whether s is 7–40 lowercase hex (the shape git
// rev-parse emits and ValidateTrailer enforces at write time).
func shaOK(s string) bool {
	if len(s) < 7 || len(s) > 40 {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

// short truncates a SHA to 7 chars for human-readable messages,
// preserving full precision in the structured EntityID field.
func short(sha string) string {
	if len(sha) <= 7 {
		return sha
	}
	return sha[:7]
}
