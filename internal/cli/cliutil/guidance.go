package cliutil

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/projectguidance"
	"github.com/23min/aiwf/internal/render"
)

// GuidancePrompt owns the interactive IO boundary shared by init and update.
type GuidancePrompt struct {
	In  io.Reader
	Out io.Writer
}

// GuidanceSelector supplies interactive selection only when a human can answer.
func GuidanceSelector(noPrompt bool) projectguidance.Selector {
	if noPrompt || !render.IsTTY(os.Stdin) {
		return nil
	}
	return (GuidancePrompt{In: os.Stdin, Out: os.Stderr}).Select
}

// Select saves a completed prompt session once. EOF, read/write failure, or
// cancellation leaves both configuration and the caller's selection unchanged.
func (p GuidancePrompt) Select(ctx context.Context, root string, catalogue projectguidance.Catalogue, cfg *config.Config) error {
	matches, err := projectguidance.Detect(ctx, root, catalogue)
	if err != nil {
		return err
	}
	var selected []string
	if cfg.Guidance.Packs != nil {
		selected = slices.Clone(*cfg.Guidance.Packs)
	}
	ignored := slices.Clone(cfg.Guidance.Ignored)
	changed := false
	scanner := bufio.NewScanner(p.In)
	for _, match := range matches {
		if slices.Contains(selected, match.PackID) || slices.Contains(ignored, match.PackID) {
			continue
		}
		for {
			if cancelErr := ctx.Err(); cancelErr != nil {
				return cancelErr
			}
			if _, err = fmt.Fprintf(p.Out, "\n%s — %s\nMatches %s (%s).\nselect (s) / not now (n) / ignore (i) [not now]: ", match.PackID, match.Description, match.File, match.Pattern); err != nil {
				return fmt.Errorf("displaying guidance choice: %w", err)
			}
			if !scanner.Scan() {
				if err = scanner.Err(); err != nil {
					return fmt.Errorf("reading guidance choice: %w", err)
				}
				return fmt.Errorf("guidance selection interrupted; no choices saved: %w", io.EOF)
			}
			answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
			switch answer {
			case "s", "select":
				selected = append(selected, match.PackID)
				changed = true
			case "i", "ignore":
				ignored = append(ignored, match.PackID)
				changed = true
			case "", "n", "not now":
			default:
				continue
			}
			break
		}
	}
	if cancelErr := ctx.Err(); cancelErr != nil {
		return cancelErr
	}
	if !changed {
		return nil
	}
	configPath := filepath.Join(root, config.FileName)
	doc, _, err := aiwfyaml.Read(configPath)
	if err != nil {
		return err
	}
	desired := cfg.Guidance.Packs
	if len(selected) > 0 {
		desired = &selected
	}
	if err := doc.SetGuidanceSelection(desired, ignored); err != nil {
		return err
	}
	if err := doc.Write(configPath); err != nil {
		return err
	}
	cfg.Guidance.Packs = desired
	cfg.Guidance.Ignored = ignored
	return nil
}
