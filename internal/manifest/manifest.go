// Package manifest defines the import manifest format consumed by
// `aiwf import`. The manifest is a declarative list of entities to
// materialize into a consumer tree; see docs/pocv3/migration/import-format.md
// for the public contract.
//
// This package owns the manifest's data model, parser (YAML and JSON),
// and structural validation (shape, not entity correctness). Entity
// correctness is the import verb's job, evaluated against the
// projected tree by `aiwf check` semantics.
package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/23min/ai-workflow-v2/internal/entity"
)

// AutoID is the literal id value that requests auto-allocation. Used
// in manifest entries when the producer does not pin an id.
const AutoID = "auto"

// Commit modes.
const (
	CommitSingle    = "single"
	CommitPerEntity = "per-entity"
)

// supportedVersion is the only manifest schema version this binary
// understands. Bumping the version is a kernel-level change.
const supportedVersion = 1

// Manifest is the parsed import manifest. Field tags match both YAML
// and JSON because the same struct serves both lexers.
type Manifest struct {
	Version  int        `yaml:"version" json:"version"`
	Actor    string     `yaml:"actor,omitempty" json:"actor,omitempty"`
	Commit   CommitSpec `yaml:"commit,omitempty" json:"commit,omitempty"`
	Entities []Entry    `yaml:"entities" json:"entities"`
}

// CommitSpec controls how the import is committed. Mode defaults to
// single; Message overrides the default subject.
type CommitSpec struct {
	Mode    string `yaml:"mode,omitempty" json:"mode,omitempty"`
	Message string `yaml:"message,omitempty" json:"message,omitempty"`
}

// Entry is one entity to materialize. Kind, ID, and Frontmatter are
// required. Body is verbatim markdown; aiwf does not interpret it.
type Entry struct {
	Kind        string         `yaml:"kind" json:"kind"`
	ID          string         `yaml:"id" json:"id"`
	Frontmatter map[string]any `yaml:"frontmatter" json:"frontmatter"`
	Body        string         `yaml:"body,omitempty" json:"body,omitempty"`
}

// IsAuto reports whether the entry requests auto-allocation.
func (e *Entry) IsAuto() bool { return e.ID == AutoID }

// Parse reads a manifest from raw bytes. Format is "yaml" or "json".
// Returns a structurally validated Manifest or an error describing
// the first shape violation.
func Parse(data []byte, format string) (*Manifest, error) {
	var m Manifest
	switch format {
	case "yaml", "yml":
		if err := yaml.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("parsing yaml: %w", err)
		}
	case "json":
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("parsing json: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported format %q (want yaml or json)", format)
	}
	if err := Validate(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// ParseFile reads a manifest from disk. Format is detected from the
// extension: .yaml/.yml → yaml, .json → json. Other extensions error.
func ParseFile(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest %s: %w", path, err)
	}
	ext := strings.ToLower(filepath.Ext(path))
	var format string
	switch ext {
	case ".yaml", ".yml":
		format = "yaml"
	case ".json":
		format = "json"
	default:
		return nil, fmt.Errorf("manifest %s: unsupported extension %q (want .yaml, .yml, or .json)", path, ext)
	}
	return Parse(data, format)
}

// Validate performs structural checks on a parsed manifest:
//
//   - version is the supported value
//   - commit.mode, when set, is "single" or "per-entity"
//   - every entry has a known kind, an id (explicit or "auto") that
//     matches the kind's regex when explicit, and a non-nil
//     frontmatter map
//
// Validate does not check entity correctness (required fields per
// kind, ref resolution, status legality). Those run against the
// projected tree at import time.
func Validate(m *Manifest) error {
	if m.Version == 0 {
		return errors.New("manifest: missing required field `version`")
	}
	if m.Version != supportedVersion {
		return fmt.Errorf("manifest: version %d is not supported (this binary supports version %d)", m.Version, supportedVersion)
	}
	switch m.Commit.Mode {
	case "", CommitSingle, CommitPerEntity:
		// ok
	default:
		return fmt.Errorf("manifest: commit.mode %q must be %q or %q", m.Commit.Mode, CommitSingle, CommitPerEntity)
	}
	for i, e := range m.Entities {
		if err := validateEntry(&e, i); err != nil {
			return err
		}
	}
	return nil
}

func validateEntry(e *Entry, idx int) error {
	loc := fmt.Sprintf("manifest entry %d", idx)
	if e.Kind == "" {
		return fmt.Errorf("%s: missing required field `kind`", loc)
	}
	k := entity.Kind(e.Kind)
	if !isKnownKind(k) {
		return fmt.Errorf("%s: unknown kind %q (want one of: epic, milestone, adr, gap, decision, contract)", loc, e.Kind)
	}
	if e.ID == "" {
		return fmt.Errorf("%s (%s): missing required field `id` (use %q to allocate)", loc, k, AutoID)
	}
	if !e.IsAuto() {
		if err := entity.ValidateID(k, e.ID); err != nil {
			return fmt.Errorf("%s: %w", loc, err)
		}
	}
	if e.Frontmatter == nil {
		return fmt.Errorf("%s (%s/%s): missing required field `frontmatter`", loc, k, e.ID)
	}
	return nil
}

// isKnownKind reports whether k is one of the six aiwf entity kinds.
func isKnownKind(k entity.Kind) bool {
	for _, known := range entity.AllKinds() {
		if known == k {
			return true
		}
	}
	return false
}

// EffectiveCommitMode returns the commit mode to use, defaulting to
// single when unset.
func (m *Manifest) EffectiveCommitMode() string {
	if m.Commit.Mode == "" {
		return CommitSingle
	}
	return m.Commit.Mode
}
