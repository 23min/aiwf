package doctor

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/projectguidance"
)

func appendProjectGuidanceReport(lines []string, problems []Problem, root string, cfg *config.Config, selection config.HostSelection) ([]string, []Problem) {
	prefix := label("project policy:")
	selected := "not adopted (guidance.packs unset)"
	if cfg != nil && cfg.Guidance.Packs != nil {
		selected = strings.Join(*cfg.Guidance.Packs, ", ")
		if selected == "" {
			selected = "none (explicit empty selection)"
		}
	}
	lines = append(lines, prefix+"selected: "+selected)
	if !cfg.GuidanceEnabled() {
		lines = append(lines, prefix+"maintenance disabled; installed policy remains in use")
	}
	report, err := projectguidance.Inspect(context.Background(), root)
	if err != nil {
		message := "cannot inspect installed policy: " + err.Error()
		return append(lines, prefix+message), append(problems, Problem{Path: ".guidance", Severity: SeverityWarn, Message: message})
	}
	revision := report.Revision
	if revision == "" {
		revision = "unknown (no intact owned index)"
	}
	lines = append(lines, prefix+"installed revision: "+revision+"; upstream freshness not checked")
	for _, file := range report.Files {
		lines = append(lines, prefix+"recorded artifact: "+file)
	}
	if report.Pending {
		report.Issues = append(report.Issues, projectguidance.Issue{Path: ".guidance/.aiwf-pending", Detail: "installation incomplete; rerun aiwf update before using the pack set"})
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
