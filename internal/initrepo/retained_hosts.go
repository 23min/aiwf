package initrepo

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/23min/aiwf/internal/config"
)

// RetainedHostSteps reports unselected host paths without inspecting their
// contents or following links. Presence does not imply an aiwf-owned or healthy
// installation; these paths may include the user's own files.
func RetainedHostSteps(root string, selection config.HostSelection) []StepResult {
	var steps []StepResult
	for _, host := range []struct {
		name  config.Host
		paths []string
	}{
		{config.HostClaudeCode, []string{".claude", "CLAUDE.md"}},
		{config.HostCodex, []string{".agents", "AGENTS.md"}},
	} {
		if slices.Contains(selection.Hosts, host.name) {
			continue
		}
		for _, relative := range host.paths {
			_, err := os.Lstat(filepath.Join(root, relative))
			if os.IsNotExist(err) {
				continue
			}
			step := StepResult{What: relative, Action: ActionPreserved, Detail: fmt.Sprintf("%s not selected; retained without refresh", host.name)}
			if err != nil {
				step.Action = ActionSkipped
				step.Detail = fmt.Sprintf("%s not selected; could not inspect path: %v; left untouched", host.name, err)
			}
			steps = append(steps, step)
		}
	}
	return steps
}
