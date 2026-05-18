package render

import "github.com/23min/aiwf/internal/entity"

// StatusGlyph returns the canonical glyph for a kernel status string,
// or "" when the status is unrecognised (e.g. a value loaded from a
// pre-canonical commit, or a typo). The palette is per G-0080's
// resolution sketch:
//
//	✓  met / done / addressed / accepted    — "the work is finished"
//	→  in_progress / active                 — "the work is moving"
//	○  open / draft / proposed              — "the work hasn't started"
//	✗  cancelled / wontfix / rejected /
//	   retired / superseded                 — "the work is closed off"
//
// Every glyph is 1-cell BMP so it works under text/tabwriter's
// rune-counting (the kernel does not pull github.com/mattn/go-runewidth
// per G-0080's *Out of scope*). The mapping is per-status-value, not
// per-kind: every kind's status set is a subset of the four buckets
// above, so one map covers epic/milestone/ADR/gap/decision/contract.
//
// Unknown statuses return "" — callers render the status text without
// a glyph rather than guess. The set is intentionally exhaustive over
// the kernel's current status vocabulary; ADR-0008 keeps that
// vocabulary stable.
func StatusGlyph(status string) string {
	switch status {
	// ✓ — finished
	case entity.StatusDone, entity.StatusMet, entity.StatusAddressed, entity.StatusAccepted:
		return "✓"
	// → — moving
	case entity.StatusInProgress, entity.StatusActive:
		return "→"
	// ○ — not started
	case entity.StatusOpen, entity.StatusDraft, entity.StatusProposed:
		return "○"
	// ✗ — closed off
	case entity.StatusCancelled, entity.StatusWontfix, entity.StatusRejected, entity.StatusRetired, entity.StatusSuperseded:
		return "✗"
	default:
		return ""
	}
}
