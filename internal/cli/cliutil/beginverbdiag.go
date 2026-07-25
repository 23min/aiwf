package cliutil

import (
	"context"
	"log/slog"
	"os"

	"github.com/23min/aiwf/internal/logger"
)

// BeginVerbDiag sets up this invocation's diagnostic-logging lifecycle
// — the ~11-line ResolveLogger → run-id / WithVerb → deferred
// EmitVerbOutcome block every instrumented verb otherwise hand-rolls
// (M-0238/AC-5) — and returns a finish closure the caller defers.
// Canonical use:
//
//	finish := cliutil.BeginVerbDiag(rootDir, "promote", id, actorStr, out.CorrelationID)
//	var sha string
//	defer finish(&code, &sha)
//
// finish takes pointers to the verb's named code return and its sha
// local so the deferred EmitVerbOutcome reports the values as they
// stand at return time — exactly what the inline
// `defer func() { EmitVerbOutcome(diagLog, "verb", code, sha) }()`
// captured. It fires EmitVerbOutcome and then closes the diagnostic
// destination, preserving the inline block's LIFO order (emit before
// close). Binding os.Getenv here is what lets a migrated verb drop the
// os / log/slog / logger imports that existed only to host the block.
//
// A verb with an idiosyncratic diagnostic setup — a non-"verb" event
// prefix, or a non-deferred emit at several call sites — is a
// deliberate non-member and keeps its inline wiring rather than being
// forced through this helper.
func BeginVerbDiag(rootDir, verbName, entityID, actor, correlationID string) func(code *int, sha *string) {
	return beginVerbDiag(rootDir, os.Getenv, verbName, entityID, actor, correlationID)
}

// beginVerbDiag is BeginVerbDiag with an injectable getenv seam for
// tests; production callers use BeginVerbDiag, which binds os.Getenv.
func beginVerbDiag(rootDir string, getenv func(string) string, verbName, entityID, actor, correlationID string) func(code *int, sha *string) {
	diagLog, closeDiagLog := ResolveLogger(rootDir, getenv)
	if diagLog.Enabled(context.Background(), slog.LevelInfo) {
		runID := correlationID
		if runID == "" {
			runID = logger.NewRunID()
		}
		diagLog = logger.WithVerb(diagLog, verbName, entityID, actor, runID)
	}
	return func(code *int, sha *string) {
		EmitVerbOutcome(diagLog, "verb", *code, *sha)
		_ = closeDiagLog()
	}
}
