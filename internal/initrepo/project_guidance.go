package initrepo

import (
	"context"
	"fmt"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/projectguidance"
)

func ensureProjectGuidance(ctx context.Context, root string, cfg *config.Config, selection config.HostSelection, opts RefreshOptions) *StepResult {
	if cfg == nil || !cfg.GuidanceEnabled() || (cfg.Guidance.Packs == nil && opts.SelectGuidance == nil) {
		return nil
	}
	step := &StepResult{What: ".guidance (project engineering guidance)", Action: ActionSkipped}
	if opts.DryRun {
		step.Detail = "would retrieve and validate selected guidance, then install project files and host routing"
		return step
	}
	instructions, err := inspectInstructionFiles(ctx, root)
	if err != nil {
		step.Detail = fmt.Sprintf("guidance incomplete: %v", err)
		return step
	}
	if err = projectguidance.CheckCompatibility(ctx); err != nil {
		step.Detail = fmt.Sprintf("guidance incomplete: %v", err)
		return step
	}
	var hosts []string
	for _, host := range selection.Hosts {
		if host == config.HostClaudeCode && opts.WireClaudeMd {
			if instructions.claude.refusal != "" {
				step.Detail = instructions.claude.refusal
				return step
			}
			hosts = append(hosts, "CLAUDE.md")
		}
		if host == config.HostCodex && cfg.WireAgentsMd() {
			if instructions.agents.refusal != "" {
				step.Detail = instructions.agents.refusal
				return step
			}
			hosts = append(hosts, "AGENTS.md")
		}
	}
	var snapshot *projectguidance.Snapshot
	if opts.SelectGuidance == nil {
		snapshot, err = projectguidance.Retrieve(ctx, cfg.GuidanceSource(), *cfg.Guidance.Packs)
	} else {
		snapshot, err = projectguidance.RetrieveWithSelection(ctx, cfg.GuidanceSource(), func(catalogue projectguidance.Catalogue) ([]string, error) {
			if selectErr := opts.SelectGuidance(ctx, root, catalogue, cfg); selectErr != nil {
				return nil, selectErr
			}
			if cfg.Guidance.Packs == nil {
				return nil, nil
			}
			return *cfg.Guidance.Packs, nil
		})
	}
	if err != nil {
		step.Detail = fmt.Sprintf("guidance incomplete: %v", err)
		return step
	}
	if cfg.Guidance.Packs == nil {
		step.Detail = "no project guidance selected; existing delivery retained"
		return step
	}
	changed, err := projectguidance.Install(ctx, root, snapshot, projectguidance.InstallOptions{Source: cfg.GuidanceSource(), Selected: *cfg.Guidance.Packs, HostFiles: hosts})
	if err != nil {
		step.Detail = fmt.Sprintf("guidance incomplete: %v", err)
		return step
	}
	step.Action = ActionPreserved
	if changed {
		step.Action = ActionUpdated
	}
	step.Detail = "selected project guidance at " + snapshot.Commit
	return step
}
