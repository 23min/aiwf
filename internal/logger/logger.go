package logger

import (
	"io"
	"log/slog"
)

// New returns a *slog.Logger reflecting cfg. When cfg.Enabled is
// false — the default (ADR-0017 Decision #2) — the returned logger is
// backed by slog.DiscardHandler, whose Enabled method reports false
// for every level: emitting a record does no I/O and no allocation
// beyond the call itself, and w is never touched.
//
// When cfg.Enabled is true, records at or above cfg.Level are written
// to w in cfg.Format ("text" or "json"). w's destination (the default
// XDG-state-home file vs. an explicit override) and its concurrent-
// append safety are resolved by the caller — this constructor only
// wraps whatever writer it is given.
func New(cfg Config, w io.Writer) *slog.Logger {
	if !cfg.Enabled {
		return slog.New(slog.DiscardHandler)
	}
	opts := &slog.HandlerOptions{Level: cfg.Level}
	var h slog.Handler
	if cfg.Format == "json" {
		h = slog.NewJSONHandler(w, opts)
	} else {
		h = slog.NewTextHandler(w, opts)
	}
	return slog.New(h)
}
