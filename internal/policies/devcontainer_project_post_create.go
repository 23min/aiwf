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
// needs: repairs the host's core.hooksPath before anything writes hooks
// (hooks-path.sh, tested on its own, fed AIWF_HOST_CHECKOUT from
// devcontainer.json), installs its pinned Go tools, builds aiwf from this checkout and
// materializes its framework files with stdin from /dev/null, installs the
// kernel pre-commit chain, and gates Playwright behind AIWF_DEVCONTAINER_E2E.
// Its golangci-lint and govulncheck pins must match .github/workflows/go.yml,
// which CI treats as the source of truth. Only commands count: a line whose first non-blank
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

	if first, _, _ := strings.Cut(content, "\n"); strings.TrimSpace(first) != "#!/usr/bin/env bash" || !strings.Contains(commands.String(), "set -euo pipefail") {
		report("must start with `#!/usr/bin/env bash` and set `set -euo pipefail`")
	}

	checks := []struct {
		name    string
		needles []string // every one must appear
	}{
		{"golangci-lint, installed once at its pin", []string{"command -v golangci-lint", `"${GOLANGCI_LINT_VERSION}"`}},
		{"gofumpt, installed once", []string{"command -v gofumpt"}},
		{"govulncheck, installed once at its pin", []string{"command -v govulncheck", "govulncheck@${GOVULNCHECK_VERSION}"}},
		{"host core.hooksPath repaired", []string{"bash .devcontainer/project/hooks-path.sh"}},
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

	const repair = "bash .devcontainer/project/hooks-path.sh"
	if at := strings.Index(commands.String(), repair); at >= 0 {
		for _, later := range []string{"aiwf init --no-prompt", "make install-hooks"} {
			if i := strings.Index(commands.String(), later); i >= 0 && i < at {
				report(fmt.Sprintf("%s must run before %s, which writes into the hooks directory it repairs", repair, later))
			}
		}
	}
	if cfg, problem := readDevcontainerConfig(root); problem == "" && cfg.ContainerEnv["AIWF_HOST_CHECKOUT"] != "${localWorkspaceFolder}" {
		vs = append(vs, Violation{Policy: "devcontainer-project-post-create", File: devcontainerConfigPath, Detail: fmt.Sprintf(
			"containerEnv.AIWF_HOST_CHECKOUT = %q, want \"${localWorkspaceFolder}\": hooks-path.sh compares core.hooksPath with it", cfg.ContainerEnv["AIWF_HOST_CHECKOUT"])})
	}

	ciWorkflow, ciErr := os.ReadFile(filepath.Join(root, goWorkflowPath))
	pins := []struct {
		tool string
		here *regexp.Regexp // the pin in the project hook
		inCI *regexp.Regexp // the same tool's pin in go.yml
	}{
		{"golangci-lint", regexp.MustCompile(`GOLANGCI_LINT_VERSION="?(v\d+\.\d+\.\d+)"?`), regexp.MustCompile(`(?m)^\s*version:\s*"?(v\d+\.\d+\.\d+)"?\s*$`)},
		{"govulncheck", regexp.MustCompile(`GOVULNCHECK_VERSION="?(v\d+\.\d+\.\d+)"?`), regexp.MustCompile(`govulncheck@(v\d+\.\d+\.\d+)`)},
	}
	for _, p := range pins {
		here := p.here.FindStringSubmatch(commands.String())
		switch {
		case here == nil:
			report(fmt.Sprintf("no %s version pin to compare with CI", p.tool))
		case ciErr != nil:
			report(fmt.Sprintf("can't read %s: %v", goWorkflowPath, ciErr))
		default:
			ci := p.inCI.FindSubmatch(ciWorkflow)
			switch {
			case ci == nil:
				report(fmt.Sprintf("no %s version pin in %s to compare with", p.tool, goWorkflowPath))
			case string(ci[1]) != here[1]:
				report(fmt.Sprintf("%s pinned at %s here but %s in %s; CI is the source of truth", p.tool, here[1], ci[1], goWorkflowPath))
			}
		}
	}
	return vs, nil
}
