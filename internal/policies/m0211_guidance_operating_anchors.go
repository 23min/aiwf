package policies

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PolicyM0211GuidanceOperatingAnchors asserts that a curated set of
// consumer-operating rules is present in the shippable embedded guidance
// source (internal/skills/embedded-guidance/aiwf-guidance.md) — the drift
// chokepoint for M-0211 / G-0313.
//
// The gap G-0313 addresses: consumer-facing *operating* guidance (how to drive
// aiwf the tool in any repo) keeps accreting in this repo's CLAUDE.md, which
// never ships to consumers. The shippable home is the embedded guidance source,
// which aiwf init / aiwf update materialize into a consumer's
// .claude/aiwf-guidance.md and @-import into their CLAUDE.md. A rule that
// belongs there but lands in CLAUDE.md instead is invisible to every consumer
// and forks from the single source of truth.
//
// A fully mechanical "this rule belongs in the guidance" test is infeasible —
// tool-operation vs repo-development is a judgment call this policy cannot make.
// What it *can* police is the regression direction: an operating rule that has
// already shipped must not silently drift back *out* of the fragment. So the
// policy asserts a curated anchor set (each a named rule plus the distinctive
// fragments that mark its presence) is present in the shipped guidance; trimming
// any of them out reddens CI. The forward direction — a brand-new rule authored
// into the wrong home — is the judgment call the CLAUDE.md authoring rule
// ("audience, not importance") plus review is the interim catch for, until the
// rule is added to this curated set.
//
// This is an aiwf-repo development invariant — the embedded-guidance tree exists
// only here — so it lives as a Go policy test, mirroring the sibling M-0209 /
// M-0210 guidance/ritual policies, not as an `aiwf check` finding (which would
// be inert in a consumer tree, where the guidance is materialized rather than
// authored).
func PolicyM0211GuidanceOperatingAnchors(root string) ([]Violation, error) {
	rel := filepath.ToSlash(filepath.Join("internal", "skills", "embedded-guidance", "aiwf-guidance.md"))
	data, err := os.ReadFile(filepath.Join(root, "internal", "skills", "embedded-guidance", "aiwf-guidance.md"))
	if err != nil {
		return []Violation{{
			Policy: "m0211-guidance-operating-anchors",
			File:   rel,
			Detail: fmt.Sprintf("shippable guidance source is unreadable — consumer-operating rules cannot ship from it: %v", err),
		}}, nil
	}

	lower := strings.ToLower(string(data))
	var vs []Violation
	for _, a := range m0211OperatingAnchors {
		if !a.present(lower) {
			vs = append(vs, Violation{
				Policy: "m0211-guidance-operating-anchors",
				File:   rel,
				Detail: fmt.Sprintf("consumer-operating anchor %q has drifted out of the shippable guidance source — an operating rule that ships nowhere (G-0313); restore it to the embedded guidance", a.name),
			})
		}
	}
	return vs, nil
}

// m0211Anchor names a consumer-operating rule and the distinctive lower-case
// fragments that must all appear for the rule to count as present in the
// shipped guidance. Fragments are matched against a lower-cased body, so a
// sentence-position capitalization difference does not defeat the match.
type m0211Anchor struct {
	name      string
	fragments []string
}

// present reports whether every fragment of the anchor appears in lowerBody
// (which the caller has already lower-cased).
func (a m0211Anchor) present(lowerBody string) bool {
	for _, f := range a.fragments {
		if !strings.Contains(lowerBody, f) {
			return false
		}
	}
	return true
}

// m0211OperatingAnchors is the curated set of consumer-operating rules whose
// presence in the shipped guidance the chokepoint guarantees. Adding a new
// consumer-operating rule to the guidance and to this slice extends the
// guarantee; the set is deliberately hand-curated (the audience call is human
// judgment, per the CLAUDE.md "audience, not importance" authoring rule).
var m0211OperatingAnchors = []m0211Anchor{
	{"gate-per-mutation", []string{"each mutating action", "approval gate"}},
	{"reallocate-not-git-mv", []string{"aiwf reallocate", "git mv"}},
	{"ac-mechanical-evidence", []string{"mechanical evidence"}},
	{"one-decision-at-a-time", []string{"one thing at a time"}},
	{"never-suggest-pause", []string{"never suggest", "pause"}},
	{"body-prose-id", []string{"body-prose-id"}},
	{"cross-branch-allocation", []string{"--fetch", "push promptly"}},
	{"bless-mode-body-edits", []string{"bless mode", "review-before-commit"}},
	{"verb-template-managed-entities", []string{"template-managed", "hand-edit frontmatter"}},
}
