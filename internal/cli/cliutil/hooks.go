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
		fmt.Fprintf(os.Stderr, "aiwf: %v\n", err)
		return ExitInternal
	}
	decisions, err := doc.Hooks()
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf: %v\n", err)
		return ExitInternal
	}

	if err := skills.MaterializeHooks(rootDir, target, hooks, decisions); err != nil {
		fmt.Fprintf(os.Stderr, "aiwf: %v\n", err)
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
				fmt.Fprintf(os.Stderr, "aiwf: %v\n", wireErr)
				return ExitInternal
			}
			continue
		}
		if _, unwireErr := skills.UnwireHookSettings(settingsPath, command); unwireErr != nil {
			fmt.Fprintf(os.Stderr, "aiwf: %v\n", unwireErr)
			return ExitInternal
		}
	}
	return ExitOK
}
