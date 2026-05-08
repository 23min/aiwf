// Package pluginstate is a read-only view of Claude Code's
// installed-plugin index (`~/.claude/plugins/installed_plugins.json`).
// It exists so `aiwf doctor` can answer the question "is plugin X
// installed for this consumer's project scope?" without re-deriving
// the JSON shape at every call site.
//
// The package is deliberately narrow: it loads the index, exposes a
// typed match function, and treats a missing file as an empty index
// (the M-070 spec's "Claude Code never run on this machine" case).
// Other fields on the on-disk JSON (installPath, version,
// installedAt, lastUpdated, gitCommitSha) are ignored — the matcher
// only needs scope and projectPath.
package pluginstate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Index is the parsed shape of installed_plugins.json. The map key
// is `<name>@<marketplace>` (the same string a user types into
// `claude /plugin install`); the value is the list of scope entries
// recorded for that plugin.
type Index struct {
	Plugins map[string][]InstallEntry `json:"plugins"`
}

// InstallEntry captures the scope-level fields the matcher consumes.
// Only Scope and ProjectPath are read; other fields on the on-disk
// JSON are silently dropped on unmarshal.
//
// Scope is one of "project", "user". For "project" entries
// ProjectPath is the absolute path of the consumer repo the install
// is bound to; for "user" entries it is empty.
type InstallEntry struct {
	Scope       string `json:"scope"`
	ProjectPath string `json:"projectPath,omitempty"`
}

// installedPluginsRelPath is the path under the user's home directory
// where Claude Code stores the canonical plugin index.
const installedPluginsRelPath = ".claude/plugins/installed_plugins.json"

// Load reads installed_plugins.json from `<home>/.claude/plugins/`
// and returns a typed view. A missing file is treated as an empty
// index (no error) — see package doc. Read errors and JSON parse
// errors are returned to the caller so a corrupted state does not
// silently degrade to "no plugins."
func Load(home string) (*Index, error) {
	path := filepath.Join(home, installedPluginsRelPath)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &Index{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if idx.Plugins == nil {
		idx.Plugins = map[string][]InstallEntry{}
	}
	return &idx, nil
}

// HasProjectScope reports whether the index contains an installed
// entry for `plugin` (e.g. `aiwf-extensions@ai-workflow-rituals`)
// with scope `project` and a `projectPath` that resolves to the same
// absolute path as `projectRoot`.
//
// Both paths are normalized via filepath.Abs before comparison so an
// already-absolute argument is a no-op and a relative argument
// resolves against the caller's current working directory. Exact
// string equality after normalization — no case folding, no symlink
// resolution. Per AC-2 of M-070.
//
// User-scope installs are intentionally NOT considered a match: the
// kernel's recommendation surface is per-project ("the consumer
// declared this plugin in *this* repo's aiwf.yaml"), so a plugin
// installed only for the user must still warn until the operator
// installs it for the project too.
func (i *Index) HasProjectScope(plugin, projectRoot string) (bool, error) {
	if i == nil {
		return false, nil
	}
	entries, ok := i.Plugins[plugin]
	if !ok {
		return false, nil
	}
	wantAbs, err := filepath.Abs(projectRoot)
	if err != nil {
		return false, fmt.Errorf("normalizing project root %q: %w", projectRoot, err)
	}
	for _, e := range entries {
		if e.Scope != "project" {
			continue
		}
		gotAbs, err := filepath.Abs(e.ProjectPath)
		if err != nil {
			// A malformed projectPath in the index can't satisfy any
			// match; skip it rather than fail the whole lookup.
			continue
		}
		if gotAbs == wantAbs {
			return true, nil
		}
	}
	return false, nil
}
