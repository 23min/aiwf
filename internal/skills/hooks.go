package skills

import "sort"

// HookDef describes one Claude Code hook aiwf can materialize and wire into
// a consumer's `.claude/settings.json`, gated by per-hook consent recorded
// in aiwf.yaml's `hooks:` map (ADR-0032). Name is the registry key —
// matches `hooks.<name>` in aiwf.yaml and the map key config.Config.Hooks
// uses, and doubles as the on-disk filename under Target.HooksDir (mirrors
// how Skill.Name for agents/templates already carries its own extension).
// Description is the one-line effect shown in the consent prompt at
// `aiwf init`/`aiwf update`. Content is the script's bytes, materialized
// verbatim when the hook is enabled.
type HookDef struct {
	Name        string
	Description string
	Content     []byte
}

// ShippedHooks is the registry of hooks aiwf currently ships. Empty until a
// milestone registers its first concrete hook (M-0236) — the consent-gating
// machinery this registry feeds (M-0235) is built and tested ahead of any
// real entry, via an explicit registry parameter callers can substitute in
// tests, mirroring how ListRitualAgents/AgentNames back Config.Agents
// validation without config itself depending on skills.
var ShippedHooks = []HookDef{}

// HookNamesFrom returns the sorted names of hooks, the derived form callers
// validating aiwf.yaml's `hooks:` map keys or building shell completion need.
// Takes the registry explicitly (rather than always reading ShippedHooks)
// so tests can exercise a non-empty registry ahead of any real hook landing.
func HookNamesFrom(hooks []HookDef) []string {
	names := make([]string, 0, len(hooks))
	for _, h := range hooks {
		names = append(names, h.Name)
	}
	sort.Strings(names)
	return names
}

// Command returns the command string a materialized hook wires into
// settings.json under target — the single source of truth for that
// convention, so HookDrift's "wired" check and a future WireHookSettings
// caller (M-0236) always derive the identical string rather than each
// reconstructing it independently.
func (h HookDef) Command(target Target) string {
	return target.HooksDir + "/" + h.Name
}
