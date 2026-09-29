package policies

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const devcontainerConfigPath = ".devcontainer/devcontainer.json"

// devcontainerConfig is the part of .devcontainer/devcontainer.json the
// devcontainer policies read.
type devcontainerConfig struct {
	WorkspaceMount string            `json:"workspaceMount"`
	ContainerEnv   map[string]string `json:"containerEnv"`
	// text is the file with its // comment lines dropped: what was decoded.
	text string
}

// readDevcontainerConfig decodes .devcontainer/devcontainer.json. The kit
// generates it with whole-line // comments, which JSON does not allow, so
// those lines are dropped before decoding; the kit writes no trailing
// comments. A read or decode failure comes back as the detail the calling
// policy reports; it is empty when the file decoded.
func readDevcontainerConfig(root string) (cfg devcontainerConfig, problem string) {
	raw, err := os.ReadFile(filepath.Join(root, devcontainerConfigPath))
	if err != nil {
		return cfg, fmt.Sprintf("missing or unreadable: %v", err)
	}
	var kept []string
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "//") {
			kept = append(kept, line)
		}
	}
	cfg.text = strings.Join(kept, "\n")
	if err := json.Unmarshal([]byte(cfg.text), &cfg); err != nil {
		return cfg, fmt.Sprintf("not valid JSON once // comment lines are dropped: %v", err)
	}
	return cfg, ""
}

// mountOption returns the value of key in a devcontainer mount string of the
// form "source=...,target=...,type=...", or "" when the key is absent.
func mountOption(mount, key string) string {
	for _, part := range strings.Split(mount, ",") {
		if value, ok := strings.CutPrefix(strings.TrimSpace(part), key+"="); ok {
			return value
		}
	}
	return ""
}

// PolicyDevcontainerWorkspaceMount asserts that aiwf's development container
// mounts the checkout itself at /workspaces/aiwf and that nothing in
// devcontainer.json names the checkout's parent, ${localWorkspaceFolder}/..,
// whatever the mount's spelling or route (mounts, runArgs). Binding the parent
// exposes every sibling repository and any instruction file sitting beside
// the clone (G-0524); a sibling the container needs is listed explicitly
// through the kit's siblings answers.
func PolicyDevcontainerWorkspaceMount(root string) ([]Violation, error) {
	cfg, problem := readDevcontainerConfig(root)
	if problem != "" {
		return []Violation{{Policy: "devcontainer-workspace-mount", File: devcontainerConfigPath, Detail: problem}}, nil
	}
	var vs []Violation
	const wantSource, wantTarget = "${localWorkspaceFolder}", "/workspaces/aiwf"
	if source, target := mountOption(cfg.WorkspaceMount, "source"), mountOption(cfg.WorkspaceMount, "target"); source != wantSource || target != wantTarget {
		vs = append(vs, Violation{Policy: "devcontainer-workspace-mount", File: devcontainerConfigPath, Detail: fmt.Sprintf(
			"workspaceMount binds %q at %q, want %q at %q: the container mounts the checkout, not its parent", source, target, wantSource, wantTarget)})
	}
	if strings.Contains(cfg.text, wantSource+"/..") {
		vs = append(vs, Violation{Policy: "devcontainer-workspace-mount", File: devcontainerConfigPath, Detail: fmt.Sprintf(
			"names %s/.., the checkout's parent; list a needed sibling through the kit's siblings answers instead", wantSource)})
	}
	return vs, nil
}
