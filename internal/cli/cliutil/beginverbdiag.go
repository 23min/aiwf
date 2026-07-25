package cliutil

import (
	"context"
	"log/slog"
	"os"

	"github.com/23min/aiwf/internal/logger"
)

// BeginVerbDiag sets up this invocation's diagnostic-logging lifecycle
// — the ResolveLogger → run-id / WithVerb → deferred EmitVerbOutcome
// block every instrumented verb otherwise hand-rolls (M-0238/AC-5) —
// and returns a finish closure the caller defers. It is the entrypoint
// for a verb that resolves its actor eagerly in the prelude (every
// mutating verb, which needs the actor for its commit trailer anyway):
//
//	finish := cliutil.BeginVerbDiag(rootDir, "promote", id, actorStr, out.CorrelationID)
//	var sha string
//	defer finish(&code, &sha)
//
// finish takes pointers to the verb's named code return and its sha
// local so the deferred EmitVerbOutcome reports the values as they
// stand at return time — what the inline
// `defer func() { EmitVerbOutcome(diagLog, "verb", code, sha) }()`
// captured. It fires EmitVerbOutcome and then closes the diagnostic
// destination, preserving the inline block's LIFO order (emit before
// close). Binding os.Getenv here is what lets a migrated verb drop the
// os / log/slog / logger imports that existed only to host the block.
//
// A read verb with no --actor flag uses BeginReadVerbDiag instead.
// upgrade keeps its inline wiring as a deliberate non-member: it emits
// under a non-"verb" prefix ("install") from several non-deferred call
// sites, a shape neither helper models.
func BeginVerbDiag(rootDir, verbName, entityID, actor, correlationID string) func(code *int, sha *string) {
	return beginVerbDiagCore(rootDir, os.Getenv, verbName, entityID, func() string { return actor }, correlationID)
}

// BeginReadVerbDiag is BeginVerbDiag for a read verb that has no
// --actor flag and derives a best-effort actor from git config. That
// derivation execs `git config user.email`, so it is deferred into the
// Enabled guard: a read verb with logging off (the default) pays no
// subprocess cost, exactly as the hand-rolled block these verbs carried
// resolved the actor lazily inside their own guard (ADR-0017 —
// diagnostic logging must never affect a verb's own behavior, latency
// included, when disabled).
func BeginReadVerbDiag(rootDir, verbName, entityID, correlationID string) func(code *int, sha *string) {
	resolveActor := func() string { return bestEffortActor(rootDir) }
	return beginVerbDiagCore(rootDir, os.Getenv, verbName, entityID, resolveActor, correlationID)
}

// bestEffortActor derives the actor from git config for a verb with no
// --actor flag, collapsing a resolution failure to "" — a read verb
// must never fail for want of a git identity it never needed.
func bestEffortActor(root string) string {
	actor, err := ResolveActor("", root)
	if err != nil {
		return ""
	}
	return actor
}

// beginVerbDiagCore is the shared engine behind BeginVerbDiag and
// BeginReadVerbDiag: it resolves the logger and — only when the logger
// is enabled — binds verb/entity/actor/run-id, calling resolveActor
// inside that guard so an expensive actor lookup is skipped when
// logging is off. getenv is the ResolveLogger env seam (os.Getenv in
// production; a fake map in tests).
func beginVerbDiagCore(rootDir string, getenv func(string) string, verbName, entityID string, resolveActor func() string, correlationID string) func(code *int, sha *string) {
	diagLog, closeDiagLog := ResolveLogger(rootDir, getenv)
	if diagLog.Enabled(context.Background(), slog.LevelInfo) {
		runID := correlationID
		if runID == "" {
			runID = logger.NewRunID()
		}
		diagLog = logger.WithVerb(diagLog, verbName, entityID, resolveActor(), runID)
	}
	return func(code *int, sha *string) {
		EmitVerbOutcome(diagLog, "verb", *code, *sha)
		_ = closeDiagLog()
	}
}
