package initrepo

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/projectguidance"
)

func ensureProjectGuidance(ctx context.Context, root string, cfg *config.Config, selection config.HostSelection, opts RefreshOptions) *StepResult {
	if cfg == nil || !cfg.GuidanceEnabled() || (cfg.Guidance.Packs == nil && !opts.AdoptGuidance) {
		return nil
	}
	step := &StepResult{What: ".guidance (project engineering guidance)", Action: ActionSkipped}
	if opts.DryRun {
		step.Detail = "would retrieve and validate selected guidance, then install project files and host routing"
		if opts.AdoptGuidance {
			step.Detail = "would retrieve guidance, adopt applicable packs not listed in guidance.ignored, then install project files and host routing"
		}
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
	selected := cfg.Guidance.Packs
	if opts.AdoptGuidance {
		snapshot, err = projectguidance.RetrieveWithSelection(ctx, cfg.GuidanceSource(), func(catalogue projectguidance.Catalogue) ([]string, error) {
			matches, detectErr := projectguidance.Detect(ctx, root, catalogue)
			if detectErr != nil {
				return nil, detectErr
			}
			selected = projectguidance.Adopt(matches, cfg.Guidance.Packs, cfg.Guidance.Ignored)
			if selected == nil {
				return nil, nil
			}
			return *selected, nil
		})
	} else {
		snapshot, err = projectguidance.Retrieve(ctx, cfg.GuidanceSource(), *cfg.Guidance.Packs)
	}
	if err != nil {
		step.Detail = fmt.Sprintf("guidance incomplete: %v", err)
		return step
	}
	if selected == nil {
		step.Detail = "no project guidance selected; existing delivery retained"
		return step
	}
	missing := ""
	if len(snapshot.Missing) > 0 {
		missing = fmt.Sprintf("; the catalogue no longer has %s: remove it from guidance.packs or move it to guidance.ignored", strings.Join(snapshot.Missing, ", "))
		// Only a run that adopts may retire a pack it cannot replace; any
		// other refresh leaves the installed guidance as it is.
		if !opts.AdoptGuidance {
			step.Detail = "guidance incomplete: installed guidance left unchanged" + missing
			return step
		}
	}
	// An adopted selection is prepared before installation and written only
	// after it, so aiwf.yaml never names a pack the working tree lacks.
	var adopted *aiwfyaml.Doc
	if selected != cfg.Guidance.Packs {
		if adopted, err = guidanceSelectionEdit(root, selected, cfg.Guidance.Ignored); err != nil {
			step.Detail = fmt.Sprintf("guidance incomplete: %v", err) + missing
			return step
		}
	}
	// Packs the catalogue no longer lists stay selected but are not installed,
	// so their unmodified files are retired like any unselected pack's.
	installed := slices.DeleteFunc(slices.Clone(*selected), func(id string) bool { return slices.Contains(snapshot.Missing, id) })
	changed, err := projectguidance.Install(ctx, root, snapshot, projectguidance.InstallOptions{Source: cfg.GuidanceSource(), Selected: installed, HostFiles: hosts})
	if err != nil {
		step.Detail = fmt.Sprintf("guidance incomplete: %v", err) + missing
		return step
	}
	if adopted != nil {
		if err = adopted.Write(filepath.Join(root, config.FileName)); err != nil {
			step.Detail = fmt.Sprintf("guidance incomplete: recording adopted guidance: %v", err) + missing
			return step
		}
		changed = true
	}
	step.Action = ActionPreserved
	if changed {
		step.Action = ActionUpdated
	}
	step.Detail = "selected project guidance at " + snapshot.Commit + missing
	return step
}

// guidanceSelectionEdit returns aiwf.yaml with the selection applied, unwritten.
func guidanceSelectionEdit(root string, selected *[]string, ignored []string) (*aiwfyaml.Doc, error) {
	doc, _, err := aiwfyaml.Read(filepath.Join(root, config.FileName))
	if err != nil {
		return nil, fmt.Errorf("recording adopted guidance: %w", err)
	}
	if err = doc.SetGuidanceSelection(selected, ignored); err != nil {
		return nil, fmt.Errorf("recording adopted guidance: %w", err)
	}
	return doc, nil
}
