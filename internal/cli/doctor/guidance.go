package doctor

import (
	"context"
	"fmt"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/initrepo"
	"github.com/23min/aiwf/internal/skills"
	"github.com/23min/aiwf/internal/version"
)

func appendHostGuidanceReport(lines []string, problems []Problem, root string, host config.Host, cfg *config.Config) ([]string, []Problem) {
	ctx := context.Background()
	path, optedOut := "AGENTS.md", !cfg.WireAgentsMd()
	if host == config.HostClaudeCode {
		path, optedOut = "CLAUDE.md", !cfg.WireClaudeMd()
		expected, err := skills.RenderGuidance(version.Current().Version)
		if err != nil { //coverage:ignore compiled-in guidance cannot fail to render
			return lines, append(problems, Problem{Host: host, Path: skills.GuidanceFile, Severity: SeverityWarn, Message: err.Error()})
		}
		status := skills.InspectArtifact(ctx, root, skills.FamilyGuidance, skills.GuidanceFile, expected)
		if status.State != skills.ArtifactCurrent {
			lines, problems = appendArtifactProblem(lines, problems, host, status, SeverityWarn)
		}
	}
	if optedOut {
		return append(lines, fmt.Sprintf("%sopted out (%s unchanged)", label(string(host)+" guidance:"), path)), problems
	}
	plan, err := initrepo.InspectGuidance(ctx, root, host, cfg)
	status := skills.ArtifactStatus{Family: skills.FamilyGuidance, Path: path, State: skills.ArtifactCurrent}
	switch {
	case err != nil:
		status.State, status.Detail = skills.ArtifactBlocked, err.Error()
	case plan.Action == initrepo.ActionSkipped:
		status.State, status.Detail = skills.ArtifactBlocked, "update skipped: "+plan.Detail
	case plan.Action == initrepo.ActionCreated:
		status.State = skills.ArtifactMissing
	case plan.Action == initrepo.ActionUpdated:
		status.State = skills.ArtifactDrifted
	}
	if status.State != skills.ArtifactCurrent {
		return appendArtifactProblem(lines, problems, host, status, SeverityWarn)
	}
	return append(lines, label(string(host)+" guidance:")+"ok (managed root instructions match the writer's plan)"), problems
}
