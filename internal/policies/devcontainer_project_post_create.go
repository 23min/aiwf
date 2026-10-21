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
// needs: installs golangci-lint and govulncheck at their pins, and gofumpt;
// builds aiwf from this checkout and materializes its framework files with
// stdin from /dev/null; installs the kernel pre-commit and pre-push chain; and
// names AIWF_DEVCONTAINER_E2E (default false) and installs Playwright.
// Its golangci-lint and govulncheck pins must match .github/workflows/go.yml,
// which CI treats as the source of truth. Only commands count: everything from
// a line's first # on is read as a comment and satisfies nothing. What the
// kit's own scripts do is the kit's to test.
func PolicyDevcontainerProjectPostCreate(root string) ([]Violation, error) {
	raw, err := os.ReadFile(filepath.Join(root, projectPostCreatePath))
	if err != nil {
		return []Violation{{Policy: "devcontainer-project-post-create", File: projectPostCreatePath, Detail: fmt.Sprintf("missing or unreadable: %v", err)}}, nil
	}
	content := string(raw)
	var commands strings.Builder
	for _, line := range strings.Split(content, "\n") {
		command, _, _ := strings.Cut(line, "#")
		commands.WriteString(command + "\n")
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
		{"aiwf built from this checkout", []string{"go install ./cmd/aiwf"}},
		{"aiwf init with stdin from /dev/null", []string{"aiwf init --no-prompt </dev/null"}},
		{"kernel pre-commit and pre-push chain", []string{"make install-hooks"}},
		{"AIWF_DEVCONTAINER_E2E (default false) named and Playwright installed", []string{"AIWF_DEVCONTAINER_E2E:-false", "playwright install chromium"}},
	}
	for _, c := range checks {
		for _, n := range c.needles {
			if !strings.Contains(commands.String(), n) {
				report(fmt.Sprintf("%s: %q not found", c.name, n))
				break
			}
		}
	}

	ciWorkflow, ciErr := os.ReadFile(filepath.Join(root, goWorkflowPath))
	pins := []struct {
		tool string
		here *regexp.Regexp // the pin in the project hook; each pin is read only as a whole value
		inCI *regexp.Regexp // the same tool's pin in go.yml; golangci-lint's is the action's own version:, two lines below it
	}{
		{"golangci-lint", regexp.MustCompile(`GOLANGCI_LINT_VERSION="?(v\d+\.\d+\.\d+)"?[ \t]*(?:\n|$)`), regexp.MustCompile(`golangci/golangci-lint-action@[^\n]*\n[^\n]*\n\s*version:\s*"?(v\d+\.\d+\.\d+)"?[ \t]*(?:\n|$)`)},
		{"govulncheck", regexp.MustCompile(`GOVULNCHECK_VERSION="?(v\d+\.\d+\.\d+)"?[ \t]*(?:\n|$)`), regexp.MustCompile(`(?m)^[^#\n]*govulncheck@(v\d+\.\d+\.\d+)(?:["'\s]|$)`)},
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
