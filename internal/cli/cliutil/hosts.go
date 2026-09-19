package cliutil

import (
	"strings"

	"github.com/23min/aiwf/internal/config"
)

// PrintHostSelection reports the effective host set and its selection source.
func PrintHostSelection(selection config.HostSelection) {
	names := make([]string, len(selection.Hosts))
	for i, host := range selection.Hosts {
		names[i] = string(host)
	}
	value := strings.Join(names, ", ")
	if value == "" {
		value = "none"
	}
	Printf("Hosts (%s): %s\n", selection.Source, value)
}

// HostSetupHelp is the shared host configuration reference for lifecycle help.
const HostSetupHelp = "\n\n" +
	"Host setup (aiwf.yaml):\n" +
	"Omit hosts or use null to detect executable claude and codex commands on PATH.\n" +
	"Detection does not launch either assistant or save machine state to aiwf.yaml.\n" +
	"An explicit list overrides detection, even if the commands are unavailable.\n" +
	"Select both hosts:\n" +
	"```yaml\n" +
	"hosts: [claude-code, codex]\n" +
	"```\n" +
	"Select no host artifacts while keeping core aiwf setup and Git hooks:\n" +
	"```yaml\n" +
	"hosts: []\n" +
	"```\n" +
	"Customize automatic detection: manage root instructions yourself and choose\n" +
	"where aiwf creates worktrees:\n" +
	"```yaml\n" +
	"hosts: null\n" +
	"guidance:\n" +
	"  wire_claudemd: false\n" +
	"  wire_agentsmd: false\n" +
	"worktree:\n" +
	"  dir: .worktrees\n" +
	"```\n" +
	"Guidance wiring defaults to true for each selected host. Keep personal rules\n" +
	"outside aiwf's managed blocks. Symlinked or conflicting root instructions are\n" +
	"left untouched with a reason in the ledger; repair the condition before update.\n" +
	"\n" +
	"Claude artifacts use .claude/skills, .claude/agents, .claude/templates and a\n" +
	"CLAUDE.md import. Codex uses .agents/skills, .agents/aiwf/templates and native\n" +
	"AGENTS.md guidance. Codex role TOML, lifecycle hooks and statusline are not\n" +
	"installed. Claude hooks and statusline retain their separate consent rules.\n" +
	"\n" +
	"Init, update and doctor report the effective hosts and selection source.\n" +
	"Upgrade delegates refresh to the new binary's update command. Worktree add\n" +
	"uses the new checkout's configuration; --format=json includes host_selection\n" +
	"and the artifact steps. When a host leaves the detected set or is explicitly\n" +
	"deselected, retain its files without refresh. Disk checks do not prove delivery\n" +
	"into a model context.\n" +
	"\n" +
	"Worktrees default to .claude/worktrees for either host. Set worktree.dir or\n" +
	"pass an explicit path to worktree add to override placement. Use separate\n" +
	"worktrees for simultaneous Claude and Codex implementation sessions.\n"
