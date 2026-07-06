package cliutil

import (
	"fmt"
	"os"

	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/skills"
)

// GateHookDecisions returns a consent decision for every hook in the
// registry (ADR-0032): a hook named in enableHooks (the repeatable
// --enable-hook flag) is enabled without prompting — the non-TTY consent
// escape hatch, mirroring --wire-settings; otherwise, absent a TTY (or
// under --format=json), it silently declines rather than hanging on a
// prompt; with a TTY present, it prompts [y/N] naming the hook and its
// one-line effect (default declines).
//
// Used by `aiwf init`, where every registry hook is by definition
// undecided (there is no pre-existing aiwf.yaml to have already recorded a
// decision). A caller gating against an EXISTING aiwf.yaml (`aiwf update`,
// M-0235/AC-3) pre-filters hooks to just the ones absent from the current
// hooks: map before calling this, so an already-decided hook is never
// re-prompted.
func GateHookDecisions(hooks []skills.HookDef, enableHooks []string, formatJSON bool) map[string]bool {
	enable := make(map[string]bool, len(enableHooks))
	for _, name := range enableHooks {
		enable[name] = true
	}
	decisions := make(map[string]bool, len(hooks))
	for _, h := range hooks {
		switch {
		case enable[h.Name]:
			decisions[h.Name] = true
		case !formatJSON && render.IsTTY(os.Stdin): //coverage:ignore go test's stdin is never a real TTY, so this arm never taken under any automated test — the same untestable-without-a-fake-tty gap RunStatuslineScaffoldForVersion's identical promptYN branch has (no pty library in this repo's dependencies); covered by NonTTYDeclinesByDefault/FormatJSONForcesNonInteractive exercising the surrounding condition, not this arm's body
			decisions[h.Name] = promptYN(fmt.Sprintf("Enable hook %q — %s?", h.Name, h.Description))
		default:
			decisions[h.Name] = false
		}
	}
	return decisions
}
