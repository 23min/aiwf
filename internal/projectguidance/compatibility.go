package projectguidance

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ErrIncompatibleDelivery means project handover cannot safely replace legacy delivery.
var ErrIncompatibleDelivery = errors.New("incompatible ai-dotfiles guidance delivery")

// CheckCompatibility delegates routing policy to the installed ai-dotfiles checker.
// Without it, only an environment with no recognized legacy delivery may proceed.
func CheckCompatibility(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if checker, err := exec.LookPath("dotfiles-guidance-check"); err == nil {
		checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(checkCtx, checker)
		cmd.WaitDelay = time.Second
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w: dotfiles-guidance-check failed: %w; %s; update/reconcile ai-dotfiles in this environment, then retry", ErrIncompatibleDelivery, err, strings.TrimSpace(string(output)))
		}
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("%w: cannot locate personal instructions: %w", ErrIncompatibleDelivery, err)
	}
	claude, codex := os.Getenv("CLAUDE_CONFIG_DIR"), os.Getenv("CODEX_HOME")
	if claude == "" {
		claude = filepath.Join(home, ".claude")
	}
	if codex == "" {
		codex = filepath.Join(home, ".codex")
	}
	override := filepath.Join(codex, "AGENTS.override.md")
	content, err := compatibilityInstructions(override)
	if err != nil {
		return err
	}
	codexFile := filepath.Join(codex, "AGENTS.md")
	if len(content) > 0 {
		codexFile = override
	}
	for _, name := range []string{filepath.Join(claude, "CLAUDE.md"), codexFile} {
		content, err := compatibilityInstructions(name)
		if err != nil {
			return err
		}
		if strings.Contains(string(content), "ai-dotfiles") || strings.Contains(string(content), ".agents/guidance/") {
			return missingChecker(name)
		}
	}
	launchers := []string{filepath.Join(home, ".local", "bin", "dotfiles-sync")}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		launchers = append(launchers, filepath.Join(dir, "dotfiles-sync"))
	}
	for _, launcher := range launchers {
		if _, err := os.Lstat(launcher); err == nil {
			return missingChecker(launcher)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%w: cannot inspect %s: %w", ErrIncompatibleDelivery, launcher, err)
		}
	}
	return nil
}

func missingChecker(path string) error {
	return fmt.Errorf("%w: detected %s but dotfiles-guidance-check is unavailable; update/reconcile ai-dotfiles and make its checker available on PATH, then retry", ErrIncompatibleDelivery, path)
}

func compatibilityInstructions(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%w: cannot inspect %s: %w", ErrIncompatibleDelivery, path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		info, err = os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("%w: cannot resolve %s: %w", ErrIncompatibleDelivery, path, err)
		}
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %s is not a regular instruction file", ErrIncompatibleDelivery, path)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot read %s; reconcile personal instructions: %w", ErrIncompatibleDelivery, path, err)
	}
	return content, nil
}
