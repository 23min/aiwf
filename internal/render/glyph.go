package render

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
	case "done", "met", "addressed", "accepted":
		return "✓"
	// → — moving
	case "in_progress", "active":
		return "→"
	// ○ — not started
	case "open", "draft", "proposed":
		return "○"
	// ✗ — closed off
	case "cancelled", "wontfix", "rejected", "retired", "superseded":
		return "✗"
	default:
		return ""
	}
}
