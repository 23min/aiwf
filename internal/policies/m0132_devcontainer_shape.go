package policies

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PolicyM0132DevcontainerShape asserts that .devcontainer/devcontainer.json
// parses as JSON and contains the agreed structural shape per M-0132/AC-1.
//
// The shape captures the decisions recorded in the milestone Approach
// section: features-first composition on the Microsoft Go base image,
// one-level-up workspace mount so sibling repos under ~/Projects/ are
// visible (cross-repo plugin testing per CLAUDE.md), and the three
// /tmp host mounts that back the claude-code#31388 shadow-mount
// workaround.
//
// Pins M-0132/AC-1. A change that breaks any structural field (wrong
// image, missing feature, mount targets inverted, workspaceMount
// collapsing to a single repo) fails this check before it lands.
func PolicyM0132DevcontainerShape(root string) ([]Violation, error) {
	relPath := ".devcontainer/devcontainer.json"
	abs := filepath.Join(root, relPath)
	raw, err := os.ReadFile(abs)
	if err != nil {
		return []Violation{{
			Policy: "m0132-devcontainer-shape",
			File:   relPath,
			Detail: fmt.Sprintf("missing or unreadable: %v", err),
		}}, nil
	}

	var cfg struct {
		Image             string         `json:"image"`
		WorkspaceFolder   string         `json:"workspaceFolder"`
		WorkspaceMount    string         `json:"workspaceMount"`
		RemoteUser        string         `json:"remoteUser"`
		InitializeCommand string         `json:"initializeCommand"`
		Mounts            []string       `json:"mounts"`
		Features          map[string]any `json:"features"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return []Violation{{
			Policy: "m0132-devcontainer-shape",
			File:   relPath,
			Detail: fmt.Sprintf("not valid JSON: %v", err),
		}}, nil
	}

	var vs []Violation
	report := func(detail string) {
		vs = append(vs, Violation{
			Policy: "m0132-devcontainer-shape",
			File:   relPath,
			Detail: detail,
		})
	}

	const wantImage = "mcr.microsoft.com/devcontainers/go:1-1.25-bookworm"
	if cfg.Image != wantImage {
		report(fmt.Sprintf("image = %q, want %q (Microsoft first-party Go base image per M-0132 Approach / Q6)", cfg.Image, wantImage))
	}

	if cfg.RemoteUser != "vscode" {
		report(fmt.Sprintf("remoteUser = %q, want \"vscode\" (matches Liminara/FlowTime convention)", cfg.RemoteUser))
	}

	// workspaceMount must reference ${localWorkspaceFolder}/.. so
	// sibling repos under ~/Projects/ are visible per Q7 / the
	// cross-repo plugin testing pattern in CLAUDE.md.
	if !strings.Contains(cfg.WorkspaceMount, "${localWorkspaceFolder}/..") {
		report(fmt.Sprintf("workspaceMount = %q must reference ${localWorkspaceFolder}/.. (one-level-up; siblings visible per Q7)", cfg.WorkspaceMount))
	}

	// Three host mounts back the shadow-mount workaround. Each mount
	// string is "source=...,target=...,type=...,..." — we check the
	// expected source-path appears with the "source=" prefix and a
	// trailing comma so partial-prefix mounts don't false-positive.
	wantMountSources := []string{
		"/tmp/.claude-mount",
		"/tmp/.claude-plugins-mount",
		"/tmp/.gh-mount",
	}
	found := map[string]bool{}
	for _, m := range cfg.Mounts {
		for _, want := range wantMountSources {
			if strings.Contains(m, "source="+want+",") {
				found[want] = true
			}
		}
	}
	var missingMounts []string
	for _, w := range wantMountSources {
		if !found[w] {
			missingMounts = append(missingMounts, w)
		}
	}
	if len(missingMounts) > 0 {
		sort.Strings(missingMounts)
		report(fmt.Sprintf("mounts missing entries with source= %s (the /tmp symlink dance per Q5 / initialize.sh)", strings.Join(missingMounts, ", ")))
	}

	// Three features must be declared. SHA pins live in
	// devcontainer-lock.json and are checked by AC-2's policy.
	wantFeatures := []string{
		"ghcr.io/devcontainers/features/common-utils:2",
		"ghcr.io/devcontainers/features/github-cli:1",
		"ghcr.io/devcontainers/features/node:1",
	}
	var missingFeatures []string
	for _, w := range wantFeatures {
		if _, ok := cfg.Features[w]; !ok {
			missingFeatures = append(missingFeatures, w)
		}
	}
	if len(missingFeatures) > 0 {
		sort.Strings(missingFeatures)
		report(fmt.Sprintf("features missing entries for: %s (per Q6 / features-first composition)", strings.Join(missingFeatures, ", ")))
	}

	return vs, nil
}
