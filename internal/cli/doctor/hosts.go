package doctor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/initrepo"
	"github.com/23min/aiwf/internal/skills"
)

func appendHostsReport(lines []string, problems []Problem, root string, cfg *config.Config, selection config.HostSelection) ([]string, []Problem) {
	names := make([]string, len(selection.Hosts))
	for i, host := range selection.Hosts {
		names[i] = string(host)
	}
	value := strings.Join(names, ", ")
	if value == "" {
		value = "none"
	}
	lines = append(lines, fmt.Sprintf("%s%s (%s)", label("hosts:"), value, selection.Source))
	for _, step := range initrepo.RetainedHostSteps(root, selection) {
		lines = append(lines, fmt.Sprintf("%s%s: %s (%s)", label("retained:"), step.What, step.Action, step.Detail))
	}
	for _, host := range selection.Hosts {
		command := "codex"
		if host == config.HostClaudeCode {
			command = "claude"
		}
		if _, err := exec.LookPath(command); err != nil {
			val := fmt.Sprintf("%s: executable %s unavailable on PATH; install it or adjust hosts in aiwf.yaml; disk artifacts are checked independently", host, command)
			lines = append(lines, label("host-tool:")+val)
			problems = append(problems, Problem{Host: host, Severity: SeverityWarn, Message: val})
		}
		lines, problems = appendHostArtifactsReport(lines, problems, root, host, cfg)
		lines, problems = appendHostGuidanceReport(lines, problems, root, host, cfg)
	}
	lines = append(lines, label("host-context:")+"disk checks do not establish delivery of instructions into a model's context")
	return lines, problems
}

func hostTarget(host config.Host) skills.Target {
	if host == config.HostClaudeCode {
		return skills.ClaudeTarget
	}
	return skills.CodexTarget()
}

func configuredAgentTiers(cfg *config.Config) map[string]skills.AgentTier {
	tiers := map[string]skills.AgentTier{}
	if cfg != nil {
		for name, agent := range cfg.Agents {
			tiers[name] = skills.AgentTier{Model: agent.Model, Effort: agent.Effort}
		}
	}
	return tiers
}

func appendHostArtifactsReport(lines []string, problems []Problem, root string, host config.Host, cfg *config.Config) ([]string, []Problem) {
	statuses, err := skills.InspectArtifacts(context.Background(), root, hostTarget(host), configuredAgentTiers(cfg))
	if err != nil { //coverage:ignore known built-in targets and compiled-in sources cannot fail to render
		val := fmt.Sprintf("%s: could not render expected artifacts: %v", host, err)
		return append(lines, label("artifacts:")+val), append(problems, Problem{Host: host, Severity: SeverityError, Message: val})
	}
	for _, family := range []skills.ArtifactFamily{skills.FamilySkills, skills.FamilyRituals, skills.FamilyAgents, skills.FamilyTemplates} {
		count, findings := 0, 0
		for _, status := range statuses {
			if status.Family != family {
				continue
			}
			count++
			if status.State != skills.ArtifactCurrent {
				findings++
				severity := SeverityWarn
				if family == skills.FamilySkills {
					severity = SeverityError
				}
				lines, problems = appendArtifactProblem(lines, problems, host, status, severity)
			}
		}
		if count > 0 && findings == 0 {
			lines = append(lines, fmt.Sprintf("%sok (%d artifacts materialized, byte-equal to rendered definitions)", label(string(host)+" "+string(family)+":"), count))
		}
	}
	lines = append(lines, subIndent+"managed by aiwf; `aiwf update` refreshes generated files — do not hand-edit")
	return lines, problems
}

func appendArtifactProblem(lines []string, problems []Problem, host config.Host, status skills.ArtifactStatus, severity Severity) ([]string, []Problem) {
	state := string(status.State)
	if status.State == skills.ArtifactMissing {
		state += " (not materialized)"
	}
	remediation := "run `aiwf update` to refresh"
	if status.State == skills.ArtifactBlocked {
		remediation = "resolve the reported path or marker condition, then run `aiwf update`"
	}
	val := fmt.Sprintf("%s %s: %s %s; %s", host, status.Family, state, status.Path, remediation)
	if status.Detail != "" {
		val += "; " + status.Detail
	}
	return append(lines, val), append(problems, Problem{Host: host, Path: status.Path, Severity: severity, Message: val})
}
