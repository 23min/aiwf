package initrepo

import (
	"context"
	"fmt"

	"github.com/23min/aiwf/internal/config"
)

// InspectGuidance returns the selected root instruction writer's read-only plan.
// It shares the writer's opt-out, marker and alias guards. A preserved result
// describes disk state only; it cannot establish delivery into a model context.
func InspectGuidance(ctx context.Context, root string, host config.Host, cfg *config.Config) (StepResult, error) {
	switch host {
	case config.HostClaudeCode:
		return ensureGuidanceImport(ctx, root, RefreshOptions{DryRun: true, WireClaudeMd: cfg.WireClaudeMd()})
	case config.HostCodex:
		return ensureAgentsGuidance(ctx, root, cfg, true)
	default:
		return StepResult{}, fmt.Errorf("inspecting guidance: %w %q", config.ErrInvalidHost, host)
	}
}
