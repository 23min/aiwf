package verb

import (
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/check"
)

// ScopeOutOfReachError reports that an authorized agent's verb was
// refused because its target falls outside the three-edge reach
// (D-0006) of every active scope attached to the actor. It implements
// [entity.Coded], carrying check.CodeProvenanceAuthorizationOutOfScope —
// the same code the check-time provenance audit emits for the identical
// violation (one predicate, two enforcement times; D-0014).
type ScopeOutOfReachError struct {
	// Actor is the non-human operator whose act was refused.
	Actor string
	// Target is the entity the verb would have mutated (the destination,
	// for a move). Empty for a creation act, where Refs carries the
	// proposed outbound references instead.
	Target string
	// Refs are the creation act's proposed outbound references, used for
	// the message subject when Target is empty.
	Refs []string
}

// Error implements error, naming the actor and the unreachable subject
// and including the code id so message-matching consumers recognize it.
func (e *ScopeOutOfReachError) Error() string {
	subject := e.Target
	if subject == "" && len(e.Refs) > 0 {
		subject = strings.Join(e.Refs, ", ")
	}
	return fmt.Sprintf("actor %q: target %s is outside the reach of every active scope (%s)",
		e.Actor, subject, check.CodeProvenanceAuthorizationOutOfScope)
}

// Code returns check.CodeProvenanceAuthorizationOutOfScope, satisfying
// [entity.Coded].
func (e *ScopeOutOfReachError) Code() string { return check.CodeProvenanceAuthorizationOutOfScope }

// NoActiveScopeError reports that a non-human actor attempted a verb
// with no active scope at all — distinct from out-of-reach, where a
// scope exists but does not contain the target. It implements
// [entity.Coded], carrying check.CodeProvenanceNoActiveScope.
type NoActiveScopeError struct {
	// Actor is the non-human operator whose act was refused.
	Actor string
}

// Error implements error.
func (e *NoActiveScopeError) Error() string {
	return fmt.Sprintf("actor %q has no active scope authorizing this act (%s)",
		e.Actor, check.CodeProvenanceNoActiveScope)
}

// Code returns check.CodeProvenanceNoActiveScope, satisfying [entity.Coded].
func (e *NoActiveScopeError) Code() string { return check.CodeProvenanceNoActiveScope }
