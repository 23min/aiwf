package policies

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const projectPostCreatePath = ".devcontainer/project/post-create.sh"

// PolicyDevcontainerProjectPostCreate asserts that aiwf's project hook,
// .devcontainer/project/post-create.sh, does what only aiwf's container
// needs: installs its pinned Go tools, builds aiwf from this checkout and
// materializes its framework files with stdin from /dev/null, installs the
// kernel pre-commit chain, and gates Playwright behind AIWF_DEVCONTAINER_E2E.
// Its golangci-lint pin must match .github/workflows/go.yml, which CI treats
// as the source of truth. Only commands count: a line whose first non-blank
// character is # is a comment and satisfies nothing. What the kit's own
// scripts do is the kit's to test.
func PolicyDevcontainerProjectPostCreate(root string) ([]Violation, error) {
	raw, err := os.ReadFile(filepath.Join(root, projectPostCreatePath))
	if err != nil {
		return []Violation{{Policy: "devcontainer-project-post-create", File: projectPostCreatePath, Detail: fmt.Sprintf("missing or unreadable: %v", err)}}, nil
	}
	content := string(raw)
	var commands strings.Builder
	for _, line := range strings.Split(content, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "#") {
			commands.WriteString(line + "\n")
		}
	}
	var vs []Violation
	report := func(detail string) {
		vs = append(vs, Violation{Policy: "devcontainer-project-post-create", File: projectPostCreatePath, Detail: detail})
	}

	if first, _, _ := strings.Cut(content, "\n"); strings.TrimSpace(first) != "#!/usr/bin/env bash" || !strings.Contains(content, "set -euo pipefail") {
		report("must start with `#!/usr/bin/env bash` and set `set -euo pipefail`")
	}

	checks := []struct {
		name    string
		needles []string // every one must appear
	}{
		{"golangci-lint, installed once", []string{"command -v golangci-lint"}},
		{"gofumpt, installed once", []string{"command -v gofumpt"}},
		{"govulncheck, installed once", []string{"command -v govulncheck"}},
		{"aiwf built from this checkout", []string{"go install ./cmd/aiwf"}},
		{"aiwf init with stdin from /dev/null", []string{"aiwf init --no-prompt </dev/null"}},
		{"kernel pre-commit chain", []string{"make install-hooks"}},
		{"Playwright behind AIWF_DEVCONTAINER_E2E (default false)", []string{"AIWF_DEVCONTAINER_E2E:-false", "playwright install chromium"}},
	}
	for _, c := range checks {
		for _, n := range c.needles {
			if !strings.Contains(commands.String(), n) {
				report(fmt.Sprintf("%s: %q not found", c.name, n))
				break
			}
		}
	}

	m := regexp.MustCompile(`GOLANGCI_LINT_VERSION="?(v\d+\.\d+\.\d+)"?`).FindStringSubmatch(commands.String())
	if m == nil {
		report("no `GOLANGCI_LINT_VERSION=\"vX.Y.Z\"` pin to compare with CI")
		return vs, nil
	}
	switch ciVer, ciErr := extractGolangciVersionFromCI(root); {
	case ciErr != nil:
		report(fmt.Sprintf("can't read golangci-lint's version from %s: %v", goWorkflowPath, ciErr))
	case ciVer != m[1]:
		report(fmt.Sprintf("golangci-lint pinned at %s here but %s in %s; CI is the source of truth", m[1], ciVer, goWorkflowPath))
	}
	return vs, nil
}

// extractGolangciVersionFromCI reads golangci-lint's version from
// .github/workflows/go.yml: the first `version: vX.Y.Z` line. go.yml's Go
// versions carry no `v` prefix, so requiring one singles out the
// golangci-lint action's pin.
func extractGolangciVersionFromCI(root string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(root, goWorkflowPath))
	if err != nil {
		return "", err
	}
	if m := regexp.MustCompile(`(?m)^\s*version:\s*"?(v\d+\.\d+\.\d+)"?\s*$`).FindSubmatch(raw); m != nil {
		return string(m[1]), nil
	}
	return "", fmt.Errorf("no `version: vX.Y.Z` line in %s", goWorkflowPath)
}
