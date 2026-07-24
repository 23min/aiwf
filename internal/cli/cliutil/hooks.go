package cliutil

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/skills"
)

// GateHookDecisions returns consent decisions for the hooks it can decide
// (ADR-0032). A hook named in enableHooks (the repeatable --enable-hook flag)
// is enabled without prompting — the non-interactive consent escape hatch,
// mirroring --wire-settings. Otherwise, when an interactive answer is
// available (a TTY present, not --format=json, not --no-prompt) it prompts
// [y/N] naming the hook and its one-line effect, recording the answer.
//
// When no interactive answer is available, the hook is left UNDECIDED — its
// name is omitted from the returned map, never recorded as a false decision.
// This is the load-bearing distinction (G-0446): an omitted hook surfaces as
// a doctor "undecided" warning (yellow in the statusline) so a human notices
// and decides later, whereas a recorded false is an honored decline (green)
// that hides the missed config. The map therefore contains only genuinely
// decided hooks — a missing key means undecided, matching how HookDrift reads
// it (`_, decided := decisions[h.Name]`).
//
// noPrompt forces the non-interactive path even when stdin is a TTY: a
// devcontainer postCreateCommand can allocate a pty with no human behind it,
// so IsTTY alone cannot tell "no answerer" from "answerer present" — the
// caller must say so. Undecided hooks are left undecided; nothing is assumed.
//
// A caller gating against an EXISTING aiwf.yaml (both `aiwf init` on a
// re-run and `aiwf update`) pre-filters hooks to just the ones absent from
// the current hooks: map before calling this, so an already-decided hook is
// never re-prompted or re-defaulted.
func GateHookDecisions(hooks []skills.HookDef, enableHooks []string, formatJSON, noPrompt bool) map[string]bool {
	enable := make(map[string]bool, len(enableHooks))
	for _, name := range enableHooks {
		enable[name] = true
	}
	interactive := !noPrompt && !formatJSON && render.IsTTY(os.Stdin)
	decisions := make(map[string]bool, len(hooks))
	for _, h := range hooks {
		switch {
		case enable[h.Name]:
			decisions[h.Name] = true
		case interactive: //coverage:ignore go test's stdin is never a real TTY, so this arm never taken under any automated test — the same untestable-without-a-fake-tty gap RunStatuslineScaffoldForVersion's identical promptYN branch has (no pty library in this repo's dependencies); the surrounding non-interactive condition is covered by NonTTYLeavesUndecided/FormatJSONLeavesUndecided/NoPromptLeavesUndecided
			decisions[h.Name] = promptYN(fmt.Sprintf("Enable hook %q — %s?", h.Name, h.Description))
		default:
			// Undecided: omit from the map so it surfaces as a doctor
			// warning rather than an honored decline (G-0446).
		}
	}
	return decisions
}

// SyncHookMaterialization materializes each registry hook's script and
// wires or unwires its settings.json entries per its current aiwf.yaml
// decision (ADR-0032: "materialize the script and wire the settings
// entry when true; remove both when false"). Called after the consent
// gate (GateHookDecisions plus the caller's own persistence step) has
// already written decisions to aiwf.yaml — reads them back fresh
// rather than threading a map through, so `aiwf init` and `aiwf
// update` share one call site regardless of how each computed its own
// gating step (init: every registry hook; update: only the
// newly-introduced ones, unioned with what already existed). An
// undecided hook (absent from aiwf.yaml's hooks: map) is left
// untouched — mirrors MaterializeHooks'/HookDrift's identical
// "undecided = not this function's job" convention.
func SyncHookMaterialization(rootDir string, target skills.Target, hooks []skills.HookDef) int {
	if len(hooks) == 0 {
		return ExitOK
	}

	configPath := filepath.Join(rootDir, config.FileName)
	doc, _, err := aiwfyaml.Read(configPath)
	if err != nil {
		Errorf("aiwf: %v\n", err)
		return ExitInternal
	}
	decisions, err := doc.Hooks()
	if err != nil {
		Errorf("aiwf: %v\n", err)
		return ExitInternal
	}

	if err := skills.MaterializeHooks(rootDir, target, hooks, decisions); err != nil {
		Errorf("aiwf: %v\n", err)
		return ExitInternal
	}

	settingsPath := filepath.Join(rootDir, skills.SharedSettingsRelPath)
	for _, h := range hooks {
		enabled, decided := decisions[h.Name]
		if !decided {
			continue
		}
		command := h.Command(target)
		if enabled {
			if _, wireErr := skills.WireHookSettings(settingsPath, command, h.Events); wireErr != nil {
				Errorf("aiwf: %v\n", wireErr)
				return ExitInternal
			}
			continue
		}
		if _, unwireErr := skills.UnwireHookSettings(settingsPath, command); unwireErr != nil {
			Errorf("aiwf: %v\n", unwireErr)
			return ExitInternal
		}
	}
	return ExitOK
}
