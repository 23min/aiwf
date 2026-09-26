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

// GuidanceOutcome is what one run did to project guidance, for the report
// that ends init and update.
type GuidanceOutcome struct {
	// Selection is guidance.packs as recorded after the run.
	Selection *[]string
	// Disabled reports guidance.enabled: false; nothing is retrieved or changed.
	Disabled bool
	// Adopted lists the packs this run added to the recorded selection.
	Adopted []string
	Ignored []string
	// Blocker names why installation did not complete and, where the
	// installer names one, the action that clears it.
	Blocker string
	// Installed is the installed guidance read after the run; it is valid only
	// when InspectError is empty.
	Installed projectguidance.Inspection
	// InspectError words installed guidance that cannot be read.
	InspectError string
}

// projectGuidance runs the guidance step and reports its outcome, reading the
// installed set from disk after the step so the report states what is in place.
// A run that neither adopts nor has a selection, and a dry run, report nothing.
func projectGuidance(ctx context.Context, root string, cfg *config.Config, selection config.HostSelection, opts RefreshOptions) (*StepResult, *GuidanceOutcome) {
	step, outcome := ensureProjectGuidance(ctx, root, cfg, selection, opts)
	if outcome == nil {
		return step, nil
	}
	inspection, err := projectguidance.Inspect(ctx, root)
	if err != nil {
		outcome.InspectError = projectguidance.InspectErrorState(err)
		return step, outcome
	}
	outcome.Installed = inspection
	return step, outcome
}

func ensureProjectGuidance(ctx context.Context, root string, cfg *config.Config, selection config.HostSelection, opts RefreshOptions) (*StepResult, *GuidanceOutcome) {
	if cfg == nil || (cfg.Guidance.Packs == nil && !opts.AdoptGuidance) {
		return nil, nil
	}
	if opts.DryRun {
		return dryRunGuidance(cfg, opts), nil
	}
	outcome := &GuidanceOutcome{Selection: cfg.Guidance.Packs, Ignored: cfg.Guidance.Ignored, Disabled: !cfg.GuidanceEnabled()}
	if outcome.Disabled {
		return nil, outcome
	}
	step := &StepResult{What: ".guidance (project engineering guidance)", Action: ActionSkipped}
	missing := ""
	// A blocked run installs nothing; every block after retrieval also
	// carries the missing-pack note.
	blocked := func(detail string) (*StepResult, *GuidanceOutcome) {
		outcome.Blocker = detail
		step.Detail = "guidance incomplete: " + detail + missing
		return step, outcome
	}
	instructions, err := inspectInstructionFiles(ctx, root)
	if err != nil {
		return blocked(err.Error())
	}
	if err = projectguidance.CheckCompatibility(ctx); err != nil {
		return blocked(err.Error())
	}
	var hosts []string
	for _, host := range selection.Hosts {
		if host == config.HostClaudeCode && opts.WireClaudeMd {
			if instructions.claude.refusal != "" {
				return blocked(instructions.claude.refusal)
			}
			hosts = append(hosts, "CLAUDE.md")
		}
		if host == config.HostCodex && cfg.WireAgentsMd() {
			if instructions.agents.refusal != "" {
				return blocked(instructions.agents.refusal)
			}
			hosts = append(hosts, "AGENTS.md")
		}
	}
	var snapshot *projectguidance.Snapshot
	selected := cfg.Guidance.Packs
	var added []string
	if opts.AdoptGuidance {
		snapshot, err = projectguidance.RetrieveWithSelection(ctx, cfg.GuidanceSource(), func(catalogue projectguidance.Catalogue) ([]string, error) {
			matches, detectErr := projectguidance.Detect(ctx, root, catalogue)
			if detectErr != nil {
				return nil, detectErr
			}
			selected, added = projectguidance.Adopt(matches, cfg.Guidance.Packs, cfg.Guidance.Ignored)
			if selected == nil {
				return nil, nil
			}
			return *selected, nil
		})
	} else {
		snapshot, err = projectguidance.Retrieve(ctx, cfg.GuidanceSource(), *cfg.Guidance.Packs)
	}
	if err != nil {
		return blocked(err.Error())
	}
	if selected == nil {
		step.Detail = "no project guidance selected; existing delivery retained"
		return step, outcome
	}
	if len(snapshot.Missing) > 0 {
		missing = fmt.Sprintf("; the catalogue no longer has %s: remove it from guidance.packs or move it to guidance.ignored", strings.Join(snapshot.Missing, ", "))
		// Only a run that adopts may retire a pack it cannot replace; any
		// other refresh leaves the installed guidance as it is.
		if !opts.AdoptGuidance {
			return blocked("installed guidance left unchanged")
		}
	}
	// An adopted selection is prepared before installation and written only
	// after it, so aiwf.yaml never names a pack the working tree lacks.
	var adopted *aiwfyaml.Doc
	if selected != cfg.Guidance.Packs {
		if adopted, err = guidanceSelectionEdit(root, selected, cfg.Guidance.Ignored); err != nil {
			return blocked(err.Error())
		}
	}
	// Packs the catalogue no longer lists stay selected but are not installed,
	// so their unmodified files are retired like any unselected pack's.
	installed := slices.DeleteFunc(slices.Clone(*selected), func(id string) bool { return slices.Contains(snapshot.Missing, id) })
	changed, err := projectguidance.Install(ctx, root, snapshot, projectguidance.InstallOptions{Source: cfg.GuidanceSource(), Selected: installed, HostFiles: hosts})
	if err != nil {
		return blocked(err.Error())
	}
	if adopted != nil {
		if err = adopted.Write(filepath.Join(root, config.FileName)); err != nil {
			return blocked("recording adopted guidance: " + err.Error())
		}
		changed = true
		outcome.Selection, outcome.Adopted = selected, added
	}
	step.Action = ActionPreserved
	if changed {
		step.Action = ActionUpdated
	}
	step.Detail = "selected project guidance at " + snapshot.Commit + missing
	return step, outcome
}

// dryRunGuidance previews the guidance step; disabled maintenance has
// nothing to preview.
func dryRunGuidance(cfg *config.Config, opts RefreshOptions) *StepResult {
	if !cfg.GuidanceEnabled() {
		return nil
	}
	step := &StepResult{What: ".guidance (project engineering guidance)", Action: ActionSkipped}
	step.Detail = "would retrieve and validate selected guidance, then install project files and host routing"
	if opts.AdoptGuidance {
		step.Detail = "would retrieve guidance, adopt applicable packs not listed in guidance.ignored, then install project files and host routing"
	}
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
