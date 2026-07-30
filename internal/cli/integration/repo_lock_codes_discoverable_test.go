package integration

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
)

// repo_lock_codes_discoverable_test.go — G-0467. The repo-lock refusal
// codes are a machine-consumable contract: a caller scripting
// --format=json switches on them to tell retryable contention from a
// terminal failure. CLAUDE.md's AI-discoverability principle says a
// kernel surface reachable only from source is undocumented, so the
// root banner documents both — and this test is what stops the banner
// and the constants drifting apart in either direction.
//
// PolicyFindingCodesAreDiscoverable does not cover them: it enumerates
// codes out of internal/check/ and internal/contractcheck/ only, so it
// structurally cannot see a constant declared in internal/cli/cliutil.
//
// The assertion is section-scoped rather than a banner-wide grep, per
// CLAUDE.md §"Substring assertions are not structural assertions": a
// code named anywhere in a 100-line banner proves it exists somewhere,
// not that a reader looking up concurrent-invocation behavior will
// find it.
func TestRepoLockCodesDocumentedInHelpBanner(t *testing.T) {
	section := concurrencySection(t, captureHelpBanner(t))

	for _, code := range []string{cliutil.CodeRepoLockBusy, cliutil.CodeRepoLockAcquireFailed} {
		if !strings.Contains(section, code) {
			t.Errorf("repo-lock code %q is not named in the root help banner's Concurrency section.\n"+
				"Add it to printHelp() in internal/cli/root.go — a --format=json consumer has no other\n"+
				"documented channel for it.\nSection was:\n%s", code, section)
		}
	}
}

// concurrencySection returns the body of the banner's "Concurrency:"
// block — every line from the heading up to the next unindented line,
// which is how printHelp separates its sections. It fails the test
// when the heading is absent, so deleting the whole block is caught
// rather than silently yielding an empty haystack.
func concurrencySection(t *testing.T, banner string) string {
	t.Helper()
	const heading = "Concurrency:"

	lines := strings.Split(banner, "\n")
	start := -1
	for i, line := range lines {
		if line == heading {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("root help banner has no %q section; printHelp() in internal/cli/root.go must document\n"+
			"the repo-lock refusal codes somewhere a reader can find them.\nBanner was:\n%s", heading, banner)
	}

	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		line := lines[i]
		if line != "" && !strings.HasPrefix(line, " ") {
			end = i
			break
		}
	}
	return strings.Join(lines[start:end], "\n")
}
