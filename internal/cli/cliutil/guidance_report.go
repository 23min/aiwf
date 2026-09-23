package cliutil

import (
	"context"
	"fmt"
	"io"
	"slices"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/projectguidance"
)

func reportGuidance(ctx context.Context, root string, catalogue projectguidance.Catalogue, cfg *config.Config, out io.Writer) error {
	matches, err := projectguidance.Detect(ctx, root, catalogue)
	if err != nil {
		return err
	}
	var selected []string
	if cfg.Guidance.Packs != nil {
		selected = *cfg.Guidance.Packs
	}
	for _, id := range selected {
		if slices.ContainsFunc(matches, func(m projectguidance.Match) bool { return m.PackID == id }) {
			continue
		}
		if _, err := fmt.Fprintf(out, "Selected guidance %s has no current matching files; retained.\n", id); err != nil {
			return fmt.Errorf("reporting selected guidance: %w", err)
		}
	}
	suggested := false
	for _, match := range matches {
		if slices.Contains(selected, match.PackID) || slices.Contains(cfg.Guidance.Ignored, match.PackID) {
			continue
		}
		if _, err := fmt.Fprintf(out, "Suggested guidance %s — %s; matches %s (%s).\n", match.PackID, match.Description, match.File, match.Pattern); err != nil {
			return fmt.Errorf("reporting guidance suggestion: %w", err)
		}
		suggested = true
	}
	if suggested {
		if _, err := fmt.Fprintln(out, "To select or ignore suggestions, edit guidance.packs or guidance.ignored in aiwf.yaml."); err != nil {
			return fmt.Errorf("reporting guidance selection instructions: %w", err)
		}
	}
	return nil
}
