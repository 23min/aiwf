package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/pathutil"
)

// healthFile is the ai-dotfiles per-producer health schema: one file per
// producer at .claude/health.<source>.json, unioned by the statusline.
// Empty findings (or no file) means healthy.
type healthFile struct {
	GeneratedAt string          `json:"generated_at"`
	Findings    []healthFinding `json:"findings"`
}

type healthFinding struct {
	Source   string `json:"source"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// healthFileFrom maps doctor's problems onto the health schema. Pure and
// deterministic (generatedAt is passed in) so the mapping is unit-tested
// without a clock or filesystem.
func healthFileFrom(ps []Problem, generatedAt string) healthFile {
	findings := make([]healthFinding, 0, len(ps))
	for _, p := range ps {
		findings = append(findings, healthFinding{Source: "aiwf", Severity: string(p.Severity), Message: p.Message})
	}
	return healthFile{GeneratedAt: generatedAt, Findings: findings}
}

// WriteHealth writes .claude/health.aiwf.json from doctor's warnings and
// errors, in the fixed ai-dotfiles schema. The write is atomic and lands
// in the main checkout's .claude/ even when rootDir is a linked worktree,
// so one file serves every worktree. generatedAt (ISO 8601 UTC) is passed
// in so this stays wall-clock-free.
func WriteHealth(ctx context.Context, rootDir, generatedAt string, opts DoctorOptions) error {
	data, err := json.MarshalIndent(healthFileFrom(Problems(rootDir, opts), generatedAt), "", "  ")
	if err != nil {
		return fmt.Errorf("encoding health.aiwf.json: %w", err) //coverage:ignore MarshalIndent of this fixed shape cannot fail
	}
	root, err := gitops.MainCheckoutRoot(ctx, rootDir)
	if err != nil {
		return fmt.Errorf("resolving main checkout: %w", err)
	}
	claudeDir := filepath.Join(root, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", claudeDir, err) //coverage:ignore MkdirAll fails only on filesystem faults
	}
	dest := filepath.Join(claudeDir, "health.aiwf.json")
	if err := pathutil.AtomicWriteFile(dest, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", dest, err) //coverage:ignore AtomicWriteFile fails only on filesystem faults
	}
	return nil
}
