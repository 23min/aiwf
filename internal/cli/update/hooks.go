package update

import (
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/skills"
)

// completeHookNames offers the shipped hook registry's names for
// `--enable-hook <TAB>`. Empty (no completions) until a milestone registers
// the first concrete hook (M-0236) — mirrors initcmd's identical helper.
func completeHookNames(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return skills.HookNamesFrom(skills.ShippedHooks), cobra.ShellCompDirectiveNoFileComp
}

// gateAndSyncHookDecisions runs the consent gate (ADR-0032) over only the
// registry hooks ABSENT from the existing aiwf.yaml's hooks: map — an
// already-decided hook (present with an explicit enabled: true/false) is
// never re-prompted, regardless of its recorded value. The union of the
// existing decisions and the newly gated ones is spliced back via
// aiwfyaml's surgical hooks: block writer, so a decision for a hook not
// named in this run's registry (e.g. one since removed from it) survives
// untouched rather than being silently dropped.
func gateAndSyncHookDecisions(rootDir string, hooks []skills.HookDef, enableHooks []string) int {
	configPath := filepath.Join(rootDir, config.FileName)
	doc, _, err := aiwfyaml.Read(configPath)
	if err != nil {
		cliutil.Errorf("aiwf update: %v\n", err)
		return cliutil.ExitInternal
	}

	existing, err := doc.Hooks()
	if err != nil {
		cliutil.Errorf("aiwf update: %v\n", err)
		return cliutil.ExitInternal
	}

	var newHooks []skills.HookDef
	for _, h := range hooks {
		if _, decided := existing[h.Name]; !decided {
			newHooks = append(newHooks, h)
		}
	}

	newDecisions := cliutil.GateHookDecisions(newHooks, enableHooks, false)

	union := make(map[string]bool, len(existing)+len(newDecisions))
	for name, enabled := range existing {
		union[name] = enabled
	}
	for name, enabled := range newDecisions {
		union[name] = enabled
	}

	doc.SetHooks(union)
	if err := doc.Write(configPath); err != nil { //coverage:ignore the preceding Read already succeeded against the same path; only external interference (disk failure, permission change between the two calls) reaches this, not any code path this binary's own control flow produces
		cliutil.Errorf("aiwf update: %v\n", err)
		return cliutil.ExitInternal
	}

	for _, h := range newHooks {
		state := "declined"
		if newDecisions[h.Name] {
			state = "enabled"
		}
		cliutil.Printf("aiwf update: hook %q — %s (new)\n", h.Name, state)
	}
	return cliutil.ExitOK
}
