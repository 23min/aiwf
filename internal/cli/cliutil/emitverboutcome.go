package cliutil

import "log/slog"

// EmitVerbOutcome is the single tail-diagnostic-event shape every
// instrumented verb converges on (M-0238/AC-5, AC-6): "<prefix>.completed"
// on ExitOK (with sha when the verb produced a commit — cancel/move do,
// upgrade/statusline don't), "<prefix>.failed" otherwise, carrying the
// exit code and its error class. log is expected to already be
// WithVerb-bound; a disabled logger's own Enabled gate makes this a
// cheap no-op call regardless of code/sha.
func EmitVerbOutcome(log *slog.Logger, prefix string, code int, sha string) {
	if code == ExitOK {
		if sha != "" {
			log.Info(prefix+".completed", "sha", sha)
			return
		}
		log.Info(prefix + ".completed")
		return
	}
	log.Error(prefix+".failed", "exit_code", code, "error_class", errorClassForExitCode(code))
}

// errorClassForExitCode names the outcome class an operator-facing
// exit code represents, reusing this repo's own existing exit-code
// taxonomy (CLAUDE.md §CLI conventions) rather than inventing a
// second one for diagnostic events.
func errorClassForExitCode(code int) string {
	switch code {
	case ExitFindings:
		return "findings"
	case ExitUsage:
		return "usage"
	case ExitInternal:
		return "internal"
	default:
		return "unknown"
	}
}
