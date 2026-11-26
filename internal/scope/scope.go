// Package scope is the I2.5 scope FSM: the typed lifecycle of an
// authorization grant created by `aiwf authorize`.
//
// A scope is the kernel's unit of "this work is authorized." Replay
// computes each scope's state by walking its entity's commits in order
// and applying the transitions they record. The scope's "frontmatter"
// is the trailer set on the original authorize commit; transitions are
// themselves commits with trailers.
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
// Reference: docs/design/provenance-model.md §"Scope as a
// first-class FSM".
package scope

import (
	"github.com/23min/aiwf/internal/gitops"
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
// message. Callers build it from git log output, so Replay stays
// I/O-free and unit-testable.
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

// Replay derives every scope opened on one entity and each scope's
// current state. commits are that entity's commits, oldest first: those
// whose aiwf-entity trailer names it. Scopes come back in open order.
//
//   - An authorize commit carrying aiwf-scope: opened opens a scope,
//     active.
//   - An authorize commit carrying aiwf-scope: paused or resumed moves
//     the scope Target picks. When Target picks none, the commit
//     changes nothing.
//   - Each aiwf-scope-ends trailer, on any commit, ends the scope opened
//     by the commit it names, unless that scope has already ended.
//
// The verb refuses a pause or resume no scope can make, but history can
// still hold one: merging two branches that each paused the entity's only
// active scope leaves the second pause with none to move. Replay skips it,
// so the entity's scopes still read rather than the whole history failing.
//
// The function is pure: no I/O, no git subprocess.
func Replay(commits []Commit) []*Scope {
	var scopes []*Scope
	byAuth := map[string]*Scope{}
	for _, c := range commits {
		idx := indexTrailers(c.Trailers)
		reason := idx[gitops.TrailerReason]
		if idx[gitops.TrailerVerb] == "authorize" {
			switch idx[gitops.TrailerScope] {
			case "opened":
				s := &Scope{
					AuthSHA:   c.SHA,
					Entity:    idx[gitops.TrailerEntity],
					Agent:     idx[gitops.TrailerTo],
					Principal: idx[gitops.TrailerActor],
					State:     StateActive,
					Events:    []Event{{SHA: c.SHA, State: StateActive, Reason: reason}},
				}
				scopes = append(scopes, s)
				byAuth[c.SHA] = s
			case "paused":
				if s := Target(scopes, StatePaused); s != nil {
					s.move(c.SHA, StatePaused, reason)
				}
			case "resumed":
				if s := Target(scopes, StateActive); s != nil {
					s.move(c.SHA, StateActive, reason)
				}
			}
		}
		// aiwf-scope-ends repeats, one trailer per scope a commit ends,
		// so walk the trailers rather than the index.
		for _, tr := range c.Trailers {
			if tr.Key != gitops.TrailerScopeEnds {
				continue
			}
			if s := byAuth[tr.Value]; s != nil && IsLegalScopeTransition(s.State, StateEnded) {
				s.move(c.SHA, StateEnded, reason)
			}
		}
	}
	return scopes
}

// Target returns the scope a pause (to = StatePaused) or a resume
// (to = StateActive) moves: of the scopes IsLegalScopeTransition lets
// move to `to`, the most recently opened, or nil when there is none.
// scopes are in open order, as Replay returns them. A pause or resume
// commit names no scope, so `aiwf authorize`, deciding whether to refuse
// one, and Replay, reading it back, both ask Target which scope it moves.
func Target(scopes []*Scope, to State) *Scope {
	for i := len(scopes) - 1; i >= 0; i-- {
		if IsLegalScopeTransition(scopes[i].State, to) {
			return scopes[i]
		}
	}
	return nil
}

// move records the transition to `to` made by commit sha.
func (s *Scope) move(sha string, to State, reason string) {
	s.State = to
	s.Events = append(s.Events, Event{SHA: sha, State: to, Reason: reason})
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
