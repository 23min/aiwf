// Package scope is the I2.5 scope FSM: the typed lifecycle of an
// authorization grant created by `aiwf authorize`.
//
// A scope is the kernel's unit of "this work is authorized." Its
// state is computed by walking commits forward from the authorize
// commit and applying transitions in commit order. The scope's
// "frontmatter" is the trailer set on the original authorize commit;
// transitions are themselves commits with trailers.
//
// State machine (closed set: active | paused | ended):
//
//	authorize commit lands → state: active
//	   ↓
//	active ──pause──→ paused ──resume──→ active ──...
//	   ↓                              ↓
//	ended ←──── (auto: terminal-promote of the scope-entity carries
//	             aiwf-scope-ends: <auth-sha>)
//
// Legal transitions are enforced by IsLegalScopeTransition. Ended is
// terminal: un-canceling the scope-entity does not resurrect a
// previously-ended scope; the human must issue a new authorization.
//
// Reference: docs/pocv3/design/provenance-model.md §"Scope as a
// first-class FSM".
package scope

import (
	"fmt"

	"github.com/23min/ai-workflow-v2/internal/gitops"
)

// State is the closed set of scope lifecycle states. A scope is
// `active` from the moment its authorize commit lands; pause/resume
// transitions cycle it between `active` and `paused`; either may
// transition to the terminal `ended` when the scope-entity reaches
// a terminal status.
type State string

// State values.
const (
	StateActive State = "active"
	StatePaused State = "paused"
	StateEnded  State = "ended"
)

// Commit is the minimal git-commit shape the scope package consumes:
// a SHA plus the trailer key/value set extracted from the commit
// message. Adapters in cmd/aiwf wrap gitops.HeadTrailers / git log
// output into this shape so the FSM stays I/O-free and unit-testable.
type Commit struct {
	SHA      string
	Trailers []gitops.Trailer
}

// Event is one transition recorded against a scope: the commit SHA
// that produced it, the resulting state, and (for pause/resume) the
// reason text from the aiwf-reason: trailer when present.
type Event struct {
	SHA    string
	State  State
	Reason string
}

// Scope is the materialized lifecycle of one authorization grant.
// AuthSHA is the SHA of the originating authorize commit; Entity is
// the scope-entity id (the value of aiwf-entity: on the authorize
// commit); Agent is the value of aiwf-to: (the agent the scope
// authorizes); Principal is the human who issued the authorization
// (= aiwf-actor: on the authorize commit). State is the current
// state derived from Events; Events lists every transition the scope
// has gone through, in commit order.
type Scope struct {
	AuthSHA   string
	Entity    string
	Agent     string
	Principal string
	State     State
	Events    []Event
}

// IsLegalScopeTransition reports whether moving from -> to is allowed
// by the FSM. Self-loops (active→active, paused→paused) are not legal
// — every transition is meaningful. Ended is terminal.
func IsLegalScopeTransition(from, to State) bool {
	switch from {
	case StateActive:
		return to == StatePaused || to == StateEnded
	case StatePaused:
		return to == StateActive || to == StateEnded
	case StateEnded:
		return false
	}
	return false
}

