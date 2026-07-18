package integration

import (
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// TestPolicy_HelpBannerCoversVerbs is the drift-prevention chokepoint
// behind G-0285: the root `aiwf --help` banner (printHelp() in
// internal/cli/root.go) is a hand-maintained string literal, so it
// can silently drift from the registered Cobra command tree. This
// test walks the tree NewRootCmd() actually built and asserts every
// top-level verb name is mentioned in the rendered banner.
//
// Verb names are checked at word-boundary precision (bannerHasToken)
// rather than plain substring, because several verb names are
// prefixes of others ("rename" / "rename-area") — a naive
// strings.Contains would pass "rename" against a banner that only
// documents "rename-area".
func TestPolicy_HelpBannerCoversVerbs(t *testing.T) {
	banner := captureHelpBanner(t)
	root := cli.NewRootCmd("")

	var failures []string
	for _, c := range root.Commands() {
		name := c.Name()
		if !bannerHasVerb(banner, name) {
			failures = append(failures, name)
		}
	}
	if len(failures) > 0 {
		sort.Strings(failures)
		t.Errorf("verb(s) registered in NewRootCmd but missing from the printHelp() banner (G-0285):\n  %s\n\n"+
			"Add a line under 'Verbs:' in internal/cli/root.go's printHelp() for each.",
			joinFailures(failures))
	}
}

// TestPolicy_HelpBannerCoversFlags mirrors the verb-drift test above
// for value-taking flags (boolean flags are self-evident from a
// verb's own --help and aren't worth enumerating in the root
// banner — same exclusion completion_drift_test.go makes). Every
// registered non-bool flag in the Cobra tree must either appear as a
// literal `--<name>` token somewhere in the banner, or have a
// one-line rationale in helpBannerOptOutFlags.
//
// The check is deliberately banner-wide rather than scoped to a
// specific "Flags for '<verb>'" section: several flags are shared
// across verbs (--format, --area, --priority, --root, ...) and the
// banner documents them once rather than per verb. Presence anywhere
// is what makes the flag AI-discoverable; exact per-verb semantics
// remain `aiwf <verb> --help`'s job.
func TestPolicy_HelpBannerCoversFlags(t *testing.T) {
	banner := captureHelpBanner(t)
	root := cli.NewRootCmd("")

	// helpBannerOptOutFlags names (cmd-path, flag-name) pairs that
	// intentionally have no banner mention. The cmd-path side is
	// optional: an empty string opts the flag out across every
	// command where it appears.
	helpBannerOptOutFlags := map[flagKey]string{
		{cmd: "aiwf add", flag: "body"}:           "free-form inline alternative to --body-file with the same creation-time semantics; documented via 'aiwf add --help'",
		{cmd: "aiwf add", flag: "path-hint"}:      "advisory area-derivation hint, supplementary to --area; documented via 'aiwf add --help'",
		{cmd: "aiwf check", flag: "commit-msg"}:   "internal path used only by the installed commit-msg hook; not an operator-facing flag",
		{cmd: "aiwf check", flag: "since"}:        "advisory base-ref override for the provenance audit; documented via 'aiwf check --help'",
		{cmd: "aiwf init", flag: "enable-hook"}:   "non-interactive consent knob (ADR-0032); documented via 'aiwf init --help'",
		{cmd: "aiwf update", flag: "enable-hook"}: "non-interactive consent knob (ADR-0032); documented via 'aiwf update --help'",
		{cmd: "aiwf init", flag: "scope"}:         "statusline install-scope knob; documented via 'aiwf init --help'",
		{cmd: "aiwf update", flag: "scope"}:       "statusline install-scope knob; documented via 'aiwf update --help'",
		{cmd: "aiwf render", flag: "scope"}:       "reserved (not yet implemented)",
		{cmd: "aiwf show", flag: "history"}:       "integer event-count cap (0=none, -1=all); documented via 'aiwf show --help'",
	}

	var failures []string
	walkCommands(root, func(cmd *cobra.Command) {
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Value.Type() == "bool" {
				return
			}
			if _, ok := helpBannerOptOutFlags[flagKey{cmd: cmd.CommandPath(), flag: f.Name}]; ok {
				return
			}
			if _, ok := helpBannerOptOutFlags[flagKey{flag: f.Name}]; ok {
				return
			}
			if bannerHasToken(banner, "--"+f.Name) {
				return
			}
			failures = append(failures, cmd.CommandPath()+" --"+f.Name)
		})
	})
	if len(failures) > 0 {
		sort.Strings(failures)
		t.Errorf("flag(s) registered in the Cobra tree but missing from the printHelp() banner (G-0285):\n  %s\n\n"+
			"Either mention the flag somewhere in internal/cli/root.go's printHelp() banner, or add an "+
			"entry to helpBannerOptOutFlags in help_banner_drift_test.go with a one-line rationale.",
			joinFailures(failures))
	}
}

// captureHelpBanner runs `aiwf --help` through the real dispatcher
// and returns the printed banner as a string.
func captureHelpBanner(t *testing.T) string {
	t.Helper()
	out := testutil.CaptureStdout(t, func() {
		cli.Execute([]string{"--help"})
	})
	return string(out)
}

// bannerHasVerb reports whether name appears in banner as a Verbs:
// bullet — a line starting with exactly two spaces then name at a
// token boundary. Anchoring on the two-space bullet indent (rather
// than a bare substring search) avoids both false positives from verb
// names that prefix another ("rename" inside "rename-area") and false
// matches inside section headers or flag descriptions.
func bannerHasVerb(banner, name string) bool {
	marker := "\n  " + name
	idx := 0
	for {
		i := indexFrom(banner, marker, idx)
		if i < 0 {
			return false
		}
		end := i + len(marker)
		if end >= len(banner) || !isTokenChar(banner[end]) {
			return true
		}
		idx = i + len(marker)
	}
}

// bannerHasToken reports whether token (e.g. "--body-file") appears
// in banner as a standalone token — flanked by non-token characters
// (or string boundaries) on both sides. Go's RE2-backed regexp has no
// lookaround, so this is done by hand: a plain substring search would
// let "--by" match inside "--by-commit".
func bannerHasToken(banner, token string) bool {
	idx := 0
	for {
		i := indexFrom(banner, token, idx)
		if i < 0 {
			return false
		}
		before := i == 0 || !isTokenChar(banner[i-1])
		end := i + len(token)
		after := end >= len(banner) || !isTokenChar(banner[end])
		if before && after {
			return true
		}
		idx = i + len(token)
	}
}

// isTokenChar reports whether b can appear inside a flag or verb
// name (letters, digits, underscore, hyphen) — used to detect token
// boundaries without regexp lookaround.
func isTokenChar(b byte) bool {
	return b == '-' || b == '_' ||
		(b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// indexFrom finds the first occurrence of sub in s at or after
// offset, returning -1 if absent (mirrors strings.Index but resumable
// mid-string for the token-boundary scan loops above).
func indexFrom(s, sub string, offset int) int {
	if offset >= len(s) {
		return -1
	}
	i := strings.Index(s[offset:], sub)
	if i < 0 {
		return -1
	}
	return offset + i
}
