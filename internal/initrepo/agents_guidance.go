package initrepo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/pathutil"
	"github.com/23min/aiwf/internal/skills"
)

// ensureAgentsGuidance maintains native Codex instructions only when selected
// by its caller. Opting out preserves even a previously installed block.
func ensureAgentsGuidance(ctx context.Context, root string, cfg *config.Config, dryRun bool) (StepResult, error) {
	const what = "AGENTS.md (guidance)"
	if !cfg.WireAgentsMd() {
		return StepResult{What: what, Action: ActionSkipped, Detail: "guidance.wire_agentsmd is false; existing AGENTS.md is unchanged"}, nil
	}
	files, err := inspectInstructionFiles(ctx, root)
	if err != nil {
		return StepResult{}, fmt.Errorf("checking AGENTS.md guidance safety: %w", err)
	}
	if files.agents.refusal != "" {
		return StepResult{What: what, Action: ActionSkipped, Detail: files.agents.refusal}, nil
	}
	path := filepath.Join(root, "AGENTS.md")
	absent := files.agents.info == nil
	mode := os.FileMode(0o644)
	var content []byte
	if !absent {
		mode = files.agents.info.Mode().Perm()
		content, err = os.ReadFile(path)
		if err != nil {
			return StepResult{}, fmt.Errorf("reading AGENTS.md: %w", err)
		}
	}
	body, err := skills.RenderCodexGuidance()
	if err != nil { //coverage:ignore compiled-in guidance cannot fail to render
		return StepResult{}, fmt.Errorf("rendering AGENTS.md guidance: %w", err)
	}
	rebuilt, err := spliceAgentsGuidance(string(content), string(body))
	if err != nil {
		return StepResult{What: what, Action: ActionSkipped, Detail: fmt.Sprintf("AGENTS.md has ambiguous aiwf guidance markers (%v); repair them to one ordered START/END pair, or remove the markers to append a new block", err)}, nil
	}
	if rebuilt == string(content) {
		return StepResult{What: what, Action: ActionPreserved}, nil
	}
	if !dryRun {
		if err := pathutil.AtomicWriteFile(path, []byte(rebuilt), mode); err != nil {
			return StepResult{}, fmt.Errorf("writing AGENTS.md: %w", err)
		}
	}
	action := ActionUpdated
	if absent {
		action = ActionCreated
	}
	return StepResult{What: what, Action: action, Detail: "wired native guidance"}, nil
}

func spliceAgentsGuidance(content, body string) (string, error) {
	return pathutil.SpliceManagedBlock(content, body, guidanceImportStartMarker, guidanceImportEndMarker, "<!-- aiwf:guidance:")
}
