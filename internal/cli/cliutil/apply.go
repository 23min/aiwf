package cliutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/verb"
)

// internalError marks an error that FinishVerb/FinishVerbOutcome's err
// branch maps to ExitInternal instead of the default ExitUsage — the
// caller's own infrastructure breaking (a config/tree load failure, a
// domain verb call erroring outright) rather than a usage mistake.
// Wrap with ErrInternal; unexported so callers can't construct one
// bypassing that constructor.
type internalError struct{ err error }

func (e *internalError) Error() string { return e.err.Error() }
func (e *internalError) Unwrap() error { return e.err }

// ErrInternal wraps err as an error that FinishVerb/FinishVerbOutcome
// report as ExitInternal (3) rather than the default ExitUsage (2) for
// a non-Coded error — e.g. archive/rewidth/import's early
// LoadTreeWithTrunk / verb-call failures (M-0271/AC-2), which predate
// FinishVerb and were always ExitInternal. Unwraps to err, so a caller
// further up the stack can still errors.Is/As into the original cause.
func ErrInternal(err error) error { return &internalError{err: err} }

// FinishVerb is the post-verb handler shared by every mutating
// subcommand: it surfaces the verb's outcome in the chosen output format
// (text by default, a JSON envelope under --format=json per D-0013),
// applies the plan when present, and reports the exit code.
//
// Exit-code contract (format-independent):
//   - a Coded error (entity.Code resolves) → ExitFindings (1): a
//     legality refusal, unified with the check-time exit for the same
//     violation class (D-0013, decision C2);
//   - an ErrInternal-wrapped error → ExitInternal (3): the caller's
//     own infrastructure broke, not a usage mistake;
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
//
// FinishVerb is a thin single-Plan adapter over FinishVerbOutcome
// (M-0271/AC-1): the conversion below is total and lossless for
// verb.Result's shape, so every existing caller's behavior is
// unchanged.
func FinishVerb(ctx context.Context, root, label string, result *verb.Result, err error, out OutputFormat) (code int, sha string) {
	var outcome *Outcome
	if result != nil {
		var plans []*verb.Plan
		if result.Plan != nil {
			plans = []*verb.Plan{result.Plan}
		}
		outcome = &Outcome{
			Findings:    result.Findings,
			Plans:       plans,
			NoOp:        result.NoOp,
			NoOpMessage: result.NoOpMessage,
			Metadata:    result.Metadata,
		}
	}
	return FinishVerbOutcome(ctx, root, label, outcome, err, out)
}

// Outcome generalizes verb.Result for FinishVerbOutcome's dry-run- and
// multi-Plan-capable callers (archive, rewidth, import — M-0271/AC-2):
// Plans replaces the single Plan field (the common case populates
// exactly one), and DryRun selects the preview branch — report what
// would happen without calling verb.Apply — instead of applying.
type Outcome struct {
	Findings    []check.Finding
	Plans       []*verb.Plan
	NoOp        bool
	NoOpMessage string
	Metadata    map[string]any

	// DryRun skips verb.Apply; Subject/Metadata are reported as a
	// preview instead of an applied result, and the returned sha is
	// always "".
	DryRun bool

	// Subject overrides the emitted JSON result's subject and the
	// text-mode dry-run fallback line when set; otherwise the last
	// entry of Plans supplies it. Text-mode apply output ignores this
	// field entirely (see below) — it always prints one line per Plan.
	Subject string

	// TextDetail, when set, replaces the bare subject line for a
	// text-mode dry-run with a verb-specific multi-line preview
	// (archive's move list, rewidth's per-op summary, import's
	// per-plan write listing). JSON mode never calls it — the envelope
	// carries no such narrative. Ignored outside DryRun.
	TextDetail func()
}

// FinishVerbOutcome is FinishVerb's generalization: Plans may hold
// more than one entry (each applied in order; the returned sha is the
// last one's), and DryRun previews the outcome without ever calling
// verb.Apply. Exit-code contract and error-envelope shape are
// otherwise identical to FinishVerb — see its doc comment.
//
// Text-mode output diverges deliberately by branch: a successful
// apply prints one subject line per Plan (reproducing import's
// pre-existing per-plan loop, which for a single Plan is the same
// bytes FinishVerb always printed), while a dry-run prints
// TextDetail's verb-specific preview instead — the two shapes were
// already this different before the migration, and unifying them
// would change observable output rather than just its plumbing.
func FinishVerbOutcome(ctx context.Context, root, label string, outcome *Outcome, err error, out OutputFormat) (code int, sha string) {
	if err != nil {
		codeStr, isCoded := entity.Code(err)
		out.emitErrorEnvelope(label, codeStr, err.Error())
		var internalErr *internalError
		switch {
		case isCoded:
			return ExitFindings, ""
		case errors.As(err, &internalErr):
			return ExitInternal, ""
		default:
			return ExitUsage, ""
		}
	}
	if outcome == nil {
		out.emitErrorEnvelope(label, "", "no result returned")
		return ExitInternal, ""
	}
	if check.HasErrors(outcome.Findings) {
		out.emitFindings(outcome.Findings)
		return ExitFindings, ""
	}
	if outcome.NoOp {
		out.emitSuccess(outcome.NoOpMessage, nil, outcome.Metadata)
		return ExitOK, ""
	}
	if len(outcome.Plans) == 0 {
		out.emitErrorEnvelope(label, "", "validation passed but no plan produced")
		return ExitInternal, ""
	}

	subject := outcome.Subject
	if subject == "" {
		subject = outcome.Plans[len(outcome.Plans)-1].Subject
	}

	if outcome.DryRun {
		switch {
		case out.JSON():
			out.emitSuccess(subject, nil, outcome.Metadata)
		case outcome.TextDetail != nil:
			outcome.TextDetail()
		default:
			Println(subject)
		}
		return ExitOK, ""
	}

	var traceLog func()
	if out.Trace {
		traceLog = startTrace(root)
	}
	for i, p := range outcome.Plans {
		planSHA, applyErr := verb.Apply(ctx, root, p)
		if applyErr != nil {
			if traceLog != nil {
				traceLog()
			}
			msg := applyErr.Error()
			if len(outcome.Plans) > 1 {
				msg = fmt.Sprintf("applying plan %d: %v", i, applyErr)
			}
			out.emitErrorEnvelope(label, "", msg)
			return ExitInternal, ""
		}
		sha = planSHA
	}
	if traceLog != nil {
		traceLog()
	}

	if !out.JSON() {
		// Warning-level findings may travel with a successful plan
		// (e.g., reallocate body-prose mentions).
		if len(outcome.Findings) > 0 {
			_ = render.Text(os.Stderr, outcome.Findings)
		}
		for _, p := range outcome.Plans {
			Println(p.Subject)
		}
		return ExitOK, sha
	}
	// commit_sha rides alongside the verb's own metadata (M-0239/AC-2)
	// rather than mutating outcome.Metadata, so a caller reusing that
	// map elsewhere never sees a surprise key.
	out.emitSuccess(subject, outcome.Findings, withCommitSHA(outcome.Metadata, sha))
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
