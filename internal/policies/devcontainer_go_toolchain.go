package policies

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

const goWorkflowPath = ".github/workflows/go.yml"

// goVersionRe matches go.yml's workflow-level `GO_VERSION: "X.Y.Z"` entry,
// quoted or bare.
var goVersionRe = regexp.MustCompile(`(?m)^\s*GO_VERSION:\s*"?(\d+\.\d+\.\d+)"?\s*$`)

// PolicyDevcontainerGoToolchain asserts that the development container runs
// the Go toolchain CI runs: containerEnv.GOTOOLCHAIN in
// .devcontainer/devcontainer.json must be exactly "go" followed by the
// GO_VERSION pinned in .github/workflows/go.yml. The kit's image carries the
// latest Go; without the pin, code that builds and lints locally can fail in
// CI on a standard library or toolchain difference.
func PolicyDevcontainerGoToolchain(root string) ([]Violation, error) {
	cfg, problem := readDevcontainerConfig(root)
	if problem != "" {
		return []Violation{{Policy: "devcontainer-go-toolchain", File: devcontainerConfigPath, Detail: problem}}, nil
	}
	raw, err := os.ReadFile(filepath.Join(root, goWorkflowPath))
	if err != nil {
		return []Violation{{Policy: "devcontainer-go-toolchain", File: goWorkflowPath, Detail: fmt.Sprintf("missing or unreadable: %v", err)}}, nil
	}
	m := goVersionRe.FindSubmatch(raw)
	if m == nil {
		return []Violation{{Policy: "devcontainer-go-toolchain", File: goWorkflowPath, Detail: `no exact GO_VERSION: "X.Y.Z" pin to compare the container's toolchain with`}}, nil
	}
	if want, got := "go"+string(m[1]), cfg.ContainerEnv["GOTOOLCHAIN"]; got != want {
		return []Violation{{Policy: "devcontainer-go-toolchain", File: devcontainerConfigPath, Detail: fmt.Sprintf(
			"containerEnv.GOTOOLCHAIN = %q, want %q to match %s's GO_VERSION", got, want, goWorkflowPath)}}, nil
	}
	return nil, nil
}
