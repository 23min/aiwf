// Package config loads and writes the consumer repo's `aiwf.yaml`.
//
// The file is small and deliberately so — see
// docs/pocv3/design/design-decisions.md §"aiwf.yaml config". The fields are:
//
//	aiwf_version: 0.1.0       # required; engine version the repo expects
//	actor: human/peter        # required; default for the aiwf-actor: trailer
//	hosts: [claude-code]      # optional; PoC default and only supported value
//	status_md:                # optional; opt-out for the STATUS.md auto-update
//	  auto_update: false      # default true — see StatusMdAutoUpdate
//
// Validation rules:
//   - actor must match `^[^\s/]+/[^\s/]+$` (single '/', no whitespace,
//     neither side empty).
//   - aiwf_version must be a non-empty string (no semver enforcement
//     at this stage; doctor warns on mismatch with binary version).
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

// FileName is the canonical filename at the consumer repo root.
const FileName = "aiwf.yaml"

// ErrNotFound reports that aiwf.yaml does not exist in the queried
// directory. Callers (notably resolveActor) handle this gracefully,
// since the file is optional pre-`aiwf init`.
var ErrNotFound = errors.New("aiwf.yaml not found")

// ActorPattern enforces the actor format documented in
// docs/pocv3/design/design-decisions.md: `<role>/<identifier>`, exactly one '/',
// no whitespace, neither side empty.
var ActorPattern = regexp.MustCompile(`^[^\s/]+/[^\s/]+$`)

// Config is the in-memory shape of aiwf.yaml. Hosts is omitted when
// the on-disk file leaves it absent (which is the typical case).
//
// StatusMd is the opt-out surface for the pre-commit hook that keeps
// `STATUS.md` in sync with the entity tree. Default behavior (block
// absent, or block present with `auto_update` absent) is on; an
// explicit `auto_update: false` opts out. See StatusMdAutoUpdate.
type Config struct {
	AiwfVersion string   `yaml:"aiwf_version"`
	Actor       string   `yaml:"actor"`
	Hosts       []string `yaml:"hosts,omitempty"`
	StatusMd    StatusMd `yaml:"status_md,omitempty"`
}

// StatusMd carries the opt-out for the pre-commit hook that
// regenerates `STATUS.md`. AutoUpdate is a tristate via *bool:
// nil means "not specified, take the default (true)", &false is an
// explicit opt-out, &true is an explicit opt-in. Use the getter
// Config.StatusMdAutoUpdate rather than reading the pointer directly
// so callers don't have to repeat the default.
type StatusMd struct {
	AutoUpdate *bool `yaml:"auto_update,omitempty"`
}

// StatusMdAutoUpdate returns whether the consumer wants the
// pre-commit hook installed and `STATUS.md` regenerated on every
// commit. Default true: the framework's opt-out, not opt-in. The
// committed `STATUS.md` is the user's content once tracked; flipping
// the flag controls whether the *hook* is installed, not whether
// the file is deleted.
func (c *Config) StatusMdAutoUpdate() bool {
	if c.StatusMd.AutoUpdate == nil {
		return true
	}
	return *c.StatusMd.AutoUpdate
}

// Load reads aiwf.yaml from root. Returns ErrNotFound when the file is
// absent so callers can distinguish "missing config" (acceptable
// pre-init) from "malformed config" (always an error).
func Load(root string) (*Config, error) {
	path := filepath.Join(root, FileName)
	bytes, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", FileName, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(bytes, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", FileName, err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", FileName, err)
	}
	return &cfg, nil
}

// Validate enforces the documented constraints. Called by Load and
// expected to be called by Write before serialization.
func (c *Config) Validate() error {
	if c.AiwfVersion == "" {
		return errors.New("aiwf_version is required")
	}
	if c.Actor == "" {
		return errors.New("actor is required")
	}
	if !ActorPattern.MatchString(c.Actor) {
		return fmt.Errorf("actor %q must match <role>/<identifier> (single '/', no whitespace)", c.Actor)
	}
	return nil
}

// Write marshals cfg to root/aiwf.yaml. Refuses to overwrite an
// existing file — callers (notably `aiwf init`) decide what to do
// when one is already there.
func Write(root string, cfg *Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	path := filepath.Join(root, FileName)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", FileName)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("statting %s: %w", FileName, err)
	}
	bytes, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling %s: %w", FileName, err)
	}
	if err := os.WriteFile(path, bytes, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", FileName, err)
	}
	return nil
}
