package cliutil

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
)

// ParseKind parses a CLI kind argument (lowercase string) into the
// entity.Kind constant. Returns the canonical Kind and true on match;
// the empty Kind and false when no kind in entity.AllKinds() matches s.
func ParseKind(s string) (entity.Kind, bool) {
	for _, k := range entity.AllKinds() {
		if string(k) == s {
			return k, true
		}
	}
	return "", false
}

// ParseTestsFlag parses the --tests flag value (e.g. "pass=12 fail=0
// skip=0") into a *gitops.TestMetrics. Empty input returns (nil, nil)
// (the flag was unset — not an error). Malformed input writes a
// one-line error to stderr (prefixed with verbLabel) and returns the
// parse error so the dispatcher exits with cliutil.ExitUsage.
//
// The "metrics parsed to zero" defensive branch returns (nil, nil) —
// gitops.ParseStrictTestMetrics returns the zero TestMetrics for empty
// input, but here the trimmed input was non-empty so this shouldn't
// fire. If it does, treat the flag as not set to avoid emitting a
// meaningless trailer.
func ParseTestsFlag(raw, verbLabel string) (*gitops.TestMetrics, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	m, err := gitops.ParseStrictTestMetrics(raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", verbLabel, err)
		return nil, err
	}
	if m == (gitops.TestMetrics{}) {
		return nil, nil
	}
	return &m, nil
}

// ReadBodyFile loads body content for `aiwf add --body-file`. A path
// of "-" reads stdin (so callers can pipe body text without a temp
// file). Any other value is read as a regular file. Returns the raw
// bytes; the verb-side resolveAddBody is the rule-checking layer (it
// refuses content that begins with a frontmatter delimiter so the
// create commit can't accidentally produce a double-frontmatter file).
func ReadBodyFile(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}

// SplitCommaList parses comma-separated CLI values into a clean slice
// (trimmed, empty entries dropped). Shared between --relates-to,
// --linked-adr, --depends-on, and similar multi-value flags.
func SplitCommaList(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, item := range strings.Split(s, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}
