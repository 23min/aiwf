package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/23min/aiwf/internal/pathutil"
)

// hookScriptMode is the permission bits a materialized hook script is
// written with — executable, mirroring the existing hand-authored
// .claude/hooks/validate-agent-isolation.sh (G-0099).
const hookScriptMode = 0o755

// MaterializeHooks syncs each hook's script to disk under
// target.HooksDir against its consent decision (ADR-0032): a hook
// decided true is written (or overwritten) with its registry content;
// a hook decided false is removed if present. A hook absent from
// decisions — undecided — is left untouched; the consent gate
// (`aiwf init`/`aiwf update`) runs before this function, so an
// undecided hook here reflects a caller that hasn't gated yet, not a
// state this function resolves on its own.
//
// Unlike Materialize's skills/agents/templates (always-on, cleaned up
// via an ownership manifest across renames), a hook's own registry
// name is its identity: this function only ever touches paths named
// in hooks, so a foreign or user-authored file is never at risk.
//
// A target with an empty HooksDir has no hooks concept — a no-op,
// mirroring materializeTo's identical AgentsDir == "" convention.
func MaterializeHooks(root string, target Target, hooks []HookDef, decisions map[string]bool) error {
	if target.HooksDir == "" {
		return nil
	}
	dir := filepath.Join(root, target.HooksDir)
	for _, h := range hooks {
		enabled, decided := decisions[h.Name]
		if !decided {
			continue
		}
		path := filepath.Join(dir, h.Name)
		if !enabled {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("removing %s: %w", path, err)
			}
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", target.HooksDir, err)
		}
		if err := pathutil.AtomicWriteFile(path, h.Content, hookScriptMode); err != nil {
			return fmt.Errorf("writing %s: %w", path, err) //coverage:ignore AtomicWriteFile fails only on filesystem faults; tempdir-based tests can't reproduce (mirrors WireHookSettings's identical, equally-untested shape)
		}
	}
	return nil
}

// HookDriftReport is ADR-0032's three doctor-visible hook-registry
// drift classes, each a sorted list of hook names.
type HookDriftReport struct {
	// Undecided lists registry hooks with no aiwf.yaml decision yet —
	// a newer aiwf shipped a hook the consumer hasn't been gated on.
	Undecided []string

	// MaterializedNotWired lists hooks decided true and present on
	// disk under target.HooksDir, but not wired into settings.json.
	MaterializedNotWired []string

	// WiredButStale lists hooks wired into settings.json despite no
	// longer being authorized to be — decided false, or decided true
	// but the script is missing from disk.
	WiredButStale []string
}

// HookDrift classifies every hook in the registry against its
// aiwf.yaml decision, its presence under target.HooksDir, and whether
// its command is wired into the settings file at settingsPath. A hook
// that is exactly as its decision demands — enabled/materialized/wired
// all true, or disabled/absent/unwired all true — appears in none of
// the three lists.
//
// A target with an empty HooksDir has no hooks concept — an empty
// report, mirroring MaterializeHooks's identical no-op convention.
func HookDrift(root string, target Target, hooks []HookDef, decisions map[string]bool, settingsPath string) (HookDriftReport, error) {
	if target.HooksDir == "" {
		return HookDriftReport{}, nil
	}
	var report HookDriftReport
	for _, h := range hooks {
		enabled, decided := decisions[h.Name]
		if !decided {
			report.Undecided = append(report.Undecided, h.Name)
			continue
		}

		_, statErr := os.Stat(filepath.Join(root, target.HooksDir, h.Name))
		materialized := statErr == nil

		wired, err := HookCommandWired(settingsPath, h.Command(target))
		if err != nil {
			return HookDriftReport{}, err
		}

		switch {
		case enabled && materialized && wired:
			// Fully synced toward "on" — in none of the three lists.
		case enabled:
			// Decided true but not yet fully applied — missing the
			// script, the settings entry, or both. One bucket: the
			// remedy (`aiwf update`) is the same regardless of which
			// half is missing.
			report.MaterializedNotWired = append(report.MaterializedNotWired, h.Name)
		case wired || materialized:
			// Decided false, but something aiwf should have removed
			// is still present — the settings entry, the script, or
			// both.
			report.WiredButStale = append(report.WiredButStale, h.Name)
			// else: decided false and fully absent — fully synced
			// toward "off", in none of the three lists.
		}
	}
	sort.Strings(report.Undecided)
	sort.Strings(report.MaterializedNotWired)
	sort.Strings(report.WiredButStale)
	return report, nil
}
