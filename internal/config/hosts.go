package config

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"
)

// Host identifies a supported local assistant in project configuration.
type Host string

// Supported local artifact hosts.
const (
	HostClaudeCode Host = "claude-code"
	HostCodex      Host = "codex"
)

// HostSource identifies the authority that selected the artifact hosts.
type HostSource string

// Host selection sources reported by lifecycle commands.
const (
	HostsDetected   HostSource = "detected"
	HostsConfigured HostSource = "configured"
)

// HostSelection is the effective host set, in stable Claude/Codex order.
// An empty successful selection has an empty slice, not a null list.
type HostSelection struct {
	Hosts  []Host     `json:"hosts"`
	Source HostSource `json:"source"`
}

// ErrInvalidHost identifies an unsupported configured host.
var ErrInvalidHost = errors.New("unsupported host")

func (c *Config) validateHosts() error {
	if c == nil || c.Hosts == nil {
		return nil
	}
	for _, name := range *c.Hosts {
		switch Host(name) {
		case HostClaudeCode, HostCodex:
		default:
			return fmt.Errorf("%w %q in aiwf.yaml hosts; use claude-code or codex, [] for none, or omit hosts for PATH detection", ErrInvalidHost, name)
		}
	}
	return nil
}

// ResolveHosts selects the explicit configured set, or detects executables on
// PATH when hosts is unset. It never launches a host or persists detected state.
// A nil config has the same detection behavior as an unset hosts field.
func (c *Config) ResolveHosts(ctx context.Context) (HostSelection, error) {
	if err := ctx.Err(); err != nil {
		return HostSelection{}, fmt.Errorf("resolving artifact hosts: %w", err)
	}
	if err := c.validateHosts(); err != nil {
		return HostSelection{}, err
	}
	selection := HostSelection{Hosts: []Host{}, Source: HostsDetected}
	if c != nil && c.Hosts != nil {
		selection.Source = HostsConfigured
	}
	for _, candidate := range []struct {
		host    Host
		command string
	}{{HostClaudeCode, "claude"}, {HostCodex, "codex"}} {
		if selection.Source == HostsConfigured {
			if slices.Contains(*c.Hosts, string(candidate.host)) {
				selection.Hosts = append(selection.Hosts, candidate.host)
			}
		} else if _, err := exec.LookPath(candidate.command); err == nil {
			selection.Hosts = append(selection.Hosts, candidate.host)
		}
	}
	return selection, nil
}
