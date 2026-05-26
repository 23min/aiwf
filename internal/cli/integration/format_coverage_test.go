package integration

import (
	"sort"
	"testing"

	"github.com/spf13/cobra"

	cli "github.com/23min/aiwf/internal/cli"
)

// TestFormatFlagUniformRollout_AC4 is M-0143/AC-4 (decision A2): every
// Runnable leaf command in the assembled command tree either registers a
// --format flag or is named in formatExempt with a rationale. A new
// mutating verb shipped without --format fails here unless consciously
// exempted — the mechanical chokepoint that pins the uniform
// --format=json rollout across mutating verbs.
func TestFormatFlagUniformRollout_AC4(t *testing.T) {
	t.Parallel()

	// Runnable commands that legitimately lack --format, each with a
	// rationale. These are utility / setup / read commands that do not
	// route a verb outcome through cliutil.FinishVerb (the envelope
	// chokepoint) or whose JSON surface is deferred elsewhere.
	formatExempt := map[string]string{
		"aiwf":         "root; --version only, dispatches to children",
		"aiwf version": "prints a version line; no verb-outcome envelope",
		"aiwf whoami":  "prints the resolved identity; no verb-outcome envelope",
		"aiwf init":    "one-shot scaffolding; no verb-outcome envelope",
		"aiwf update":  "artifact refresh; no verb-outcome envelope",
		"aiwf upgrade": "self-update; no verb-outcome envelope",
		"aiwf doctor":  "JSON envelope deferred to G-0070",
		"aiwf archive": "sweep verb; does not route through FinishVerb",

		// Mutating verbs with bespoke (non-FinishVerb) output paths.
		// D-0013's A2 scope is the shared FinishVerb/DecorateAndFinish
		// chokepoint; these don't route through it, so the uniform
		// rollout doesn't reach them. JSON wiring tracked in G-0169.
		"aiwf import":  "bespoke multi-entity output; not via FinishVerb — JSON envelope tracked in G-0169",
		"aiwf rewidth": "migration verb with bespoke multi-commit output; not via FinishVerb — G-0169",

		// Read / generate commands — JSON wiring is a separate concern
		// from the mutating-verb (FinishVerb) rollout this AC pins.
		"aiwf contract recipes":     "read/list display; not a mutating-verb outcome — G-0169",
		"aiwf contract recipe show": "read/display; not a mutating-verb outcome — G-0169",
		"aiwf render roadmap":       "generate verb; emits the roadmap artifact, not a verb-outcome envelope",
	}
	for _, sh := range []string{"", " bash", " zsh", " fish", " powershell"} {
		formatExempt["aiwf completion"+sh] = "Cobra completion script generator"
	}

	root := cli.NewRootCmd()
	var missing []string
	walkCommands(root, func(cmd *cobra.Command) {
		if !cmd.Runnable() {
			return // non-Runnable parent (e.g. contract, milestone, contract recipe)
		}
		if cmd.Name() == "help" {
			return // Cobra-generated help command on every parent
		}
		if _, ok := formatExempt[cmd.CommandPath()]; ok {
			return
		}
		if cmd.Flags().Lookup("format") == nil {
			missing = append(missing, cmd.CommandPath())
		}
	})

	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf("Runnable commands missing a --format flag (M-0143 / D-0013, decision A2 uniform rollout):\n  %s\n\n"+
			"Wire --format via cliutil.AddFormatFlags in the command's builder, or add an\n"+
			"entry to formatExempt above with a one-line rationale.", joinFailures(missing))
	}
}
