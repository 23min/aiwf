package check

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/scope"
	"github.com/23min/aiwf/internal/tree"
)

// squashMergeSubjectRE matches GitHub's default squash-merge
// commit subject pattern: any prose followed by ` (#NNN)` at end.
// Used to specialize the untrailered-entity-commit warning so the
// hint can point at the merge-strategy gotcha and the audit-only
// repair path (G31). False positives are commits whose subject
// genuinely ends with a parenthesised issue number; the bare
// warning would fire on them anyway, so the worst case is a
// slightly-too-specific hint, not a missed finding.
var squashMergeSubjectRE = regexp.MustCompile(`\s\(#\d+\)$`)

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

	// I2.5 step 7b: pre-push trailer audit (G24). Surfaces the
	// audit-trail hole when a manual `git commit` lands on entity
	// files without an aiwf-verb: trailer. Warning, not error: the
	// user's intended response is `aiwf <verb> --audit-only --reason
	// "..."` which fills the hole without rewriting history.
	CodeProvenanceUntrailedEntityCommit = "provenance-untrailered-entity-commit"

	// Companion to the above: emitted once when the untrailered-
	// audit scope cannot be determined (the branch has no upstream
	// and the operator passed no `--since <ref>`). The audit is
	// skipped — scanning all of HEAD on a long-lived branch with
	// many merges from trunk produces a flood of warnings against
	// commits that are someone else's responsibility. The hint
	// names the two opt-in paths (configure upstream, or pass
	// --since) so the operator can re-enable a deliberate scan.
	CodeProvenanceUntrailedScopeUndefined = "provenance-untrailered-scope-undefined"
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
	endedBy := buildEndedByIndex(commits)
	renameChain := buildRenameChain(commits)

	var findings []Finding
	for i := range commits {
		c := &commits[i]
		idx := indexCommitTrailersForProvenance(c.Trailers)
		findings = append(findings, provenanceShapeFindings(c, idx)...)
		findings = append(findings, provenanceCoherenceFindings(c, idx)...)
		findings = append(findings, provenanceAuthorizationFindings(c, idx, authIndex, endedAt, endedBy, chronoIdx, renameChain, t)...)
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
		emit(fmt.Sprintf("actor %q is non-human but aiwf-principal: is missing", actor))
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
// scope-end; endedBy maps each opener SHA to the commit that emitted
// the scope-ends trailer; chronoIdx is the position-by-SHA lookup.
func provenanceAuthorizationFindings(
	c *scope.Commit,
	idx map[string]string,
	authIndex map[string]*scope.Commit,
	endedAt map[string]int,
	endedBy map[string]*scope.Commit,
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
	//
	// G-0120 exception: a wrap-bundle commit (wrap-epic / wrap-
	// milestone) that lands after a same-entity terminal-promote ended
	// the scope is treated as still within scope — wraps are atomic
	// across commit boundaries even when the ritual order put the
	// promote first (see G-0119 for the forward fix). The exception is
	// narrow on purpose: same entity, wrap verb, terminating verb is
	// `promote`. See isWrapBundleCommit for the full contract.
	if endIdx, ended := endedAt[opener.SHA]; ended {
		thisIdx, ok := chronoIdx[c.SHA]
		if !ok {
			thisIdx = -1
		}
		if endIdx < thisIdx && !isWrapBundleCommit(idx, opener, endedBy[opener.SHA], renameChain, t) {
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
	scopeEntity = resolveViaPriorIDs(scopeEntity, t)
	target := idx[gitops.TrailerEntity]
	if scopeEntity != "" && target != "" && t != nil {
		from := resolveViaPriorIDs(compositeRoot(target), t)
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

// resolveViaPriorIDs maps a possibly-old id to the current id of the
// entity whose `prior_ids:` frontmatter lists it. This is the
// tree-side companion to walkRenameChain (which follows
// aiwf-prior-entity commit trailers): together they cover the case
// where a reallocate happened before the audit window so the rename
// trailer isn't in scope but the renumbered entity's frontmatter still
// witnesses the lineage (G-0118).
//
// When id resolves directly to a live entity, that entity is
// authoritative — the bare lookup wins. Only when the id is absent
// from current state, OR when a different entity reclaims the id
// post-rename (the parallel-allocation collision case in G-0118), do
// we fall through to ByPriorID. In the collision case ByID returns
// the wrong entity, so we always check ByPriorID and prefer it when
// it returns a distinct entity that actually claims this id as a
// prior.
//
// Returns id unchanged when t is nil, when id is empty, when no
// entity claims id via prior_ids, or when the prior-id lookup would
// land on the same entity already returned by ByID.
//
// Single-hop by design: each `aiwf reallocate` appends exactly one
// predecessor to the renumbered entity's prior_ids list, so one
// lookup covers the typical rename chain. If double-rename chains
// ever surface as friction the right fix is to accumulate prior_ids
// transitively at write time inside the reallocate verb — not to
// turn this reader into a multi-hop walker. The returned value is
// `prior.ID` verbatim (on-disk width), so callers must canonicalize
// before string-comparing; `Tree.Reaches` already does.
func resolveViaPriorIDs(id string, t *tree.Tree) string {
	if id == "" || t == nil {
		return id
	}
	prior := t.ByPriorID(id)
	if prior == nil {
		return id
	}
	// Prefer the prior-ids match. If ByID returned the same entity,
	// this is a no-op; if ByID returned a different (parallel-
	// allocation collision) entity, the prior-ids match is the
	// renumbered-forward entity G-0118 needs to reach.
	return prior.ID
}

// UntrailedCommit is the input shape for RunUntrailedAudit: the
// commit's SHA, its subject (first line of the message), its
// trailer set, and the relative paths it touched (as reported by
// `git diff-tree`).
//
// The Subject is consulted to specialize the warning when the
// commit looks like a GitHub squash-merge ("…(#NNN)" suffix) —
// see G31. It can be left empty by callers that don't need that
// specialization; the bare warning still fires.
type UntrailedCommit struct {
	SHA      string
	Subject  string
	Trailers []gitops.Trailer
	Paths    []string
}

// RunUntrailedAudit returns
// `provenance-untrailered-entity-commit` findings — one per
// (commit, entity) pair — for every untrailered commit in the
// supplied slice that touched an entity file. The caller is
// expected to scope `commits` to the unpushed range (typically
// `@{u}..HEAD`) so already-pushed pre-aiwf history is silently
// ignored.
//
// One finding per entity, not per commit: a manual commit that
// touches three entity files emits three findings, each tagged
// with the entity id. This is what makes per-entity audit-only
// suppression work — `aiwf <verb> M-001 --audit-only` clears the
// M-001 finding without affecting the others on the same commit.
// Each finding's message names exactly one entity, which keeps
// individual lines short even when commits touch many entity
// files (matters for squash, merge, and bulk-import commits).
//
// The finding is a WARNING. The intended user response is `aiwf
// <verb> --audit-only --reason "..."` (step 5b), which records the
// transition without rewriting history; an error severity here would
// block pushes for state that is already correct.
//
// Coverage by audit-only: when a later commit in the same range
// carries `aiwf-audit-only:` and its `aiwf-entity:` matches the
// entity id, the warning for that (commit, entity) pair is
// suppressed. Composite ids on audit-only commits roll up to the
// parent milestone for matching, mirroring how composite ids on
// manual commits resolve to the parent file.
//
// Defensive fallback: if a commit touched paths that PathKind
// recognizes but IDFromPath cannot parse to an id (a bug in the
// path scheme would be the only realistic cause), one
// path-tagged finding fires per such path. EntityID is empty in
// that branch.
func RunUntrailedAudit(commits []UntrailedCommit) []Finding {
	// Build entityID → latest chrono index of an audit-only commit
	// that backfills it. Composite ids roll up to the parent so the
	// match works against manual commits that touch the parent file.
	// Keys are canonicalized so a narrow legacy trailer
	// (`aiwf-entity: G-001`) covers a manual commit that touched the
	// canonical-shape path (`G-0001-leak.md`) and vice versa
	// (AC-2/AC-4 in M-081).
	auditAt := map[string]int{}
	for i := range commits {
		idx := indexCommitTrailersForProvenance(commits[i].Trailers)
		if strings.TrimSpace(idx[gitops.TrailerAuditOnly]) == "" {
			continue
		}
		entID := strings.TrimSpace(idx[gitops.TrailerEntity])
		if entID == "" {
			continue
		}
		entID = entity.Canonicalize(compositeRoot(entID))
		auditAt[entID] = i
	}

	var findings []Finding
	for i := range commits {
		c := &commits[i]
		idx := indexCommitTrailersForProvenance(c.Trailers)
		if idx[gitops.TrailerVerb] != "" {
			continue
		}
		var unresolvedPaths []string
		idSeen := map[string]bool{}
		var touchedIDs []string
		for _, p := range c.Paths {
			kind, ok := entity.PathKind(p)
			if !ok {
				continue
			}
			id, idOK := entity.IDFromPath(p, kind)
			if !idOK {
				unresolvedPaths = append(unresolvedPaths, p)
				continue
			}
			if idSeen[id] {
				continue
			}
			idSeen[id] = true
			touchedIDs = append(touchedIDs, id)
		}
		// Specialize when the commit looks like a GitHub squash-
		// merge — its subject ends with " (#NNN)". Source-commit
		// trailers were dropped by the GitHub UI; the hint should
		// name that explicitly so operators don't think it's a
		// hand-edit they can fix with the same recipe (G31).
		subcode := ""
		if squashMergeSubjectRE.MatchString(c.Subject) {
			subcode = "squash-merge"
		}
		for _, id := range touchedIDs {
			if isEntityCoveredByLaterAudit(id, i, auditAt) {
				continue
			}
			canonID := entity.Canonicalize(id)
			findings = append(findings, Finding{
				Code:     CodeProvenanceUntrailedEntityCommit,
				Subcode:  subcode,
				Severity: SeverityWarning,
				EntityID: canonID,
				Message: fmt.Sprintf("commit %s touched %s with no aiwf-verb: trailer",
					short(c.SHA), canonID),
			})
		}
		// Defensive: PathKind matched but IDFromPath did not.
		// Flag each such path once so the gap is visible without
		// flooding the operator's screen.
		for _, p := range unresolvedPaths {
			findings = append(findings, Finding{
				Code:     CodeProvenanceUntrailedEntityCommit,
				Severity: SeverityWarning,
				Message: fmt.Sprintf("commit %s touched entity-shaped path %q with no resolvable id and no aiwf-verb: trailer",
					short(c.SHA), p),
			})
		}
	}
	return findings
}

// isEntityCoveredByLaterAudit reports whether id has a strictly-
// later audit-only commit recorded in auditAt. Used by
// RunUntrailedAudit to suppress the warning per (commit, entity)
// pair once the operator has backfilled that entity's audit trail
// with `aiwf <verb> --audit-only`.
//
// Canonicalizes id before lookup so a query at one width matches a
// stored audit-only at another (AC-2 in M-081).
func isEntityCoveredByLaterAudit(id string, manualIdx int, auditAt map[string]int) bool {
	laterIdx, ok := auditAt[entity.Canonicalize(compositeRoot(id))]
	return ok && laterIdx > manualIdx
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

// buildEndedByIndex maps each authorize-opener SHA to the commit that
// emitted the first `aiwf-scope-ends: <auth-sha>` trailer for it.
// Openers without an end commit are absent. The companion of
// buildEndedAtIndex: that one carries chrono position for the
// -authorization-ended comparison; this one carries the end-commit
// pointer so the wrap-bundle exception (G-0120) can inspect the
// terminating commit's verb and entity.
func buildEndedByIndex(commits []scope.Commit) map[string]*scope.Commit {
	out := map[string]*scope.Commit{}
	for i := range commits {
		c := &commits[i]
		for _, tr := range c.Trailers {
			if tr.Key != gitops.TrailerScopeEnds {
				continue
			}
			if _, already := out[tr.Value]; already {
				continue
			}
			out[tr.Value] = c
		}
	}
	return out
}

// wrapBundleVerbs is the closed set of `aiwf-verb:` values that
// participate in a wrap bundle — the post-promote commit window the
// rituals plugin's `aiwfx-wrap-epic` / `aiwfx-wrap-milestone` skills
// produce (wrap.md artefact, CHANGELOG entries, merge trailers). New
// wrap-related verbs must be added here explicitly; the empty / non-
// matching case keeps the exception narrow per CLAUDE.md
// §"Engineering principles" YAGNI.
var wrapBundleVerbs = map[string]bool{
	"wrap-epic":      true,
	"wrap-milestone": true,
}

// isWrapBundleCommit reports whether the commit described by `idx` (its
// trailer map) is a wrap-bundle commit whose scope was terminated by a
// same-entity terminal-promote — the legitimate post-promote pattern
// G-0119's ritual order produces and G-0120 makes audit-tolerant.
//
// Recognition criteria (all must hold):
//   - the commit's aiwf-verb is in wrapBundleVerbs;
//   - the scope was terminated by a known end-commit (endedBy
//     supplies it);
//   - the end-commit's aiwf-verb is `promote` (the canonical
//     terminating step; a scope ended by `revoke` or another verb
//     does NOT enable the window — those terminations were
//     deliberate operator acts outside the wrap path);
//   - the end-commit's aiwf-entity matches this commit's aiwf-entity
//     (after composite-root rollup, rename-chain walk, and prior_ids
//     resolution — same lineage of equivalence the out-of-scope rule
//     uses for the target/scope reachability check).
//
// The same-entity narrowing avoids silently broadening the exception
// to any reachable descendant: a wrap commit on M-0001 under a scope
// ended on the parent epic E-0001 still fires authorization-ended
// because the scope-terminator was a different entity. The forward fix
// (G-0119) keeps the promote at the end of the wrap bundle; this
// exception only forgives the specific post-promote-on-same-entity
// pattern produced by the pre-G-0119 ritual order.
//
// Returns false when any input is missing or any criterion fails —
// the caller's default behavior (firing the finding) is the safe
// fallback.
func isWrapBundleCommit(
	idx map[string]string,
	opener *scope.Commit,
	endCommit *scope.Commit,
	renameChain map[string]string,
	t *tree.Tree,
) bool {
	if endCommit == nil {
		return false
	}
	if !wrapBundleVerbs[idx[gitops.TrailerVerb]] {
		return false
	}
	endIdx := indexCommitTrailersForProvenance(endCommit.Trailers)
	if endIdx[gitops.TrailerVerb] != "promote" {
		return false
	}
	wrapEntity := idx[gitops.TrailerEntity]
	endEntity := endIdx[gitops.TrailerEntity]
	if wrapEntity == "" || endEntity == "" {
		return false
	}
	// Resolve both sides through the same lineage helpers the out-of-
	// scope rule uses, so a reallocate across the wrap window doesn't
	// defeat the same-entity match.
	wrapRoot := resolveViaPriorIDs(walkRenameChain(compositeRoot(wrapEntity), renameChain), t)
	endRoot := resolveViaPriorIDs(walkRenameChain(compositeRoot(endEntity), renameChain), t)
	if wrapRoot != endRoot {
		return false
	}
	// Defense-in-depth: the opener's entity should also match (a wrap
	// referencing an opener on an unrelated entity but somehow
	// terminated on the wrap's entity is pathological — the
	// authorization-out-of-scope rule already catches it, but a quick
	// check here keeps the exception's contract crisp).
	if opener != nil {
		openerEnt := indexCommitTrailersForProvenance(opener.Trailers)[gitops.TrailerEntity]
		if openerEnt != "" {
			openerRoot := resolveViaPriorIDs(walkRenameChain(compositeRoot(openerEnt), renameChain), t)
			if openerRoot != wrapRoot {
				return false
			}
		}
	}
	return true
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
