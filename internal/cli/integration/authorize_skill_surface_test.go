package integration

import (
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/skills"
)

// authorizePlumbingFlags name the flags on `aiwf authorize` that are not
// part of its own surface: every verb registers them, and documenting
// them per-skill would say the same thing nineteen times.
//
// Membership is the exemption, so a flag genuinely specific to authorize
// cannot slip past by omission — it has to be added here deliberately,
// and the reason has to survive review.
var authorizePlumbingFlags = map[string]string{
	"actor":          "registered by every verb; identity resolution is documented once, in the provenance model",
	"root":           "registered by every verb; names the consumer repo, not anything about authorization",
	"format":         "registered by every verb through AddFormatFlags",
	"pretty":         "registered by every verb through AddFormatFlags",
	"correlation-id": "registered by every verb through AddFormatFlags; a diagnostics correlation token",
	"trace":          "registered by every verb through AddFormatFlags; a diagnostic-logging switch",
}

// TestPolicy_AuthorizeSkillDocumentsItsOwnSurface is M-0325/AC-3's skill
// half.
//
// The expectation is computed from the Cobra tree rather than written
// out, which is what keeps this a relationship check instead of a pin on
// how the skill reads: a flag added to the verb without a line in the
// skill turns it red and names the flag, while any rewording of the
// skill that still shows the flag keeps it green. Adding a flag *and* an
// exemption is possible, but that is a visible edit to the map above
// rather than a silent omission.
//
// The needle is a flag name — a stable identifier the CLI already
// commits to — not a phrase, which is why this carries an entry in
// derivedExpectationExemptions rather than falling under D-0070's ban on
// asserting shipped prose.
func TestPolicy_AuthorizeSkillDocumentsItsOwnSurface(t *testing.T) {
	t.Parallel()

	body := authorizeSkillBody(t)

	var missing []string
	forEachAuthorizeFlag(t, func(f *pflag.Flag) {
		if _, plumbing := authorizePlumbingFlags[f.Name]; plumbing {
			return
		}
		if !strings.Contains(body, "--"+f.Name) {
			missing = append(missing, "--"+f.Name)
		}
	})
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("aiwf authorize registers %v, which the aiwf-authorize skill never names.\n"+
			"An assistant reads the skill to learn what the verb can do, so a flag absent from it "+
			"is a capability that ships without being reachable. Document each, or add it to "+
			"authorizePlumbingFlags with a rationale.", missing)
	}
}

// authorizeSkillBody returns the embedded aiwf-authorize SKILL.md as the
// binary ships it, so the assertion reads what materializes into a
// consumer's .claude/ rather than a copy on disk here.
func authorizeSkillBody(t *testing.T) string {
	t.Helper()
	all, err := skills.List()
	if err != nil {
		t.Fatalf("skills.List: %v", err)
	}
	for _, s := range all {
		if s.Name == "aiwf-authorize" {
			return string(s.Content)
		}
	}
	t.Fatal("no aiwf-authorize skill in the embedded set; the locator is stale")
	return ""
}

// forEachAuthorizeFlag visits every flag `aiwf authorize` registers,
// resolved from the real command tree.
func forEachAuthorizeFlag(t *testing.T, visit func(*pflag.Flag)) {
	t.Helper()
	var authorize *cobra.Command
	walkCommands(cli.NewRootCmd(""), func(c *cobra.Command) {
		if c.CommandPath() == "aiwf authorize" {
			authorize = c
		}
	})
	if authorize == nil {
		t.Fatal("no `aiwf authorize` command in the tree; the locator is stale")
	}
	authorize.Flags().VisitAll(visit)
}
