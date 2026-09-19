package initrepo

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/pathutil"
	"github.com/23min/aiwf/internal/skills"
	"github.com/23min/aiwf/internal/version"
)

// ensureAgentsGuidance maintains native Codex instructions only when selected
// by its caller. Opting out preserves even a previously installed block.
func ensureAgentsGuidance(ctx context.Context, root string, cfg *config.Config, dryRun bool) (StepResult, error) {
	const what = "AGENTS.md (guidance)"
	if !cfg.WireAgentsMd() {
		return StepResult{What: what, Action: ActionSkipped, Detail: "guidance.wire_agentsmd is false; existing AGENTS.md is unchanged"}, nil
	}
	if err := ctx.Err(); err != nil {
		return StepResult{}, fmt.Errorf("maintaining AGENTS.md: %w", err)
	}
	path := filepath.Join(root, "AGENTS.md")
	info, err := os.Lstat(path)
	absent := errors.Is(err, fs.ErrNotExist)
	if err != nil && !absent {
		return StepResult{}, fmt.Errorf("inspecting AGENTS.md: %w", err)
	}
	mode := fs.FileMode(0o644)
	var content []byte
	if !absent {
		if !info.Mode().IsRegular() {
			return StepResult{What: what, Action: ActionSkipped, Detail: "AGENTS.md is not a regular file; replace it with a regular file to enable managed guidance"}, nil
		}
		mode = info.Mode().Perm()
		content, err = os.ReadFile(path)
		if err != nil {
			return StepResult{}, fmt.Errorf("reading AGENTS.md: %w", err)
		}
	}
	body, err := skills.RenderCodexGuidance(version.Current().Version)
	if err != nil {
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

// spliceAgentsGuidance owns whole marker lines and their enclosed content.
// Offsets preserve every byte outside that span, including the end line's
// delimiter. Only standalone markers count; quoted prose is ordinary text.
func spliceAgentsGuidance(content, body string) (string, error) {
	start, end := -1, -1
	offset := 0
	for _, line := range strings.SplitAfter(content, "\n") {
		trimmed := strings.TrimSpace(line)
		switch trimmed {
		case guidanceImportStartMarker:
			if start >= 0 {
				return "", errors.New("duplicate START")
			}
			start = offset
		case guidanceImportEndMarker:
			if end >= 0 {
				return "", errors.New("duplicate END")
			}
			end = offset + len(strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"))
		default:
			if strings.HasPrefix(trimmed, "<!-- aiwf:guidance:") && strings.HasSuffix(trimmed, "-->") {
				return "", errors.New("unrecognized marker")
			}
		}
		offset += len(line)
	}
	block := guidanceImportStartMarker + "\n" + strings.TrimSuffix(body, "\n") + "\n" + guidanceImportEndMarker
	switch {
	case start < 0 && end < 0:
		if content == "" {
			return block + "\n", nil
		}
		separator := "\n\n"
		if strings.HasSuffix(content, "\n") {
			separator = "\n"
		}
		return content + separator + block + "\n", nil
	case start < 0 || end < 0 || start >= end:
		return "", errors.New("missing or reversed START/END")
	default:
		return content[:start] + block + content[end:], nil
	}
}
