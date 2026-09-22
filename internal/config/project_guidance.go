package config

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ErrInvalidGuidance identifies inconsistent project guidance configuration.
var ErrInvalidGuidance = errors.New("invalid project guidance")

// GuidanceEnabled reports whether external guidance maintenance is enabled.
// Turning maintenance off preserves installed policy; it does not revoke it.
func (c *Config) GuidanceEnabled() bool {
	return c == nil || c.Guidance.Enabled == nil || *c.Guidance.Enabled
}

// GuidanceSource returns the configured Git source or the default corpus.
func (c *Config) GuidanceSource() string {
	if c == nil || c.Guidance.Source == "" {
		return "https://github.com/23min/engineering-guidance.git"
	}
	return c.Guidance.Source
}

// UnmarshalYAML requires string source and pack ids rather than YAML's scalar
// coercion, which would turn numeric pack ids into silently accepted strings.
func (g *Guidance) UnmarshalYAML(value *yaml.Node) error {
	var fields map[string]yaml.Node
	if err := value.Decode(&fields); err != nil {
		return fmt.Errorf("decoding guidance fields: %w", err)
	}
	type plain Guidance
	var raw plain
	if err := value.Decode(&raw); err != nil {
		return err
	}
	for _, key := range []string{"source", "packs", "ignored"} {
		field, present := fields[key]
		if !present {
			continue
		}
		node := &field
		for node.Kind == yaml.AliasNode {
			node = node.Alias
		}
		switch key {
		case "source":
			if node.Tag != "!!str" && node.Tag != "!!null" {
				return fmt.Errorf("%w: guidance.source must be a Git source string; use a repository URL or local path", ErrInvalidGuidance)
			}
		case "packs", "ignored":
			for _, item := range node.Content {
				for item.Kind == yaml.AliasNode {
					item = item.Alias
				}
				if item.Tag != "!!str" {
					return fmt.Errorf("%w: guidance.%s must contain string pack ids; use catalogue ids", ErrInvalidGuidance, key)
				}
			}
		}
	}
	*g = Guidance(raw)
	return nil
}

func (g Guidance) validate() error {
	if g.Source != "" && strings.TrimSpace(g.Source) == "" {
		return fmt.Errorf("%w: guidance.source is blank; omit it for the default source or supply a Git repository", ErrInvalidGuidance)
	}
	selected := []string(nil)
	if g.Packs != nil {
		selected = *g.Packs
	}
	seen := make(map[string]string)
	for _, group := range []struct {
		key string
		ids []string
	}{{"packs", selected}, {"ignored", g.Ignored}} {
		for _, id := range group.ids {
			if !ValidGuidancePackID(id) {
				return fmt.Errorf("%w: guidance.%s contains invalid pack id %q; use lowercase catalogue ids such as go/cobra", ErrInvalidGuidance, group.key, id)
			}
			if prior, ok := seen[id]; ok {
				return fmt.Errorf("%w: guidance.%s repeats %q from guidance.%s; keep each id once, selected or ignored", ErrInvalidGuidance, group.key, id, prior)
			}
			seen[id] = group.key
		}
	}
	return nil
}

// ValidGuidancePackID checks the shared configuration and catalogue id grammar.
func ValidGuidancePackID(id string) bool {
	return regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*(/[a-z0-9]+(-[a-z0-9]+)*)*$`).MatchString(id)
}
