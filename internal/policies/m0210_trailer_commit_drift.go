package policies

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PolicyM0210TrailerCommitDrift asserts that the trailered-commit /
// trailered-merge prescription duplicated across aiwfx-wrap-epic and
// aiwfx-wrap-milestone is protected against drift — the reframed M-0210
// deliverable. M-0210 originally proposed extracting the block into a
// wf-commit-trailers reference skill (ADR-0024); that was rejected in favour
// of this chokepoint, which meets the epic's drift-safety goal without a new
// reference-skill category or a mid-merge skill-invocation step. The block
// stays inline in each wrap ritual (the command is visible where it is run);
// this policy is what keeps the copies from drifting.
//
// The prescription is the block that composes an aiwf-trailered commit: a
// `git commit --trailer` invocation naming the three kernel-required trailer
// keys (aiwf-verb / aiwf-entity / aiwf-actor), the canonical variant-casings
// caveat ("variant casings … fail the kernel's trailer-keys policy"), and —
// for a trailered merge — the "resolve identity from git config user.email; do
// not hardcode" note.
//
// Two facets:
//
//   - AC-1 (presence guard): the required rituals — aiwfx-wrap-epic and
//     aiwfx-wrap-milestone — each carry a trailered-commit block naming all
//     three keys. Catches the G-0219 failure mode: a wrap ritual whose
//     trailered-commit prescription is absent or asymmetric with its sibling's.
//   - AC-2 (accompaniment guard): for every embedded ritual that composes a
//     trailered commit, the canonical variant-casings caveat is present; for
//     every ritual that stages a --no-commit merge and then composes a
//     trailered commit, the identity-resolution rule is present. This is the
//     single-source-by-policy property — a reword that drops the canonical
//     caveat or identity note, in any ritual, fails CI.
//
// The guard is per-ritual (file-level), not per-`git commit --trailer` site:
// it asserts the caveat / identity note appears somewhere in a ritual that
// carries a trailered-commit block, which matches the current bodies
// (aiwfx-wrap-epic states the caveat once, not at both of its trailer sites)
// and is sufficient for the G-0219 drift class. It pins presence of the
// canonical prescription, not byte-identity across sites.
//
// This is an aiwf-repo development invariant — the embedded-rituals tree
// exists only here — so it lives as a Go policy test, mirroring the sibling
// PolicyM0202* / PolicyM0132* ritual/devcontainer policies, not as an `aiwf
// check` finding (which would be inert in a consumer tree, where rituals are
// materialized rather than authored).
func PolicyM0210TrailerCommitDrift(root string) ([]Violation, error) {
	matches, err := filepath.Glob(filepath.Join(
		root, "internal", "skills", "embedded-rituals", "plugins", "*", "skills", "*", "SKILL.md"))
	if err != nil { //coverage:ignore unreachable: the glob pattern is a fixed literal (only `*`), never ErrBadPattern
		return nil, err
	}

	var vs []Violation
	report := func(rel, detail string) {
		vs = append(vs, Violation{
			Policy: "m0210-trailer-commit-drift",
			File:   rel,
			Detail: detail,
		})
	}

	// Ritual bodies keyed by skill directory name (e.g. "aiwfx-wrap-epic").
	type ritual struct {
		dir  string
		rel  string
		body string
	}
	found := map[string]ritual{}
	for _, m := range matches {
		rel := filepath.ToSlash(strings.TrimPrefix(m, root+string(filepath.Separator)))
		data, readErr := os.ReadFile(m)
		if readErr != nil {
			report(rel, fmt.Sprintf("unreadable ritual SKILL.md: %v", readErr))
			continue
		}
		dir := filepath.Base(filepath.Dir(m))
		found[dir] = ritual{dir: dir, rel: rel, body: string(data)}
	}

	// AC-1 — presence guard: each required wrap ritual carries a
	// trailered-commit block naming all three trailer keys.
	requiredRituals := []string{"aiwfx-wrap-epic", "aiwfx-wrap-milestone"}
	for _, name := range requiredRituals {
		r, ok := found[name]
		if !ok {
			report(
				filepath.ToSlash(filepath.Join(
					"internal", "skills", "embedded-rituals", "plugins", "aiwf-extensions", "skills", name, "SKILL.md")),
				fmt.Sprintf("required ritual %q must carry a trailered-commit block, but its SKILL.md is absent — the G-0219 drift mode", name))
			continue
		}
		if !m0210HasTrailerBlock(r.body) {
			report(r.rel, fmt.Sprintf("required ritual %q must carry a `git commit --trailer` block naming aiwf-verb/aiwf-entity/aiwf-actor — none found (G-0219 drift mode)", name))
			continue
		}
		for _, k := range m0210MissingTrailerKeys(r.body) {
			report(r.rel, fmt.Sprintf("required ritual %q trailered-commit block is missing the %q trailer flag", name, "--trailer \""+k+": …\""))
		}
	}

	// AC-2 — accompaniment guard: the canonical caveat accompanies every
	// trailered-commit block; the identity rule accompanies every staged-merge
	// trailered commit. Applies to every ritual, not only the required wraps.
	for _, r := range found {
		if m0210HasTrailerBlock(r.body) && !m0210HasCaveat(r.body) {
			report(r.rel, fmt.Sprintf("ritual %q composes a trailered commit but is missing the canonical variant-casings caveat (the caveat must accompany every trailer block — single source by policy)", r.dir))
		}
		if m0210HasStagedMerge(r.body) && m0210HasTrailerBlock(r.body) && !m0210HasIdentityRule(r.body) {
			report(r.rel, fmt.Sprintf("ritual %q composes a trailered merge commit but is missing the `git config user.email` identity-resolution rule", r.dir))
		}
	}

	return vs, nil
}

// m0210HasTrailerBlock reports whether body composes an aiwf-trailered commit
// — detected by the `--trailer "aiwf-verb:` flag form (the composition site),
// distinct from an incidental prose mention of `aiwf-verb: promote`.
func m0210HasTrailerBlock(body string) bool {
	return strings.Contains(body, `--trailer "aiwf-verb:`)
}

// m0210MissingTrailerKeys returns the kernel-required trailer keys whose
// `--trailer "<key>:` flag form is absent from body, in canonical order.
func m0210MissingTrailerKeys(body string) []string {
	var missing []string
	for _, k := range []string{"aiwf-verb", "aiwf-entity", "aiwf-actor"} {
		if !strings.Contains(body, `--trailer "`+k+`:`) {
			missing = append(missing, k)
		}
	}
	return missing
}

// m0210HasCaveat reports whether body carries the canonical variant-casings
// caveat, matched case-insensitively on its two distinctive fragments so a
// sentence-position capitalization difference ("Variant" vs "variant") passes.
func m0210HasCaveat(body string) bool {
	l := strings.ToLower(body)
	return strings.Contains(l, "variant casing") && strings.Contains(l, "trailer-keys policy")
}

// m0210HasStagedMerge reports whether body stages a --no-commit merge (the
// site that produces a merge commit needing explicit trailers).
func m0210HasStagedMerge(body string) bool {
	return strings.Contains(body, "git merge --no-ff --no-commit")
}

// m0210HasIdentityRule reports whether body carries the identity-resolution
// rule — resolve from `git config user.email`, do not hardcode — matched
// case-insensitively on its two distinctive fragments.
func m0210HasIdentityRule(body string) bool {
	l := strings.ToLower(body)
	return strings.Contains(l, "git config user.email") && strings.Contains(l, "do not hardcode")
}
