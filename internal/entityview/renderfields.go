package entityview

// renderfields.go holds the display rules for a HistoryEvent's trailer-backed
// columns. They live beside the type rather than in one of its consumers
// because three surfaces render the same events — `aiwf history`, `aiwf show`,
// and the HTML site — and a rule kept in one of them is a rule the other two
// can silently disagree with.

// RenderVerb formats the verb column. Empty renders as "-", the marker an
// absent target status already uses: an event can carry the entity trailer
// alone, and no verb committed it. A synthesized label is not the alternative —
// the column would name something no aiwf verb did, and the JSON `verb` field
// would carry a token outside the closed set.
func RenderVerb(verb string) string {
	if verb == "" {
		return "-"
	}
	return verb
}

// RenderActor formats the actor column. When a non-human principal is present
// and differs from the actor (the agent-acts-for-human case), the column reads
// `principal via agent` so the human is visually attributed first. Direct human
// acts (no principal) render the actor verbatim. An event carrying neither
// renders "-".
//
// A principal recorded without an actor renders alone. No verb writes that
// pair, but rendering it as `principal via ` with an empty side states an
// agent that is not there, and dropping the principal discards provenance the
// commit does carry.
func RenderActor(e HistoryEvent) string {
	if e.Actor == "" {
		if e.Principal != "" {
			return e.Principal
		}
		return "-"
	}
	if e.Principal == "" || e.Principal == e.Actor {
		return e.Actor
	}
	return e.Principal + " via " + e.Actor
}