// LoadScope reconstructs a scope's lifecycle by walking history
// forward from the authorize commit. authSHA names the originating
// commit; history is the commit sequence in chronological order
// (oldest first). The walk stops at the first commit that ends the
// scope (an aiwf-scope-ends: trailer naming authSHA).
//
// Errors:
//   - the authorize commit (history[0]'s SHA == authSHA) must carry
//     aiwf-verb: authorize and aiwf-scope: opened, else the SHA is
//     not actually a scope opener.
//   - a pause/resume on a state that doesn't allow it produces a
//     malformed-history error (the FSM was driven by hand-edited
//     commits that bypassed the verb).
//
// The function is pure: no I/O, no git subprocess. Adapters in
// cmd/aiwf produce the Commit slice via gitops.
func LoadScope(authSHA string, history []Commit) (*Scope, error) {
	if len(history) == 0 {
		return nil, fmt.Errorf("scope %s: empty history", authSHA)
	}
	opener := history[0]
	if opener.SHA != authSHA {
		return nil, fmt.Errorf("scope %s: history[0] SHA %q does not match authSHA", authSHA, opener.SHA)
	}
	idx := indexTrailers(opener.Trailers)
	if idx[gitops.TrailerVerb] != "authorize" {
		return nil, fmt.Errorf("scope %s: opener is not an authorize commit (verb=%q)", authSHA, idx[gitops.TrailerVerb])
	}
	if idx[gitops.TrailerScope] != "opened" {
		return nil, fmt.Errorf("scope %s: opener does not carry aiwf-scope: opened (got %q)", authSHA, idx[gitops.TrailerScope])
	}

	s := &Scope{
		AuthSHA:   authSHA,
		Entity:    idx[gitops.TrailerEntity],
		Agent:     idx[gitops.TrailerTo],
		Principal: idx[gitops.TrailerActor],
		State:     StateActive,
		Events: []Event{
			{SHA: opener.SHA, State: StateActive, Reason: idx[gitops.TrailerReason]},
		},
	}

	for _, c := range history[1:] {
		if s.State == StateEnded {
			break
		}
		next, reason, ended := classifyTransition(authSHA, c)
		if !ended && next == "" {
			// Not a transition for this scope — skip.
			continue
		}
		var resulting State
		if ended {
			resulting = StateEnded
		} else {
			resulting = next
		}
		if !IsLegalScopeTransition(s.State, resulting) {
			return nil, fmt.Errorf("scope %s: illegal transition %s → %s at commit %s",
				authSHA, s.State, resulting, c.SHA)
		}
		s.State = resulting
		s.Events = append(s.Events, Event{SHA: c.SHA, State: resulting, Reason: reason})
	}
	return s, nil
}

// classifyTransition inspects a commit and returns the resulting
// scope state for the named auth SHA, the reason text, and whether
// the commit carries an explicit aiwf-scope-ends: marker.
//
// Returns ("", "", false) for commits that don't transition this
// scope. Recognized cases:
//
//   - authorize-pause / authorize-resume on this scope's entity
//     (matched by aiwf-verb: authorize + the scope state value).
//     Note: pause/resume commits are addressed by the entity id, not
//     by the auth SHA; the LoadScope walker assumes any pause/resume
//     in the history slice is for the scope being loaded. Callers
//     pre-filter by walking entity history if multiple scopes exist
//     on the same entity (rare).
//
//   - any commit carrying aiwf-scope-ends: <authSHA>: ends this
//     scope (auto-end on terminal-promote, or — once G22 lands — an
//     explicit revoke).
func classifyTransition(authSHA string, c Commit) (next State, reason string, ended bool) {
	idx := indexTrailers(c.Trailers)

	// Auto-end takes precedence: a single commit that both pauses a
	// scope and ends it (degenerate but possible if hand-crafted)
	// resolves to ended.
	for _, tr := range c.Trailers {
		if tr.Key == gitops.TrailerScopeEnds && tr.Value == authSHA {
			return "", idx[gitops.TrailerReason], true
		}
	}

	if idx[gitops.TrailerVerb] != "authorize" {
		return "", "", false
	}
	switch idx[gitops.TrailerScope] {
	case "paused":
		return StatePaused, idx[gitops.TrailerReason], false
	case "resumed":
		return StateActive, idx[gitops.TrailerReason], false
	default:
		// "opened" on a different SHA isn't this scope's transition;
		// any other value isn't recognized.
		return "", "", false
	}
}

// indexTrailers builds a key→value map from a commit's trailers.
// When a key appears more than once (notably aiwf-scope-ends, which
// can repeat on a commit ending multiple scopes), the last value
// wins. Callers needing per-key iteration should walk the slice
// directly.
func indexTrailers(trailers []gitops.Trailer) map[string]string {
	out := make(map[string]string, len(trailers))
	for _, tr := range trailers {
		out[tr.Key] = tr.Value
	}
	return out
}
