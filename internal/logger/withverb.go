package logger

import (
	"log/slog"
	"regexp"
)

// homeLeakPatterns mirror .gitleaks.toml's path-leak-darwin-home and
// path-leak-linux-home rules — the same absolute-path contributor-
// identity leak the repo's committed-text lint polices, applied here
// to bound logger field values instead of source text.
var homeLeakPatterns = []*regexp.Regexp{
	regexp.MustCompile(`/Users/[A-Za-z][A-Za-z0-9_.-]*/`),
	regexp.MustCompile(`/home/[A-Za-z][A-Za-z0-9_.-]*/`),
}

// WithVerb binds verb/entity/actor onto l (ADR-0017 Decision #7),
// scrubbing any macOS (/Users/<name>/) or Linux (/home/<name>/)
// home-directory fragment from each value first. The scrub operates
// on string content, not provenance, so it catches a leak regardless
// of source — including a value assembled from os.Args that a caller
// passed through verb/entity/actor unfiltered.
func WithVerb(l *slog.Logger, verb, entity, actor string) *slog.Logger {
	return l.With(
		"verb", scrubHomePaths(verb),
		"entity", scrubHomePaths(entity),
		"actor", scrubHomePaths(actor),
	)
}

// scrubHomePaths replaces every macOS or Linux home-directory fragment
// in s with a redaction marker, leaving the rest of the value —
// including whatever follows the fragment — intact.
func scrubHomePaths(s string) string {
	for _, re := range homeLeakPatterns {
		s = re.ReplaceAllString(s, "<redacted-home>/")
	}
	return s
}
