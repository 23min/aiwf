package doctor

import (
	"context"
	"fmt"
	"slices"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/projectguidance"
)

func appendProjectGuidanceReport(lines []string, problems []Problem, root string, cfg *config.Config, selection config.HostSelection) ([]string, []Problem) {
	prefix := label("project policy:")
	var packs *[]string
	if cfg != nil {
		packs = cfg.Guidance.Packs
	}
	lines = append(lines, prefix+"selected: "+projectguidance.SelectionState(packs))
	if !cfg.GuidanceEnabled() {
		lines = append(lines, prefix+projectguidance.DisabledState)
	}
	report, err := projectguidance.Inspect(context.Background(), root)
	if err != nil {
		message := projectguidance.InspectErrorState(err)
		return append(lines, prefix+message), append(problems, Problem{Path: ".guidance", Severity: SeverityWarn, Message: message})
	}
	revision := report.Revision
	if revision == "" {
		revision = projectguidance.UnknownRevision
	}
	lines = append(lines, prefix+"installed revision: "+revision+"; upstream freshness not checked")
	for _, file := range report.Files {
		lines = append(lines, prefix+"recorded artifact: "+file)
	}
	if report.Pending {
		report.Issues = append(report.Issues, projectguidance.Issue{Path: ".guidance/.aiwf-pending", Detail: projectguidance.PendingState})
	}
	for _, state := range projectguidance.NotInstalledStates(packs, cfg.GuidanceEnabled(), report) {
		report.Issues = append(report.Issues, projectguidance.Issue{Path: "guidance.packs", Detail: state})
	}
	if cfg != nil && cfg.Guidance.Packs != nil && cfg.GuidanceEnabled() && !slices.Contains(report.Files, ".guidance/index.md") {
		report.Issues = append(report.Issues, projectguidance.Issue{Path: ".guidance/index.md", Detail: "selected guidance is not installed; run aiwf update"})
	}
	if cfg != nil && cfg.Guidance.Packs != nil && cfg.GuidanceEnabled() {
		for _, host := range selection.Hosts {
			path, wired := "AGENTS.md", cfg.WireAgentsMd()
			if host == config.HostClaudeCode {
				path, wired = "CLAUDE.md", cfg.WireClaudeMd()
			}
			if wired && !slices.Contains(report.Files, path) {
				report.Issues = append(report.Issues, projectguidance.Issue{Path: path, Detail: "selected host has no recorded engineering guidance route; run aiwf update"})
			}
		}
	}
	for _, issue := range report.Issues {
		host := config.Host("")
		switch issue.Path {
		case "AGENTS.md":
			host = config.HostCodex
		case "CLAUDE.md":
			host = config.HostClaudeCode
		}
		message := issue.Detail
		if host != "" && (!slices.Contains(selection.Hosts, host) || (host == config.HostCodex && !cfg.WireAgentsMd()) || (host == config.HostClaudeCode && !cfg.WireClaudeMd())) {
			message += "; unselected host file is retained unchanged"
		}
		lines = append(lines, fmt.Sprintf("%s%s: %s", prefix, issue.Path, message))
		problems = append(problems, Problem{Host: host, Path: issue.Path, Severity: SeverityWarn, Message: message})
	}
	return lines, problems
}
