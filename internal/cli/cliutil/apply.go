package cliutil

import (
	"context"
	"os"
	"time"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// FinishVerb is the post-verb handler shared by every mutating
// subcommand: it surfaces the verb's outcome in the chosen output format
// (text by default, a JSON envelope under --format=json per D-0013),
// applies the plan when present, and reports the exit code.
//
// Exit-code contract (format-independent):
//   - a Coded error (entity.Code resolves) → ExitFindings (1): a
//     legality refusal, unified with the check-time exit for the same
//     violation class (D-0013, decision C2);
//   - any other verb error → ExitUsage (2);
//   - nil result / no plan / apply failure → ExitInternal (3);
//   - error-severity findings → ExitFindings (1);
//   - success (incl. NoOp, warnings) → ExitOK (0).
//
// The second return value is the resulting commit sha on a clean
// ExitOK apply, "" otherwise (M-0238/AC-5) — most callers have no use
// for it and discard it via `_`; the handful of verbs instrumented
// with diagnostic logging (cancel, move) cite it in their completion
// event.
func FinishVerb(ctx context.Context, root, label string, result *verb.Result, err error, out OutputFormat) (code int, sha string) {
	if err != nil {
		codeStr, isCoded := entity.Code(err)
		out.emitErrorEnvelope(label, codeStr, err.Error())
		if isCoded {
			return ExitFindings, ""
		}
		return ExitUsage, ""
	}
	if result == nil {
		out.emitErrorEnvelope(label, "", "no result returned")
		return ExitInternal, ""
	}
	if check.HasErrors(result.Findings) {
		out.emitFindings(result.Findings)
		return ExitFindings, ""
	}
	if result.NoOp {
		out.emitSuccess(result.NoOpMessage, nil, result.Metadata)
		return ExitOK, ""
	}
	if result.Plan == nil {
		out.emitErrorEnvelope(label, "", "validation passed but no plan produced")
		return ExitInternal, ""
	}
	var traceLog func()
	if out.Trace {
		traceLog = startTrace(root)
	}
	sha, applyErr := verb.Apply(ctx, root, result.Plan)
	if traceLog != nil {
		traceLog()
	}
	if applyErr != nil {
		out.emitErrorEnvelope(label, "", applyErr.Error())
		return ExitInternal, ""
	}
	// Warning-level findings may travel with a successful plan (e.g.,
	// reallocate body-prose mentions); emitSuccess surfaces them but the
	// exit code stays clean. commit_sha rides alongside the verb's own
	// metadata (M-0239/AC-2) rather than mutating result.Metadata, so a
	// caller reusing that map elsewhere never sees a surprise key.
	out.emitSuccess(result.Plan.Subject, result.Findings, withCommitSHA(result.Metadata, sha))
	return ExitOK, sha
}

// startTrace resolves a debug-forced logger (M-0239/AC-3's --trace)
// and starts a timer. The returned func emits "phase.apply" at debug
// level with the elapsed milliseconds and closes the logger; call it
// exactly once, immediately after the timed operation returns —
// FinishVerb calls it right after verb.Apply, so the timing covers
// only the git-touching apply phase, not validation/planning.
func startTrace(root string) func() {
	log, closeLog := ResolveTraceLogger(root, os.Getenv)
	start := time.Now()
	return func() {
		log.Debug("phase.apply", "elapsed_ms", time.Since(start).Milliseconds())
		_ = closeLog()
	}
}

// withCommitSHA returns a copy of md with "commit_sha" set to sha,
// without mutating md. sha is always non-empty at this function's one
// call site (FinishVerb only reaches it after a successful
// verb.Apply), so the returned map is never empty in practice; when it
// is (a hypothetical future caller passing both empty), Envelope's
// Metadata field carries `omitempty`, which treats a zero-length map
// identically to nil — no special-casing needed here.
func withCommitSHA(md map[string]any, sha string) map[string]any {
	out := make(map[string]any, len(md)+1)
	for k, v := range md {
		out[k] = v
	}
	if sha != "" {
		out["commit_sha"] = sha
	}
	return out
}
